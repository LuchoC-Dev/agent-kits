package migrate

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
	"github.com/LuchoC-Dev/agent-kits/internal/source"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// harness wires a source, a catalog and a project for one migration test.
type harness struct {
	t       *testing.T
	project string
	input   Input
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	sourceDir := t.TempDir()
	internaltest.WriteNativeSource(t, sourceDir,
		internaltest.Resource{
			Name: "problem-framing", Type: model.TypeSkill, Version: "1.0.0",
			Files: map[string]string{"SKILL.md": "# framing\n"},
		},
		internaltest.Resource{
			Name: "context-builder", Type: model.TypeAgent, Version: "1.0.0",
			Files: map[string]string{"context-builder.md": "# builder\n"},
		},
		internaltest.Resource{
			Name: "context", Type: model.TypeKit, Version: "1.0.0",
			Files: map[string]string{"pack.md": "# pack\n"},
			Dependencies: []model.Dependency{
				internaltest.Dep("problem-framing"),
				internaltest.Dep("context-builder"),
			},
		},
	)

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
	project := t.TempDir()
	return &harness{t: t, project: project, input: Input{
		Project: project, Adapter: runtime, Catalog: cat,
		Now:          func() time.Time { return now },
		NewProjectID: func() (string, error) { return "generated-id", nil },
	}}
}

// writeLegacyProject materialises a workspace produced by the conversational kits-init
// flow: content and a workspace.json, but no lockfile.
func (h *harness) writeLegacyProject() {
	h.t.Helper()
	internaltest.WriteFile(h.t, h.project, ".agents/skills/problem-framing/SKILL.md", "# framing\n")
	internaltest.WriteFile(h.t, h.project, ".agents/agents/context-builder.md", "# builder\n")
	internaltest.WriteFile(h.t, h.project, ".agents/packs/context/pack.md", "# pack\n")
	internaltest.WriteFile(h.t, h.project, workspace.LegacyPath, legacyJSON)
}

func (h *harness) gather() *Plan {
	h.t.Helper()
	plan, err := Gather(h.input)
	if err != nil {
		h.t.Fatalf("Gather returned %v", err)
	}
	return plan
}

func (h *harness) lock() *model.Lock {
	h.t.Helper()
	lock, err := workspace.LoadLock(h.project, h.input.Adapter)
	if err != nil {
		h.t.Fatalf("LoadLock returned %v", err)
	}
	return lock
}

// The end-to-end path of §10: a legacy fixture becomes a v2 lockfile, an exact backup and
// no workspace.json.
func TestApplyMigratesAnInheritedWorkspace(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()

	plan := h.gather()
	if plan.Blocked() || plan.Empty() {
		t.Fatalf("plan = %+v (blockers %+v)", plan.Changes, plan.Blockers)
	}
	// Computing a migration writes nothing.
	if !internaltest.Exists(h.project, workspace.LegacyPath) ||
		internaltest.Exists(h.project, h.input.Adapter.LockPath()) {
		t.Fatal("Gather must not touch the project")
	}

	report, err := Apply(h.project, plan)
	if err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if !report.Changed || report.Origin != OriginLegacy || report.ToSchema != model.LockSchemaVersion {
		t.Errorf("report = %+v", report)
	}
	if len(report.Adopted) != 3 {
		t.Errorf("adopted = %+v", report.Adopted)
	}

	if internaltest.Exists(h.project, workspace.LegacyPath) {
		t.Error("workspace.json survived the migration")
	}
	if got := internaltest.ReadFile(t, h.project, workspace.BackupPath); got != legacyJSON {
		t.Error("the backup is not a byte-for-byte copy of the original")
	}

	lock := h.lock()
	if lock.SchemaVersion != model.LockSchemaVersion || len(lock.Resources) != 3 {
		t.Fatalf("lock = %+v", lock)
	}
	if lock.Project == nil || lock.Project.ID != "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512" {
		t.Errorf("project = %+v", lock.Project)
	}
	if lock.Migration == nil || lock.Migration.Backup != workspace.BackupPath {
		t.Fatalf("migration = %+v", lock.Migration)
	}
	// The value survives; only its indentation follows the lockfile's own formatting.
	var notes struct{ Keep bool }
	if err := json.Unmarshal(lock.Migration.Extra["notes_from_another_tool"], &notes); err != nil || !notes.Keep {
		t.Errorf("an unknown field was dropped: %v (%v)", lock.Migration.Extra, err)
	}
	// The agent named in workspace.json is adopted under its install name, and the record
	// carries the identity the catalog assigns it.
	adopted, ok := lock.FindByName("context-builder")
	if !ok || adopted.ID == "" {
		t.Errorf("context-builder was not adopted: %+v", lock.Resources)
	}
	// The installed content itself is never touched by a migration.
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != "# framing\n" {
		t.Errorf("installed content = %q", got)
	}
}

