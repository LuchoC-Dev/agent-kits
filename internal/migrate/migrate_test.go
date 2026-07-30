package migrate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

var now = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

const lockPath = ".agents/agent-kits.lock.json"

// legacyJSON is a complete inherited workspace.json, including a field no version of
// Agent Kits manages.
const legacyJSON = `{
  "$schema_version": 2,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-06-01T09:30:00Z",
  "system_version": "0.0.9",
  "runtime": "claude-code",
  "pack": { "name": "context", "source": "packs/context", "installed_at": "2026-05-22T10:00:00Z" },
  "stack": { "detected": ["go"], "source": "user-input", "confidence": "high" },
  "skills": [{ "id": "problem-framing", "source": "skills/problem-framing", "installed_at": "2026-05-22T10:00:00Z" }],
  "agents": [],
  "disciplines": ["tdd"],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": ["packs", "skills"],
  "notes_from_another_tool": {"keep": true}
}`

func legacy(t *testing.T, content string) *workspace.Legacy {
	t.Helper()
	descriptor, err := workspace.ParseDescriptor([]byte(content))
	if err != nil {
		t.Fatalf("ParseDescriptor returned %v", err)
	}
	return &workspace.Legacy{Path: workspace.LegacyPath, Raw: []byte(content), Descriptor: descriptor}
}

// base returns a state with no project state at all.
func base() State {
	return State{
		Project:      "/tmp/project",
		Runtime:      "claude-code",
		Lock:         model.NewLock("claude-code"),
		LockPath:     lockPath,
		Now:          now,
		NewProjectID: func() (string, error) { return "generated-id", nil },
	}
}

// lockV2 returns a state whose project already has a current lockfile.
func lockV2() State {
	state := base()
	state.LockSchema = model.LockSchemaVersion
	state.Lock.GeneratedAt = "2026-07-01T00:00:00Z"
	state.Lock.EnsureProject("3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512", "2026-05-22T10:00:00Z")
	state.Lock.Upsert(model.LockResource{
		ID: "problem-framing", Type: model.TypeSkill, Source: "public", Version: "1.0.0",
		Checksum: "sha256:x", Requested: true, InstalledAt: "2026-05-22T10:00:00Z",
	})
	return state
}

// lockV1 returns a state whose project carries a lockfile in the superseded schema. The
// caller has already upgraded it in memory, which is what LoadLockDetail does.
func lockV1() State {
	state := lockV2()
	state.LockSchema = model.LockSchemaVersionLegacy
	state.Lock.Project = nil
	return state
}

func compute(t *testing.T, state State) *Plan {
	t.Helper()
	plan, err := Compute(state)
	if err != nil {
		t.Fatalf("Compute returned %v", err)
	}
	return plan
}

func action(t *testing.T, plan *Plan, path string) model.FileAction {
	t.Helper()
	for _, change := range plan.Changes {
		if change.Path == path {
			return change.Action
		}
	}
	return ""
}

func codes(list []model.Diagnostic) map[errs.Code]bool {
	out := map[errs.Code]bool{}
	for _, item := range list {
		out[item.Code] = true
	}
	return out
}

