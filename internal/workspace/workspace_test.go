package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

func runtime(t *testing.T) adapter.Adapter {
	t.Helper()
	a, err := adapter.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestLoadLockReturnsEmptyLockWhenAbsent(t *testing.T) {
	project := t.TempDir()
	a := runtime(t)
	lock, err := LoadLock(project, a)
	if err != nil {
		t.Fatalf("LoadLock returned %v", err)
	}
	if len(lock.Resources) != 0 || lock.Runtime != a.Name() {
		t.Errorf("lock = %+v", lock)
	}
}

// A lockfile written by an earlier build stays readable and is upgraded in memory, so the
// next write produces the current schema (D-030).
func TestLoadLockUpgradesTheLegacySchema(t *testing.T) {
	project := t.TempDir()
	a := runtime(t)
	writeLock(t, project, a, `{
  "schema_version": 1,
  "runtime": "claude-code",
  "generated_at": "2026-05-22T10:00:00Z",
  "resources": [{"id":"tdd","type":"skill","source":"public","version":"1.0.0",
    "checksum":"sha256:x","requested":true,"files":[]}]
}`)
	lock, found, err := LoadLockDetail(project, a)
	if err != nil {
		t.Fatalf("LoadLockDetail returned %v", err)
	}
	if found != model.LockSchemaVersionLegacy {
		t.Errorf("found schema %d, want %d", found, model.LockSchemaVersionLegacy)
	}
	if lock.SchemaVersion != model.LockSchemaVersion {
		t.Errorf("the lock was not upgraded in memory: %d", lock.SchemaVersion)
	}
	if len(lock.Resources) != 1 {
		t.Errorf("resources = %+v", lock.Resources)
	}
	// Reading assigns no identity: that is an operation's decision.
	if lock.Project != nil {
		t.Errorf("project = %+v, want nil", lock.Project)
	}
}

func TestLoadLockRejectsAnUnknownSchema(t *testing.T) {
	project := t.TempDir()
	a := runtime(t)
	writeLock(t, project, a, `{"schema_version": 99, "runtime": "claude-code", "resources": []}`)
	if _, err := LoadLock(project, a); !errs.Is(err, errs.CodeWorkspaceInvalid) {
		t.Errorf("LoadLock returned %v, want workspace_invalid", err)
	}
}

func TestLoadLockDetailReportsAbsence(t *testing.T) {
	lock, found, err := LoadLockDetail(t.TempDir(), runtime(t))
	if err != nil {
		t.Fatalf("LoadLockDetail returned %v", err)
	}
	if found != 0 || lock.SchemaVersion != model.LockSchemaVersion {
		t.Errorf("found = %d, lock = %+v", found, lock)
	}
}

func TestLockBytesAreIndentedAndNewlineTerminated(t *testing.T) {
	lock := model.NewLock("claude-code")
	lock.EnsureProject("3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512", "2026-05-22T10:00:00Z")
	data, err := LockBytes(lock)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasSuffix(text, "\n") || !strings.Contains(text, "\n  \"project\"") {
		t.Errorf("lockfile is not formatted for review:\n%s", text)
	}
	var back model.Lock
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("the rendered lockfile is not valid JSON: %v", err)
	}
}