// Repeating a migration changes nothing.
func TestApplyIsIdempotent(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	if _, err := Apply(h.project, h.gather()); err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	before := internaltest.ReadFile(t, h.project, h.input.Adapter.LockPath())

	second := h.gather()
	if !second.Empty() {
		t.Fatalf("a repeated migration must be empty: %+v", second.Changes)
	}
	report, err := Apply(h.project, second)
	if err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if report.Changed {
		t.Error("a repeated migration must report no change")
	}
	if after := internaltest.ReadFile(t, h.project, h.input.Adapter.LockPath()); after != before {
		t.Error("an empty migration must not rewrite the lockfile")
	}
}

// An interrupted migration — backup written, workspace.json still present — is completed
// rather than restarted.
func TestApplyCompletesAnInterruptedRetirement(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	if _, err := Apply(h.project, h.gather()); err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	// Put the descriptor back exactly as it was, as if the retirement had never run.
	internaltest.WriteFile(t, h.project, workspace.LegacyPath, legacyJSON)
	lockBefore := internaltest.ReadFile(t, h.project, h.input.Adapter.LockPath())

	plan := h.gather()
	if plan.Blocked() || plan.Empty() {
		t.Fatalf("plan = %+v (blockers %+v)", plan.Changes, plan.Blockers)
	}
	if _, err := Apply(h.project, plan); err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if internaltest.Exists(h.project, workspace.LegacyPath) {
		t.Error("the retirement was not completed")
	}
	if after := internaltest.ReadFile(t, h.project, h.input.Adapter.LockPath()); after != lockBefore {
		t.Error("completing a retirement must not rewrite the lockfile")
	}
}

// A failure at any point leaves lockfile, backup and descriptor exactly as they were.
func TestApplyRestoresTheProjectOnFailure(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	plan := h.gather()

	// Fail the last step: removing a non-empty directory cannot succeed, so the operation
	// aborts after the lockfile and the backup have already been written.
	for i := range plan.Changes {
		if plan.Changes[i].Path == workspace.LegacyPath {
			plan.Changes[i].Path = ".agents/skills"
		}
	}

	if _, err := Apply(h.project, plan); err == nil {
		t.Fatal("Apply should have failed")
	}
	if internaltest.Exists(h.project, h.input.Adapter.LockPath()) {
		t.Error("the lockfile was not rolled back")
	}
	if internaltest.Exists(h.project, workspace.BackupPath) {
		t.Error("the backup was not rolled back")
	}
	if got := internaltest.ReadFile(t, h.project, workspace.LegacyPath); got != legacyJSON {
		t.Error("workspace.json was not restored")
	}
	if got := internaltest.ReadFile(t, h.project, ".agents/skills/problem-framing/SKILL.md"); got != "# framing\n" {
		t.Errorf("installed content was touched: %q", got)
	}
}

// A managed file that no longer matches the catalog cannot be recorded truthfully, so the
// migration aborts without writing anything.
func TestGatherBlocksOnADivergentManagedFile(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "locally edited\n")

	plan := h.gather()
	if !plan.Blocked() {
		t.Fatalf("the plan should be blocked: %+v", plan)
	}
	if !codes(plan.Blockers)[errs.CodeLocalDivergence] {
		t.Errorf("blockers = %+v, want local_divergence", plan.Blockers)
	}
	_, err := Apply(h.project, plan)
	if errs.CodeOf(err) != errs.CodeLocalDivergence {
		t.Fatalf("Apply returned %v, want local_divergence", err)
	}
	if internaltest.Exists(h.project, h.input.Adapter.LockPath()) ||
		internaltest.Exists(h.project, workspace.BackupPath) {
		t.Error("a blocked migration must write nothing")
	}
	if !internaltest.Exists(h.project, workspace.LegacyPath) {
		t.Error("a blocked migration must not retire workspace.json")
	}
}

// A resource the workspace declares but no source offers is a contradiction; a stray file
// found while scanning is only reported.
func TestGatherSeparatesDeclaredFromDiscoveredResources(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	internaltest.WriteFile(t, h.project, ".agents/skills/stray/SKILL.md", "not in any source\n")

	plan := h.gather()
	if plan.Blocked() {
		t.Fatalf("a stray directory must not block: %+v", plan.Blockers)
	}
	var reported bool
	for _, warning := range plan.Warnings {
		if warning.Ref == "stray" {
			reported = true
		}
	}
	if !reported {
		t.Errorf("warnings = %+v", plan.Warnings)
	}

	// A declared resource that cannot be identified is a different matter.
	internaltest.WriteFile(t, h.project, workspace.LegacyPath, `{
  "$schema_version": 2,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-06-01T09:30:00Z",
  "runtime": "claude-code",
  "pack": null,
  "skills": [{ "id": "gone-from-every-source", "source": "skills/gone", "installed_at": "2026-05-22T10:00:00Z" }],
  "agents": [],
  "disciplines": [],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": ["skills"]
}`)
	declared := h.gather()
	if !declared.Blocked() {
		t.Fatalf("a declared resource that cannot be identified must block: %+v", declared)
	}
}

