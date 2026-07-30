package install

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/plan"
	"github.com/LuchoC-Dev/agent-kits/internal/resolve"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

var fixedTime = time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)

// harness wires a source, a catalog, a project and a store for one test.
type harness struct {
	t         *testing.T
	sourceDir string
	project   string
	catalog   *catalog.Catalog
	adapter   adapter.Adapter
	store     *source.Store
}

func newHarness(t *testing.T, resources ...internaltest.Resource) *harness {
	t.Helper()
	sourceDir := t.TempDir()
	internaltest.WriteNativeSource(t, sourceDir, resources...)

	t.Setenv(source.HomeEnv, t.TempDir())
	store, err := source.Open()
	if err != nil {
		t.Fatalf("source.Open returned %v", err)
	}
	if err := store.Add(source.Source{
		Name: "public", URL: sourceDir,
		Access: model.AccessPublic, Trust: model.TrustTrusted,
	}); err != nil {
		t.Fatalf("Add returned %v", err)
	}
	cat, err := catalog.NewLoader().Load(store)
	if err != nil {
		t.Fatalf("Load returned %v", err)
	}
	runtime, err := adapter.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	return &harness{
		t: t, sourceDir: sourceDir, project: t.TempDir(),
		catalog: cat, adapter: runtime, store: store,
	}
}

func (h *harness) lock() *model.Lock {
	h.t.Helper()
	lock, err := workspace.LoadLock(h.project, h.adapter)
	if err != nil {
		h.t.Fatalf("LoadLock returned %v", err)
	}
	return lock
}

func (h *harness) planInstall(refs ...string) *model.Plan {
	h.t.Helper()
	result, err := resolve.New(h.catalog, h.adapter.Name()).Resolve(refs)
	if err != nil {
		h.t.Fatalf("Resolve returned %v", err)
	}
	planner := plan.New(h.adapter, h.project, h.lock())
	planner.Now = func() time.Time { return fixedTime }
	built, err := planner.Install(result)
	if err != nil {
		h.t.Fatalf("Install plan returned %v", err)
	}
	return built
}

func (h *harness) installer() *Installer {
	installer := New(h.adapter, h.project)
	installer.Now = func() time.Time { return fixedTime }
	return installer
}

func (h *harness) apply(refs ...string) *Report {
	h.t.Helper()
	report, err := h.installer().Apply(h.planInstall(refs...))
	if err != nil {
		h.t.Fatalf("Apply returned %v", err)
	}
	return report
}

func sampleKit() []internaltest.Resource {
	return []internaltest.Resource{
		{
			ID: "problem-framing", Type: model.TypeSkill, Version: "1.0.0",
			Files: map[string]string{"SKILL.md": "# framing\n"},
		},
		{
			ID: "context/context-builder", Type: model.TypeAgent, Version: "1.0.0",
			Files: map[string]string{"context-builder.md": "# builder\n"},
		},
		{
			ID: "context", Type: model.TypeKit, Version: "1.0.0",
			Files: map[string]string{"pack.md": "# pack\n"},
			Dependencies: []model.Dependency{
				internaltest.Dep("problem-framing"),
				internaltest.Dep("context/context-builder"),
			},
		},
	}
}

