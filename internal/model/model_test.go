package model

import (
	"encoding/json"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

func TestParseID(t *testing.T) {
	valid := []string{"frontend-design", "sdd", "backend/feature-development", "a1/b2-c3"}
	for _, input := range valid {
		if _, err := ParseID(input); err != nil {
			t.Errorf("ParseID(%q) returned %v", input, err)
		}
	}
	invalid := []string{
		"", "Frontend", "front_end", "-leading", "trailing-", "a//b",
		"a/b/c", "a b", "../escape", "kit/", "/name",
	}
	for _, input := range invalid {
		if _, err := ParseID(input); err == nil {
			t.Errorf("ParseID(%q) accepted an invalid id", input)
		}
	}
}

func TestIDOwnerAndName(t *testing.T) {
	global := ID("frontend-design")
	if global.Qualified() || global.Owner() != "" || global.Name() != "frontend-design" {
		t.Errorf("global id parts are wrong: %v %q %q", global.Qualified(), global.Owner(), global.Name())
	}
	owned := ID("backend/feature-development")
	if !owned.Qualified() || owned.Owner() != "backend" || owned.Name() != "feature-development" {
		t.Errorf("owned id parts are wrong: %v %q %q", owned.Qualified(), owned.Owner(), owned.Name())
	}
}

func TestIDMatches(t *testing.T) {
	owned := ID("backend/feature-development")
	if !owned.Matches("backend/feature-development") {
		t.Error("an id must match its own canonical form")
	}
	if !owned.Matches("feature-development") {
		t.Error("an owned id must match its bare name")
	}
	if owned.Matches("frontend/feature-development") {
		t.Error("an owned id must not match another owner's qualified id")
	}
	if ID("frontend-design").Matches("design") {
		t.Error("a bare name must match the whole segment, not a suffix")
	}
}

func TestManifestValidate(t *testing.T) {
	base := func() Manifest {
		return Manifest{
			SchemaVersion: ManifestSchemaVersion,
			ID:            "example",
			Type:          TypeSkill,
			Version:       "1.0.0",
			Files:         []string{"SKILL.md"},
		}
	}
	if err := (&Manifest{
		ID: "example", Type: TypeSkill, Version: "1.0.0", Files: []string{"SKILL.md"},
	}).Validate(); err != nil {
		t.Errorf("a missing schema_version should default, got %v", err)
	}

	cases := map[string]func(*Manifest){
		"unsupported schema": func(m *Manifest) { m.SchemaVersion = 99 },
		"bad type":           func(m *Manifest) { m.Type = "tool" },
		"bad version":        func(m *Manifest) { m.Version = "1.0" },
		"no files":           func(m *Manifest) { m.Files = nil },
		"self dependency":    func(m *Manifest) { m.Dependencies = []Dependency{{ID: "example"}} },
		"duplicate dep": func(m *Manifest) {
			m.Dependencies = []Dependency{{ID: "other"}, {ID: "other"}}
		},
		"bad constraint": func(m *Manifest) {
			m.Dependencies = []Dependency{{ID: "other", Version: ">=1 <2"}}
		},
	}
	for name, mutate := range cases {
		manifest := base()
		mutate(&manifest)
		err := manifest.Validate()
		if err == nil {
			t.Errorf("%s: Validate accepted an invalid manifest", name)
			continue
		}
		if code := errs.CodeOf(err); code != errs.CodeInvalidManifest && code != errs.CodeUsage {
			t.Errorf("%s: code = %s", name, code)
		}
	}
}

func TestDependencyUnmarshalAcceptsBothForms(t *testing.T) {
	var shorthand Dependency
	if err := json.Unmarshal([]byte(`"tdd"`), &shorthand); err != nil {
		t.Fatalf("shorthand form returned %v", err)
	}
	if shorthand.ID != "tdd" || shorthand.Version != "" {
		t.Errorf("shorthand = %+v", shorthand)
	}

	var explicit Dependency
	if err := json.Unmarshal([]byte(`{"id":"tdd","version":"^1.0.0"}`), &explicit); err != nil {
		t.Fatalf("explicit form returned %v", err)
	}
	if explicit.ID != "tdd" || explicit.Version != "^1.0.0" {
		t.Errorf("explicit = %+v", explicit)
	}

	var invalid Dependency
	if err := json.Unmarshal([]byte(`{"id":"NOPE"}`), &invalid); err == nil {
		t.Error("an invalid id was accepted")
	}
}

func TestSupportsRuntime(t *testing.T) {
	open := Manifest{}
	if !open.SupportsRuntime("anything") {
		t.Error("an empty runtime list must accept every runtime")
	}
	restricted := Manifest{Runtimes: []string{"claude-code"}}
	if !restricted.SupportsRuntime("claude-code") || restricted.SupportsRuntime("opencode") {
		t.Error("runtime restriction is not honoured")
	}
}

func TestPlanEmptyIgnoresMetadataAndUnchanged(t *testing.T) {
	p := &Plan{
		Changes: []FileChange{
			{Path: "a", Action: ActionUnchanged},
			{Path: "b", Action: ActionUnchanged},
		},
		Metadata: []FileChange{{Path: ".agents/agent-kits.lock.json", Action: ActionUpdate}},
	}
	if !p.Empty() {
		t.Error("a plan of unchanged files must be empty")
	}
	p.Changes = append(p.Changes, FileChange{Path: "c", Action: ActionCreate})
	if p.Empty() {
		t.Error("a plan that creates a file must not be empty")
	}
}

func TestLockRoundTrip(t *testing.T) {
	lock := NewLock("claude-code")
	lock.Upsert(LockResource{
		ID: "b", Type: TypeSkill, Version: "1.0.0", Checksum: "sha256:x",
		Files: []LockFile{{Path: ".agents/skills/b/SKILL.md", Checksum: "sha256:y"}},
	})
	lock.Upsert(LockResource{ID: "a", Type: TypeSkill, Version: "1.0.0", Checksum: "sha256:z"})

	if lock.Resources[0].ID != "a" {
		t.Errorf("Upsert must keep resources sorted, got %s first", lock.Resources[0].ID)
	}
	owner, checksum, ok := lock.FileOwner(".agents/skills/b/SKILL.md")
	if !ok || owner != "b" || checksum != "sha256:y" {
		t.Errorf("FileOwner = %s, %s, %v", owner, checksum, ok)
	}

	clone := lock.Clone()
	clone.Resources[0].Version = "2.0.0"
	if lock.Resources[0].Version != "1.0.0" {
		t.Error("Clone must not share resource storage")
	}

	if !lock.Delete("a") || lock.Delete("a") {
		t.Error("Delete must report whether the record existed")
	}
	if err := lock.Validate(); err != nil {
		t.Errorf("Validate returned %v", err)
	}

	lock.Resources = append(lock.Resources, lock.Resources[0])
	if err := lock.Validate(); err == nil {
		t.Error("Validate must reject a duplicated record")
	}
}