// The legacy reader must lose nothing: the original bytes, the managed fields and the
// fields written by another tool all survive.
func TestLoadLegacyIsLossless(t *testing.T) {
	project := t.TempDir()
	content := `{
  "$schema_version": 2,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-06-01T09:30:00Z",
  "system_version": "0.0.9",
  "runtime": "opencode",
  "pack": { "name": "context", "source": "packs/context", "installed_at": "2026-05-22T10:00:00Z" },
  "stack": { "detected": ["go"], "source": "user-input", "confidence": "high" },
  "skills": [{ "id": "problem-framing", "source": "skills/problem-framing", "installed_at": "2026-05-22T10:00:00Z" }],
  "agents": [],
  "disciplines": ["tdd"],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": ["skills"],
  "notes_from_another_tool": {"keep": true},
  "unknown_scalar": 7
}`
	writeRaw(t, project, LegacyPath, content)

	legacy, present, err := LoadLegacy(project)
	if err != nil || !present {
		t.Fatalf("LoadLegacy = %v, %v", present, err)
	}
	if string(legacy.Raw) != content {
		t.Error("the original bytes were not preserved verbatim")
	}
	descriptor := legacy.Descriptor
	if descriptor.ID != "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512" ||
		descriptor.CreatedAt != "2026-05-22T10:00:00Z" ||
		descriptor.UpdatedAt != "2026-06-01T09:30:00Z" ||
		descriptor.SystemVersion != "0.0.9" || descriptor.Runtime != "opencode" {
		t.Errorf("descriptor = %+v", descriptor)
	}
	if len(descriptor.Disciplines) != 1 || descriptor.Disciplines[0] != "tdd" {
		t.Errorf("disciplines = %v", descriptor.Disciplines)
	}
	if descriptor.Pack == nil || descriptor.Pack.Name != "context" ||
		descriptor.Stack == nil || descriptor.Stack.Confidence != "high" ||
		len(descriptor.Skills) != 1 || !descriptor.Flags.Initialized {
		t.Errorf("managed fields were dropped: %+v", descriptor)
	}

	extra := descriptor.Extra()
	if string(extra["notes_from_another_tool"]) != `{"keep": true}` || string(extra["unknown_scalar"]) != "7" {
		t.Errorf("unknown fields were not preserved: %v", extra)
	}
	// Extra returns a copy: mutating it must not corrupt the descriptor.
	extra["notes_from_another_tool"] = json.RawMessage(`null`)
	if string(descriptor.Extra()["notes_from_another_tool"]) != `{"keep": true}` {
		t.Error("Extra must not share storage with the descriptor")
	}

	// Managed fields are also reachable verbatim, so a migration can preserve the inherited
	// value instead of re-encoding it.
	pack, ok := descriptor.RawField("pack")
	if !ok || !strings.Contains(string(pack), `"name": "context"`) {
		t.Errorf("RawField(pack) = %s, %v", pack, ok)
	}
	if _, ok := descriptor.RawField("nothing_here"); ok {
		t.Error("RawField invented a field")
	}
}

func TestLoadLegacyReportsAbsenceAndRejectsInvalidContent(t *testing.T) {
	if _, present, err := LoadLegacy(t.TempDir()); err != nil || present {
		t.Fatalf("LoadLegacy = %v, %v", present, err)
	}
	for name, content := range map[string]string{
		"unsupported schema": `{"$schema_version": 99}`,
		"not json":           `{`,
	} {
		project := t.TempDir()
		writeRaw(t, project, LegacyPath, content)
		if _, _, err := LoadLegacy(project); !errs.Is(err, errs.CodeWorkspaceInvalid) {
			t.Errorf("%s: LoadLegacy returned %v, want workspace_invalid", name, err)
		}
	}
	// A v1 descriptor is still readable: it is exactly what a migration exists for.
	project := t.TempDir()
	writeRaw(t, project, LegacyPath, `{"$schema_version": 1, "id": "x", "runtime": "agents"}`)
	if _, present, err := LoadLegacy(project); err != nil || !present {
		t.Errorf("LoadLegacy on a v1 descriptor = %v, %v", present, err)
	}
}

func TestLoadLegacyRefusesAnOversizedDescriptor(t *testing.T) {
	project := t.TempDir()
	writeRaw(t, project, LegacyPath, `{"$schema_version": 2, "filler": "`+
		strings.Repeat("x", int(MaxLegacyBytes))+`"}`)
	if _, _, err := LoadLegacy(project); !errs.Is(err, errs.CodeWorkspaceInvalid) {
		t.Errorf("LoadLegacy returned %v, want workspace_invalid", err)
	}
}

func TestLoadBackupReportsAbsenceAndContent(t *testing.T) {
	project := t.TempDir()
	if _, present, err := LoadBackup(project); err != nil || present {
		t.Fatalf("LoadBackup = %v, %v", present, err)
	}
	writeRaw(t, project, BackupPath, `{"$schema_version": 2}`)
	raw, present, err := LoadBackup(project)
	if err != nil || !present || string(raw) != `{"$schema_version": 2}` {
		t.Errorf("LoadBackup = %q, %v, %v", raw, present, err)
	}
}

// Pending is the single question every mutating command asks before writing.
func TestPendingDetectsAnUnmigratedProject(t *testing.T) {
	project := t.TempDir()
	if Pending(project) {
		t.Error("an empty project is not pending migration")
	}
	writeRaw(t, project, LegacyPath, `{"$schema_version": 2}`)
	if !Pending(project) {
		t.Error("a project with workspace.json is pending migration")
	}
	if !errs.Is(PendingError(), errs.CodeWorkspaceInvalid) {
		t.Errorf("PendingError = %v, want workspace_invalid", PendingError())
	}
	if !strings.Contains(PendingError().Error(), "workspace.json") {
		t.Errorf("PendingError must name the file: %v", PendingError())
	}
}

// writeRaw materialises a project-relative file for a test.
func writeRaw(t *testing.T, project, rel, content string) {
	t.Helper()
	path := filepath.Join(project, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeLock(t *testing.T, project string, a adapter.Adapter, content string) {
	t.Helper()
	writeRaw(t, project, a.LockPath(), content)
}
