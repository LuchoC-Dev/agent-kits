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