// A backup that holds a different state is never overwritten.
func TestGatherRefusesToOverwriteADifferentBackup(t *testing.T) {
	h := newHarness(t)
	h.writeLegacyProject()
	internaltest.WriteFile(t, h.project, workspace.BackupPath, `{"$schema_version": 2, "id": "older"}`)

	plan := h.gather()
	if !codes(plan.Blockers)[errs.CodeIntegrityMismatch] {
		t.Fatalf("blockers = %+v, want integrity_mismatch", plan.Blockers)
	}
	if _, err := Apply(h.project, plan); errs.CodeOf(err) != errs.CodeIntegrityMismatch {
		t.Fatalf("Apply returned %v", err)
	}
	if got := internaltest.ReadFile(t, h.project, workspace.BackupPath); got != `{"$schema_version": 2, "id": "older"}` {
		t.Errorf("the existing backup was overwritten: %q", got)
	}
}

// A project with no state at all is left alone.
func TestGatherOnAProjectWithNoState(t *testing.T) {
	h := newHarness(t)
	plan := h.gather()
	if !plan.Empty() || plan.Origin != OriginNone {
		t.Fatalf("plan = %+v", plan)
	}
	report, err := Apply(h.project, plan)
	if err != nil {
		t.Fatalf("Apply returned %v", err)
	}
	if report.Changed {
		t.Error("nothing to migrate must report no change")
	}
	entries, err := os.ReadDir(h.project)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the project gained files: %+v", entries)
	}
}

// A workspace.json Agent Kits cannot understand stops the migration instead of being
// ignored, because ignoring it would let a later command overwrite it.
func TestGatherRejectsAnInvalidWorkspace(t *testing.T) {
	h := newHarness(t)
	internaltest.WriteFile(t, h.project, workspace.LegacyPath, `{"$schema_version": 99}`)
	if _, err := Gather(h.input); !errs.Is(err, errs.CodeWorkspaceInvalid) {
		t.Fatalf("Gather returned %v, want workspace_invalid", err)
	}
}

// A v1 lockfile is upgraded even when the project never had a workspace.json.
func TestApplyUpgradesALegacyLockfile(t *testing.T) {
	h := newHarness(t)
	internaltest.WriteFile(t, h.project, ".agents/skills/problem-framing/SKILL.md", "# framing\n")
	internaltest.WriteFile(t, h.project, h.input.Adapter.LockPath(), `{
  "schema_version": 1,
  "runtime": "claude-code",
  "generated_at": "2026-05-22T10:00:00Z",
  "resources": [{"id":"problem-framing","type":"skill","source":"public","version":"1.0.0",
    "checksum":"sha256:x","requested":true,
    "files":[{"path":".agents/skills/problem-framing/SKILL.md","checksum":"sha256:y"}]}]
}`)

	plan := h.gather()
	if plan.Origin != OriginLock || plan.FromSchema != model.LockSchemaVersionLegacy {
		t.Fatalf("plan = %+v", plan)
	}
	if _, err := Apply(h.project, plan); err != nil {
		t.Fatalf("Apply returned %v", err)
	}

	raw := internaltest.ReadFile(t, h.project, h.input.Adapter.LockPath())
	var written map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &written); err != nil {
		t.Fatalf("the lockfile is not valid JSON: %v", err)
	}
	if string(written["schema_version"]) != "2" {
		t.Errorf("schema_version = %s", written["schema_version"])
	}
	lock := h.lock()
	if lock.Project == nil || lock.Project.ID != "generated-id" {
		t.Errorf("project = %+v", lock.Project)
	}
	if len(lock.Resources) != 1 || lock.Resources[0].Checksum != "sha256:x" {
		t.Errorf("the upgrade changed the recorded resources: %+v", lock.Resources)
	}
	if internaltest.Exists(h.project, workspace.BackupPath) {
		t.Error("an upgrade with no inherited descriptor needs no backup")
	}
}

// A symlinked descriptor is refused rather than followed.
func TestGatherRefusesASymlinkedWorkspace(t *testing.T) {
	h := newHarness(t)
	target := internaltest.WriteFile(t, t.TempDir(), "elsewhere.json", legacyJSON)
	link := filepath.Join(h.project, filepath.FromSlash(workspace.LegacyPath))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this platform does not allow creating symlinks: %v", err)
	}
	if _, err := Gather(h.input); !errs.Is(err, errs.CodeUnsafePath) {
		t.Fatalf("Gather returned %v, want unsafe_path", err)
	}
}
