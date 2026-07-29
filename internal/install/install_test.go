package install

import (
	"encoding/json"
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
	installer := New(h.adapter, h.project, resourceMap(h.catalog))
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

func resourceMap(cat *catalog.Catalog) map[model.ID]*model.Resource {
	out := map[model.ID]*model.Resource{}
	for _, res := range cat.All() {
		out[res.ID] = res
	}
	return out
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

func TestApplyInstallsFilesLockAndDescriptor(t *testing.T) {
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
		".agents/workspace.json",
	} {
		if !internaltest.Exists(h.project, path) {
			t.Errorf("%s was not written", path)
		}
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != "# framing\n" {
		t.Errorf("installed content = %q", got)
	}

	lock := h.lock()
	if len(lock.Resources) != 3 || lock.Runtime != "claude-code" {
		t.Fatalf("lock = %+v", lock)
	}
	kit, ok := lock.Find("context")
	if !ok || !kit.Requested || kit.Source != "public" || kit.Checksum == "" {
		t.Errorf("kit record = %+v", kit)
	}

	descriptor, present, err := workspace.LoadDescriptor(h.project, h.adapter)
	if err != nil || !present {
		t.Fatalf("LoadDescriptor = %v, %v", present, err)
	}
	if descriptor.SchemaVersion != workspace.SchemaVersion || descriptor.Runtime != "claude-code" {
		t.Errorf("descriptor = %+v", descriptor)
	}
	if len(descriptor.Skills) != 1 || descriptor.Skills[0].ID != "problem-framing" {
		t.Errorf("descriptor skills = %+v", descriptor.Skills)
	}
	if len(descriptor.Agents) != 1 || descriptor.Agents[0].Class != 1 {
		t.Errorf("descriptor agents = %+v", descriptor.Agents)
	}
	if descriptor.Pack == nil || descriptor.Pack.Name != "context" {
		t.Errorf("descriptor pack = %+v", descriptor.Pack)
	}
	if !descriptor.Flags.Initialized {
		t.Error("descriptor must be marked as initialised")
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
	// The workspace metadata itself stays, so the project remains a managed workspace.
	if !internaltest.Exists(h.project, ".agents/workspace.json") {
		t.Error("workspace.json should survive removing every resource")
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

// writeLegacyWorkspace simulates a workspace produced by the conversational kits-init
// flow: content and a workspace.json, but no lockfile.
func writeLegacyWorkspace(t *testing.T, h *harness, extra string) {
	t.Helper()
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "# framing\n")
	internaltest.WriteFile(t, h.project, ".agents/agents/context-builder.md", "# builder\n")
	internaltest.WriteFile(t, h.project, ".agents/packs/context/pack.md", "# pack\n")
	internaltest.WriteFile(t, h.project, ".agents/workspace.json", `{
  "$schema_version": 2,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-05-22T10:00:00Z",
  "system_version": "0.1.0",
  "runtime": "claude-code",
  "pack": { "name": "context", "source": "packs/context", "installed_at": "2026-05-22T10:00:00Z" },
  "stack": { "detected": ["go"], "source": "user-input", "confidence": "high" },
  "skills": [{ "id": "problem-framing", "source": "skills/problem-framing", "installed_at": "2026-05-22T10:00:00Z" }],
  "agents": [{ "id": "context-builder", "class": 1, "source": "packs/context/agents", "installed_at": "2026-05-22T10:00:00Z" }],
  "disciplines": [],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": ["agents", "skills"]`+extra+`
}`)
}

// D-022: a workspace created by kits-init can be adopted by the CLI.
func TestImportAdoptsLegacyWorkspace(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	writeLegacyWorkspace(t, h, `,
  "custom_field": {"keep": true}`)

	built, err := Import(ImportInput{
		Project: h.project, Adapter: h.adapter, Catalog: h.catalog,
		Now: func() time.Time { return fixedTime },
	})
	if err != nil {
		t.Fatalf("Import returned %v", err)
	}
	if len(built.Resources) != 3 {
		t.Fatalf("adopted %d resources: %+v", len(built.Resources), built.Resources)
	}
	for _, change := range built.Changes {
		if change.Action != model.ActionAdopt {
			t.Errorf("%s action = %q, want adopt", change.Path, change.Action)
		}
	}

	if _, err := h.installer().Apply(built); err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	lock := h.lock()
	if len(lock.Resources) != 3 {
		t.Fatalf("lock = %+v", lock.Resources)
	}
	// The bare agent name in workspace.json resolves to its qualified canonical id.
	if _, ok := lock.Find("context/context-builder"); !ok {
		t.Errorf("context/context-builder was not adopted: %+v", lock.Resources)
	}

	// Fields Agent Kits does not own must survive the rewrite.
	raw := internaltest.ReadFile(t, h.project, ".agents/workspace.json")
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatalf("workspace.json is not valid JSON: %v", err)
	}
	if _, ok := fields["custom_field"]; !ok {
		t.Error("an unmanaged field was dropped")
	}
	if _, ok := fields["stack"]; !ok {
		t.Error("the stack field was dropped")
	}

	// After adopting, a plan for the same kit is a no-op.
	if next := h.planInstall("context"); !next.Empty() {
		t.Errorf("a plan after import should be empty, got %+v", next.Changes)
	}
}

func TestImportRefusesDivergentFiles(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	writeLegacyWorkspace(t, h, "")
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "locally edited\n")

	built, err := Import(ImportInput{
		Project: h.project, Adapter: h.adapter, Catalog: h.catalog,
		Now: func() time.Time { return fixedTime },
	})
	if err != nil {
		t.Fatalf("Import returned %v", err)
	}
	for _, res := range built.Resources {
		if res.ID == "problem-framing" {
			t.Error("a divergent resource must not be adopted")
		}
	}
	var warned bool
	for _, warning := range built.Warnings {
		if warning.Code == errs.CodeLocalDivergence {
			warned = true
		}
	}
	if !warned {
		t.Errorf("warnings = %+v", built.Warnings)
	}
}

func TestImportRequiresAWorkspace(t *testing.T) {
	h := newHarness(t, sampleKit()...)
	_, err := Import(ImportInput{Project: h.project, Adapter: h.adapter, Catalog: h.catalog})
	if errs.CodeOf(err) != errs.CodeWorkspaceInvalid {
		t.Fatalf("err = %v, want workspace_invalid", err)
	}
}