// An installation writes the resources and the lockfile — and nothing else. workspace.json
// is no longer part of a project's state (D-030).
func TestApplyInstallsFilesAndLock(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	report := h.apply("context")

	if report.Created != 3 {
		t.Errorf("created = %d, want 3", report.Created)
	}
	for _, path := range []string{
		".agents/skills/problem-framing/SKILL.md",
		".agents/agents/context-builder.md",
		".agents/packs/context/pack.md",
		".agents/agent-kits.lock.json",
	} {
		if !internaltest.Exists(h.project, path) {
			t.Errorf("%s was not written", path)
		}
	}
	if internaltest.Exists(h.project, workspace.LegacyPath) {
		t.Error("a normal command must never create workspace.json")
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != "# framing\n" {
		t.Errorf("installed content = %q", got)
	}

	lock := h.lock()
	if lock.SchemaVersion != model.LockSchemaVersion || len(lock.Resources) != 3 ||
		lock.Runtime != "claude-code" {
		t.Fatalf("lock = %+v", lock)
	}
	kit, ok := lock.Find("context")
	if !ok || !kit.Requested || kit.Source != "public" || kit.Checksum == "" {
		t.Errorf("kit record = %+v", kit)
	}
	// The first write assigns the identity the project keeps for its whole life.
	if lock.Project == nil || lock.Project.ID == "" || lock.Project.CreatedAt == "" {
		t.Fatalf("project = %+v", lock.Project)
	}

	// A later operation preserves that identity rather than minting a new one.
	identity := *lock.Project
	h.apply("problem-framing")
	after := h.lock()
	if after.Project == nil || after.Project.ID != identity.ID ||
		after.Project.CreatedAt != identity.CreatedAt {
		t.Errorf("project identity changed: %+v -> %+v", identity, after.Project)
	}
}

// A project that still carries an unmigrated descriptor is never written to.
func TestApplyRefusesAProjectPendingMigration(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	built := h.planInstall("context")
	internaltest.WriteFile(t, h.project, workspace.LegacyPath, `{"$schema_version": 2}`)

	_, err := h.installer().Apply(built)
	if errs.CodeOf(err) != errs.CodeWorkspaceInvalid {
		t.Fatalf("err = %v, want workspace_invalid", err)
	}
	if internaltest.Exists(h.project, ".agents/agent-kits.lock.json") {
		t.Error("a project pending migration was written to")
	}
}

// RF-08: repeating an installation changes nothing.
func TestApplyIsIdempotent(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("context")
	before := internaltest.ReadFile(t, h.project, ".agents/agent-kits.lock.json")

	second := h.planInstall("context")
	if !second.Empty() {
		t.Fatalf("a repeated plan must be empty, got %+v", second.Changes)
	}
	report, err := h.installer().Apply(second)
	if err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if report.Created != 0 || report.Updated != 0 {
		t.Errorf("report = %+v", report)
	}
	if after := internaltest.ReadFile(t, h.project, ".agents/agent-kits.lock.json"); after != before {
		t.Error("an empty plan must not rewrite the lockfile")
	}
}

func TestApplyRefusesBlockedPlan(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "mine\n")

	built := h.planInstall("context")
	if !built.Blocked() {
		t.Fatal("the plan should be blocked")
	}
	_, err := h.installer().Apply(built)
	if errs.CodeOf(err) != errs.CodeLocalDivergence {
		t.Fatalf("err = %v, want local_divergence", err)
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != "mine\n" {
		t.Errorf("the local file was touched: %q", got)
	}
	if internaltest.Exists(h.project, ".agents/agent-kits.lock.json") {
		t.Error("a blocked plan must not write a lockfile")
	}
}

// A failure part-way through must leave the project exactly as it was.
func TestApplyRollsBackOnFailure(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("problem-framing")

	pristineLock := internaltest.ReadFile(t, h.project, ".agents/agent-kits.lock.json")
	pristineSkill := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md")

	// Plan a second resource, then change its source content so apply detects that the
	// plan no longer describes reality.
	built := h.planInstall("context")
	packPath := filepath.Join(h.sourceDir, "kits", "context", "pack.md")
	if err := os.WriteFile(packPath, []byte("# tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := h.installer().Apply(built)
	if errs.CodeOf(err) != errs.CodeIntegrityMismatch {
		t.Fatalf("err = %v, want integrity_mismatch", err)
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/agent-kits.lock.json"); got != pristineLock {
		t.Error("the lockfile was not restored")
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != pristineSkill {
		t.Error("an already installed file was not restored")
	}
	if internaltest.Exists(h.project, ".agents/agents/context-builder.md") {
		t.Error("a file created before the failure was not rolled back")
	}
}

func TestRemoveDeletesFilesAndPrunesDirectories(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("context")

	planner := plan.New(h.adapter, h.project, h.lock())
	planner.Now = func() time.Time { return fixedTime }
	removal, err := planner.Remove([]string{"context"}, h.catalog)
	if err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	if _, err := h.installer().Apply(removal); err != nil {
		t.Fatalf("Apply returned %v", err)
	}

	for _, path := range []string{
		".agents/skills/problem-framing/SKILL.md",
		".agents/skills/problem-framing",
		".agents/agents/context-builder.md",
		".agents/packs/context/pack.md",
	} {
		if internaltest.Exists(h.project, path) {
			t.Errorf("%s survived removal", path)
		}
	}
	if len(h.lock().Resources) != 0 {
		t.Errorf("lock = %+v", h.lock().Resources)
	}
	// The lockfile itself stays, so the project remains a managed workspace.
	if !internaltest.Exists(h.project, ".agents/agent-kits.lock.json") {
		t.Error("the lockfile should survive removing every resource")
	}
}

func TestRemovePreservesUnrelatedProjectFiles(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	internaltest.WriteFile(t, h.project, ".agents/notes.md", "my own notes\n")
	internaltest.WriteFile(t, h.project, "README.md", "project readme\n")
	h.apply("context")

	planner := plan.New(h.adapter, h.project, h.lock())
	removal, err := planner.Remove([]string{"context"}, h.catalog)
	if err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	if _, err := h.installer().Apply(removal); err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/notes.md"); got != "my own notes\n" {
		t.Errorf("a foreign file inside .agents was touched: %q", got)
	}
	if got := internaltest.ReadFile(t, h.project, "README.md"); got != "project readme\n" {
		t.Errorf("a project file was touched: %q", got)
	}
}

func TestDoctorReportsHealthyProject(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("context")

	report, err := Doctor(DoctorInput{
		Project: h.project, Adapter: h.adapter, Store: h.store, Catalog: h.catalog,
	})
	if err != nil {
		t.Fatalf("Doctor returned %v", err)
	}
	if !report.Healthy {
		t.Errorf("problems = %+v", report.Problems)
	}
	if report.Installed != 3 || len(report.Sources) != 1 || !report.Sources[0].Reachable {
		t.Errorf("report = %+v", report)
	}
}

func TestDoctorDetectsMissingAndModifiedFiles(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("context")

	if err := os.Remove(filepath.Join(h.project, filepath.FromSlash(".agents/packs/context/pack.md"))); err != nil {
		t.Fatal(err)
	}
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "edited\n")

	report, err := Doctor(DoctorInput{
		Project: h.project, Adapter: h.adapter, Store: h.store, Catalog: h.catalog,
	})
	if err != nil {
		t.Fatalf("Doctor returned %v", err)
	}
	if report.Healthy {
		t.Fatal("the project is not healthy")
	}
	codes := map[errs.Code]bool{}
	for _, problem := range report.Problems {
		codes[problem.Code] = true
	}
	if !codes[errs.CodeIntegrityMismatch] {
		t.Error("a missing file was not reported")
	}
	if !codes[errs.CodeLocalDivergence] {
		t.Error("a modified file was not reported")
	}
}

func TestDoctorNotesOrphanFilesAndAvailableUpdates(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	h.apply("problem-framing")
	internaltest.WriteFile(t, h.project, ".agents/skills/stray/SKILL.md", "not mine\n")

	report, err := Doctor(DoctorInput{
		Project: h.project, Adapter: h.adapter, Store: h.store, Catalog: h.catalog,
	})
	if err != nil {
		t.Fatalf("Doctor returned %v", err)
	}
	var sawOrphan bool
	for _, note := range report.Notes {
		if note.Path == ".agents/skills/stray/SKILL.md" {
			sawOrphan = true
		}
	}
	if !sawOrphan {
		t.Errorf("notes = %+v", report.Notes)
	}
}
