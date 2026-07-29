package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

var stamp = time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)

func runtime(t *testing.T) adapter.Adapter {
	t.Helper()
	a, err := adapter.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func sampleLock() *model.Lock {
	lock := model.NewLock("claude-code")
	lock.Upsert(model.LockResource{ID: "tdd", Type: model.TypeSkill, Version: "1.0.0"})
	lock.Upsert(model.LockResource{ID: "problem-framing", Type: model.TypeSkill, Version: "1.0.0"})
	lock.Upsert(model.LockResource{ID: "context/context-builder", Type: model.TypeAgent, Version: "1.0.0"})
	lock.Upsert(model.LockResource{ID: "design-critic", Type: model.TypeAgent, Version: "1.0.0"})
	lock.Upsert(model.LockResource{ID: "context/context-building", Type: model.TypeWorkflow, Version: "1.0.0"})
	lock.Upsert(model.LockResource{ID: "context", Type: model.TypeKit, Version: "1.0.0", Requested: true})
	return lock
}

func sampleResources() map[model.ID]*model.Resource {
	discipline := &model.Resource{Manifest: model.Manifest{
		ID: "tdd", Type: model.TypeSkill, Traits: map[string]bool{"discipline": true},
	}}
	classTwo := &model.Resource{Manifest: model.Manifest{
		ID: "design-critic", Type: model.TypeAgent, Labels: map[string]string{"class": "2"},
	}}
	return map[model.ID]*model.Resource{discipline.ID: discipline, classTwo.ID: classTwo}
}

func TestSyncBuildsDescriptorFromLock(t *testing.T) {
	descriptor, err := Sync(nil, sampleLock(), "claude-code", sampleResources(), stamp)
	if err != nil {
		t.Fatalf("Sync returned %v", err)
	}
	if descriptor.SchemaVersion != SchemaVersion || descriptor.Runtime != "claude-code" {
		t.Errorf("descriptor = %+v", descriptor)
	}
	if descriptor.ID == "" || !strings.Contains(descriptor.ID, "-") {
		t.Errorf("a new workspace needs a generated id, got %q", descriptor.ID)
	}
	if descriptor.CreatedAt != descriptor.UpdatedAt {
		t.Error("a new workspace should be created and updated at the same instant")
	}
	if len(descriptor.Skills) != 2 || len(descriptor.Agents) != 2 {
		t.Errorf("skills = %+v agents = %+v", descriptor.Skills, descriptor.Agents)
	}
	// disciplines[] is the source of truth for which disciplines are active.
	if len(descriptor.Disciplines) != 1 || descriptor.Disciplines[0] != "tdd" {
		t.Errorf("disciplines = %v", descriptor.Disciplines)
	}
	if descriptor.Pack == nil || descriptor.Pack.Name != "context" {
		t.Errorf("pack = %+v", descriptor.Pack)
	}
	// structure lists only the directories that actually have content.
	want := []string{"agents", "packs", "skills", "workflows"}
	if strings.Join(descriptor.Structure, ",") != strings.Join(want, ",") {
		t.Errorf("structure = %v, want %v", descriptor.Structure, want)
	}
	if !descriptor.Flags.Initialized {
		t.Error("flags.initialized must be set")
	}
}

func TestSyncClassifiesAgentsLikeTheInheritedTaxonomy(t *testing.T) {
	descriptor, err := Sync(nil, sampleLock(), "claude-code", sampleResources(), stamp)
	if err != nil {
		t.Fatal(err)
	}
	classes := map[string]int{}
	for _, entry := range descriptor.Agents {
		classes[entry.ID] = entry.Class
	}
	// An agent owned by a kit orchestrates its workflow: class 1. A shared agent is class 2.
	if classes["context/context-builder"] != 1 {
		t.Errorf("context/context-builder class = %d, want 1", classes["context/context-builder"])
	}
	if classes["design-critic"] != 2 {
		t.Errorf("design-critic class = %d, want 2", classes["design-critic"])
	}
}

func TestSyncNamesTheCompositionCustomWhenSeveralKitsAreInstalled(t *testing.T) {
	lock := sampleLock()
	lock.Upsert(model.LockResource{ID: "tools", Type: model.TypeKit, Version: "1.0.0", Requested: true})
	descriptor, err := Sync(nil, lock, "claude-code", nil, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Pack.Name != "custom" {
		t.Errorf("pack = %+v, want custom", descriptor.Pack)
	}
}

func TestSyncPreservesUnmanagedFieldsAndTimestamps(t *testing.T) {
	original := []byte(`{
  "$schema_version": 2,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-05-22T10:00:00Z",
  "system_version": "0.0.9",
  "runtime": "opencode",
  "pack": { "name": "context", "source": "packs/context", "installed_at": "2026-05-22T10:00:00Z" },
  "stack": { "detected": ["go"], "source": "user-input", "confidence": "high" },
  "skills": [{ "id": "problem-framing", "source": "skills/problem-framing", "installed_at": "2026-05-22T10:00:00Z" }],
  "agents": [],
  "disciplines": [],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": ["skills"],
  "notes_from_another_tool": {"keep": true}
}`)
	var existing Descriptor
	if err := json.Unmarshal(original, &existing); err != nil {
		t.Fatalf("Unmarshal returned %v", err)
	}

	descriptor, err := Sync(&existing, sampleLock(), "claude-code", sampleResources(), stamp)
	if err != nil {
		t.Fatalf("Sync returned %v", err)
	}
	if descriptor.ID != "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512" {
		t.Errorf("the workspace id changed: %s", descriptor.ID)
	}
	if descriptor.CreatedAt != "2026-05-22T10:00:00Z" {
		t.Errorf("created_at changed: %s", descriptor.CreatedAt)
	}
	if descriptor.UpdatedAt == descriptor.CreatedAt {
		t.Error("updated_at should advance")
	}
	if descriptor.Stack == nil || descriptor.Stack.Confidence != "high" {
		t.Errorf("stack = %+v", descriptor.Stack)
	}
	// A resource that was already recorded keeps its original install time.
	for _, entry := range descriptor.Skills {
		if entry.ID == "problem-framing" && entry.InstalledAt != "2026-05-22T10:00:00Z" {
			t.Errorf("problem-framing installed_at = %s", entry.InstalledAt)
		}
	}

	encoded, err := DescriptorBytes(descriptor)
	if err != nil {
		t.Fatalf("DescriptorBytes returned %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := fields["notes_from_another_tool"]; !ok {
		t.Error("an unmanaged field was dropped")
	}
}

// The output must keep the documented field order so a CLI-written file stays
// diff-friendly against one written by the conversational flow.
func TestDescriptorBytesKeepsDocumentedFieldOrder(t *testing.T) {
	descriptor, err := Sync(nil, sampleLock(), "claude-code", sampleResources(), stamp)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := DescriptorBytes(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.HasSuffix(text, "\n") {
		t.Error("workspace.json must end with a newline")
	}
	if !strings.Contains(text, "\n  \"id\"") {
		t.Errorf("workspace.json is not indented:\n%s", text)
	}
	previous := -1
	for _, field := range fieldOrder {
		index := strings.Index(text, `"`+field+`"`)
		if index < 0 {
			continue
		}
		if index < previous {
			t.Fatalf("field %q appears out of order:\n%s", field, text)
		}
		previous = index
	}
}

// A v1 workspace is upgraded in place by gaining the disciplines field.
func TestSyncUpgradesSchemaVersionOne(t *testing.T) {
	var existing Descriptor
	if err := json.Unmarshal([]byte(`{
  "$schema_version": 1,
  "id": "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
  "created_at": "2026-05-22T10:00:00Z",
  "updated_at": "2026-05-22T10:00:00Z",
  "runtime": "claude-code",
  "skills": [],
  "agents": [],
  "flags": { "initialized": true, "repaired_at": null, "upgraded_at": null },
  "structure": []
}`), &existing); err != nil {
		t.Fatal(err)
	}
	descriptor, err := Sync(&existing, sampleLock(), "claude-code", sampleResources(), stamp)
	if err != nil {
		t.Fatalf("Sync returned %v", err)
	}
	if descriptor.SchemaVersion != SchemaVersion {
		t.Errorf("$schema_version = %d, want %d", descriptor.SchemaVersion, SchemaVersion)
	}
	if descriptor.Disciplines == nil {
		t.Error("the upgrade must add a disciplines array")
	}
}

func TestLoadDescriptorRejectsUnsupportedSchema(t *testing.T) {
	project := t.TempDir()
	a := runtime(t)
	writeWorkspace(t, project, a, `{"$schema_version": 99}`)
	if _, _, err := LoadDescriptor(project, a); err == nil {
		t.Error("an unsupported $schema_version was accepted")
	}
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

func TestLoadDescriptorReportsAbsence(t *testing.T) {
	_, present, err := LoadDescriptor(t.TempDir(), runtime(t))
	if err != nil {
		t.Fatalf("LoadDescriptor returned %v", err)
	}
	if present {
		t.Error("an absent descriptor was reported as present")
	}
}

// writeWorkspace materialises a raw workspace.json for a test.
func writeWorkspace(t *testing.T, project string, a adapter.Adapter, content string) {
	t.Helper()
	path := filepath.Join(project, filepath.FromSlash(a.WorkspacePath()))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