// The transition matrix of 07-cli-only-transition-plan.md §5, row by row.
func TestComputeCoversTheMigrationMatrix(t *testing.T) {
	backupOf := func(content string) ([]byte, bool) { return []byte(content), true }

	tests := []struct {
		name    string
		state   func() State
		origin  string
		empty   bool
		blocked bool
		lock    model.FileAction
		backup  model.FileAction
		retired model.FileAction
	}{
		{
			name:   "no lock and no workspace: nothing to migrate",
			state:  base,
			origin: OriginNone,
			empty:  true,
		},
		{
			name:   "lock v2 without workspace: idempotent no-op",
			state:  lockV2,
			origin: OriginLock,
			empty:  true,
			lock:   model.ActionUnchanged,
		},
		{
			name:   "lock v1 without workspace: deterministic upgrade",
			state:  lockV1,
			origin: OriginLock,
			lock:   model.ActionUpdate,
		},
		{
			name: "workspace without lock: conservative adoption",
			state: func() State {
				state := base()
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
			origin:  OriginLegacy,
			lock:    model.ActionCreate,
			backup:  model.ActionCreate,
			retired: model.ActionRemove,
		},
		{
			name: "lock v1 and workspace: merge",
			state: func() State {
				state := lockV1()
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
			origin:  OriginLegacy,
			lock:    model.ActionUpdate,
			backup:  model.ActionCreate,
			retired: model.ActionRemove,
		},
		{
			name: "lock v2 and workspace without a migration record: absorb the metadata",
			state: func() State {
				state := lockV2()
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
			origin:  OriginLegacy,
			lock:    model.ActionUpdate,
			backup:  model.ActionCreate,
			retired: model.ActionRemove,
		},
		{
			name: "migrated lock and a backup identical to the workspace: finish the retirement",
			state: func() State {
				state := lockV2()
				state.Legacy = legacy(t, legacyJSON)
				state.Lock.Migration = record(state, state.Legacy.Descriptor)
				state.Lock.Project.Stack = &model.LockStack{
					Detected: []string{"go"}, Source: "user-input", Confidence: "high",
				}
				state.Lock.Project.Disciplines = []string{"tdd"}
				state.Backup, state.BackupPresent = backupOf(legacyJSON)
				return state
			},
			origin:  OriginLegacy,
			lock:    model.ActionUnchanged,
			backup:  model.ActionUnchanged,
			retired: model.ActionRemove,
		},
		{
			name: "a different backup already exists: abort",
			state: func() State {
				state := base()
				state.Legacy = legacy(t, legacyJSON)
				state.Backup, state.BackupPresent = backupOf(`{"$schema_version": 2, "id": "other"}`)
				return state
			},
			origin:  OriginLegacy,
			blocked: true,
			lock:    model.ActionCreate,
			backup:  model.ActionCreate,
			retired: model.ActionRemove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := compute(t, test.state())
			if plan.Origin != test.origin {
				t.Errorf("origin = %q, want %q", plan.Origin, test.origin)
			}
			if plan.Empty() != test.empty {
				t.Errorf("Empty() = %v, want %v (changes %+v)", plan.Empty(), test.empty, plan.Changes)
			}
			if plan.Blocked() != test.blocked {
				t.Errorf("Blocked() = %v, want %v (blockers %+v)", plan.Blocked(), test.blocked, plan.Blockers)
			}
			if got := action(t, plan, lockPath); got != test.lock {
				t.Errorf("lockfile action = %q, want %q", got, test.lock)
			}
			if got := action(t, plan, workspace.BackupPath); got != test.backup {
				t.Errorf("backup action = %q, want %q", got, test.backup)
			}
			if got := action(t, plan, workspace.LegacyPath); got != test.retired {
				t.Errorf("workspace action = %q, want %q", got, test.retired)
			}
			if test.origin != OriginNone && plan.Lock == nil {
				t.Fatal("a migration must propose a lockfile")
			}
			if plan.Lock != nil && plan.Lock.SchemaVersion != model.LockSchemaVersion {
				t.Errorf("the proposed lock is schema %d", plan.Lock.SchemaVersion)
			}
		})
	}
}

func TestComputeAbortsOnADifferentBackupWithIntegrityMismatch(t *testing.T) {
	state := base()
	state.Legacy = legacy(t, legacyJSON)
	state.Backup, state.BackupPresent = []byte(`{"$schema_version": 2, "id": "other"}`), true

	plan := compute(t, state)
	if !codes(plan.Blockers)[errs.CodeIntegrityMismatch] {
		t.Errorf("blockers = %+v, want integrity_mismatch", plan.Blockers)
	}
}

// A migration is lossless: identity, stack, disciplines, history and unknown fields all
// survive in the new schema.
func TestComputePreservesEveryInheritedField(t *testing.T) {
	state := base()
	state.Legacy = legacy(t, legacyJSON)
	state.Adoptions = []Adoption{{
		Ref: "problem-framing",
		Record: &model.LockResource{
			ID: "problem-framing", Type: model.TypeSkill, Source: "public", Version: "1.0.0",
			Checksum: "sha256:x", Requested: false,
		},
	}}

	plan := compute(t, state)
	lock := plan.Lock
	if lock.Project == nil || lock.Project.ID != "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512" {
		t.Fatalf("project = %+v", lock.Project)
	}
	if lock.Project.CreatedAt != "2026-05-22T10:00:00Z" {
		t.Errorf("created_at = %q", lock.Project.CreatedAt)
	}
	if lock.Project.Stack == nil || lock.Project.Stack.Confidence != "high" ||
		len(lock.Project.Stack.Detected) != 1 {
		t.Errorf("stack = %+v", lock.Project.Stack)
	}
	if len(lock.Project.Disciplines) != 1 || lock.Project.Disciplines[0] != "tdd" {
		t.Errorf("disciplines = %v", lock.Project.Disciplines)
	}

	migration := lock.Migration
	if migration == nil {
		t.Fatal("a migration record is required when there was inherited data")
	}
	if migration.Source != OriginLegacy || migration.SourceSchemaVersion != 2 ||
		migration.MigratedAt != "2026-07-30T12:00:00Z" {
		t.Errorf("migration = %+v", migration)
	}
	if migration.LegacyUpdatedAt != "2026-06-01T09:30:00Z" || migration.LegacySystemVersion != "0.0.9" {
		t.Errorf("inherited timestamps were dropped: %+v", migration)
	}
	if len(migration.LegacyStructure) != 2 {
		t.Errorf("legacy_structure = %v", migration.LegacyStructure)
	}
	var pack struct{ Name string }
	if err := json.Unmarshal(migration.LegacyPack, &pack); err != nil || pack.Name != "context" {
		t.Errorf("legacy_pack = %s (%v)", migration.LegacyPack, err)
	}
	if len(migration.LegacyFlags) == 0 {
		t.Error("legacy_flags were dropped")
	}
	if string(migration.Extra["notes_from_another_tool"]) != `{"keep": true}` {
		t.Errorf("an unknown field was dropped: %v", migration.Extra)
	}
	if migration.Backup != workspace.BackupPath {
		t.Errorf("backup = %q", migration.Backup)
	}
	if string(plan.LegacyBytes) != legacyJSON {
		t.Error("the backup must be written from the original bytes")
	}

	// The adopted resource keeps the moment the workspace said it was installed.
	adopted, ok := lock.Find("problem-framing")
	if !ok || adopted.InstalledAt != "2026-05-22T10:00:00Z" {
		t.Errorf("adopted record = %+v", adopted)
	}
	if len(plan.Adopted) != 1 || plan.Adopted[0].State != "adopt" {
		t.Errorf("adopted = %+v", plan.Adopted)
	}
	if plan.Backup != workspace.BackupPath || len(plan.Retired) != 1 {
		t.Errorf("plan = %+v", plan)
	}
}

// Contradictions are never resolved by precedence: the plan blocks.
func TestComputeBlocksOnContradictions(t *testing.T) {
	tests := []struct {
		name  string
		state func() State
	}{
		{
			name: "different project ids",
			state: func() State {
				state := lockV2()
				state.Lock.Project.ID = "a-different-identity"
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
		},
		{
			name: "different creation timestamps",
			state: func() State {
				state := lockV2()
				state.Lock.Project.CreatedAt = "2020-01-01T00:00:00Z"
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
		},
		{
			name: "different stacks",
			state: func() State {
				state := lockV2()
				state.Lock.Project.Stack = &model.LockStack{
					Detected: []string{"rust"}, Source: "detected", Confidence: "low",
				}
				state.Legacy = legacy(t, legacyJSON)
				return state
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := compute(t, test.state())
			if !plan.Blocked() {
				t.Fatalf("the plan should be blocked: %+v", plan)
			}
			if !codes(plan.Blockers)[errs.CodeWorkspaceInvalid] {
				t.Errorf("blockers = %+v, want workspace_invalid", plan.Blockers)
			}
		})
	}
}

// A resource the workspace holds that cannot be verified against the catalog aborts the
// migration; a merely informative finding does not.
func TestComputeAggregatesAdoptionVerdicts(t *testing.T) {
	state := base()
	state.Legacy = legacy(t, legacyJSON)
	state.Adoptions = []Adoption{
		{
			Ref: "problem-framing",
			Blocking: []model.Diagnostic{{
				Code: errs.CodeLocalDivergence, Ref: "problem-framing",
				Path: ".agents/skills/problem-framing/SKILL.md",
				Message: "the installed file differs from the catalog",
			}},
		},
		{
			Ref: "stray",
			Warnings: []model.Diagnostic{{
				Code: errs.CodeNotFound, Ref: "stray", Message: "no source offers this resource",
			}},
		},
	}

	plan := compute(t, state)
	if !plan.Blocked() || !codes(plan.Blockers)[errs.CodeLocalDivergence] {
		t.Errorf("blockers = %+v, want local_divergence", plan.Blockers)
	}
	if !codes(plan.Warnings)[errs.CodeNotFound] {
		t.Errorf("warnings = %+v, want the informative finding", plan.Warnings)
	}
	if len(plan.Adopted) != 0 {
		t.Errorf("nothing may be adopted from a blocked verdict: %+v", plan.Adopted)
	}
}

// The lockfile proves ownership: a workspace entry never overwrites a record that exists.
func TestComputeNeverOverwritesAnExistingRecord(t *testing.T) {
	state := lockV2()
	state.Legacy = legacy(t, legacyJSON)
	state.Adoptions = []Adoption{{
		Ref: "problem-framing",
		Record: &model.LockResource{
			ID: "problem-framing", Type: model.TypeSkill, Source: "other",
			Version: "9.9.9", Checksum: "sha256:different",
		},
	}}

	plan := compute(t, state)
	record, ok := plan.Lock.Find("problem-framing")
	if !ok || record.Version != "1.0.0" || record.Source != "public" {
		t.Errorf("record = %+v", record)
	}
	if len(plan.Adopted) != 0 {
		t.Errorf("adopted = %+v, want nothing", plan.Adopted)
	}
}

// A v1 lockfile with no workspace has no identity to inherit, so one is generated. The
// resources it records are untouched.
func TestComputeUpgradesALegacyLockDeterministically(t *testing.T) {
	plan := compute(t, lockV1())
	if plan.Lock.Project == nil || plan.Lock.Project.ID != "generated-id" {
		t.Fatalf("project = %+v", plan.Lock.Project)
	}
	if plan.Lock.Project.CreatedAt != "2026-07-30T12:00:00Z" {
		t.Errorf("created_at = %q", plan.Lock.Project.CreatedAt)
	}
	if plan.Lock.Migration != nil {
		t.Errorf("an upgrade without inherited data needs no migration record: %+v", plan.Lock.Migration)
	}
	if len(plan.Lock.Resources) != 1 || plan.Lock.Resources[0].ID != "problem-framing" {
		t.Errorf("resources = %+v", plan.Lock.Resources)
	}
	if plan.FromSchema != model.LockSchemaVersionLegacy || plan.ToSchema != model.LockSchemaVersion {
		t.Errorf("schemas = %d -> %d", plan.FromSchema, plan.ToSchema)
	}

	// Same input, same plan.
	again := compute(t, lockV1())
	if again.Lock.Project.ID != plan.Lock.Project.ID || again.Lock.GeneratedAt != plan.Lock.GeneratedAt {
		t.Error("Compute is not deterministic")
	}
}

// Applying a migration and computing it again produces a plan that writes nothing.
func TestComputeIsIdempotentAfterAMigration(t *testing.T) {
	first := compute(t, func() State {
		state := base()
		state.Legacy = legacy(t, legacyJSON)
		return state
	}())

	settled := base()
	settled.LockSchema = model.LockSchemaVersion
	settled.Lock = first.Lock
	settled.Backup, settled.BackupPresent = first.LegacyBytes, true
	// The workspace is gone: the previous migration retired it.

	second := compute(t, settled)
	if !second.Empty() || second.Blocked() {
		t.Errorf("a repeated migration must change nothing: %+v", second.Changes)
	}
	if second.Lock.GeneratedAt != first.Lock.GeneratedAt {
		t.Error("a no-op migration must not restamp the lockfile")
	}
}

// The runtime is derivable, so a mismatch is reported instead of blocking: the inherited
// value survives byte for byte in the backup.
func TestComputeWarnsAboutADifferentInheritedRuntime(t *testing.T) {
	state := base()
	state.Runtime = "opencode"
	state.Lock = model.NewLock("opencode")
	state.Legacy = legacy(t, legacyJSON)

	plan := compute(t, state)
	if plan.Blocked() {
		t.Fatalf("a different runtime must not block: %+v", plan.Blockers)
	}
	if !codes(plan.Warnings)[errs.CodeRuntimeUnsupported] {
		t.Errorf("warnings = %+v", plan.Warnings)
	}
	if plan.Lock.Runtime != "opencode" {
		t.Errorf("runtime = %q", plan.Lock.Runtime)
	}
}
