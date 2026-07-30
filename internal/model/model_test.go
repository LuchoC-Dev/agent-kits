package model

import (
	"encoding/json"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

const sampleID = "9f2c1b7a-9d84-4e11-b6f2-77c1a9e0d512"

func TestParseID(t *testing.T) {
	valid := []string{sampleID, "3F1C2B7A-9D84-4E11-B6F2-77C1A9E0D512"}
	for _, input := range valid {
		if _, err := ParseID(input); err != nil {
			t.Errorf("ParseID(%q) returned %v", input, err)
		}
	}
	// An id is a UUID. A name, however well formed, is not an identity (D-035).
	invalid := []string{
		"", "frontend-design", "backend/feature-development", "9f2c1b7a9d844e11b6f277c1a9e0d512",
		"9f2c1b7a-9d84-4e11-b6f2", "zzzzzzzz-9d84-4e11-b6f2-77c1a9e0d512",
	}
	for _, input := range invalid {
		if _, err := ParseID(input); err == nil {
			t.Errorf("ParseID(%q) accepted an invalid id", input)
		}
	}
	// Identities normalise to lower case, so two spellings are one identity.
	upper, _ := ParseID("3F1C2B7A-9D84-4E11-B6F2-77C1A9E0D512")
	if upper != ID("3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512") {
		t.Errorf("ParseID did not normalise: %s", upper)
	}
}

func TestParseName(t *testing.T) {
	for _, input := range []string{"frontend-design", "sdd", "a1-b2-c3"} {
		if _, err := ParseName(input); err != nil {
			t.Errorf("ParseName(%q) returned %v", input, err)
		}
	}
	// A name carries no namespace: no kit prefix and no source prefix (D-036).
	invalid := []string{
		"", "Frontend", "front_end", "-leading", "trailing-",
		"backend/feature-development", "public:tdd", "a b",
	}
	for _, input := range invalid {
		if _, err := ParseName(input); err == nil {
			t.Errorf("ParseName(%q) accepted an invalid name", input)
		}
	}
}

// The three reference forms of D-036.
func TestParseReference(t *testing.T) {
	byID, err := ParseReference(sampleID)
	if err != nil || byID.ID != ID(sampleID) || byID.Name != "" || byID.Qualified() {
		t.Errorf("ParseReference(uuid) = %+v, %v", byID, err)
	}
	bare, err := ParseReference("frontend-design")
	if err != nil || bare.ID != "" || bare.Name != "frontend-design" || bare.Qualified() {
		t.Errorf("ParseReference(name) = %+v, %v", bare, err)
	}
	qualified, err := ParseReference("acme:frontend-design")
	if err != nil || !qualified.Qualified() || qualified.Source != "acme" ||
		qualified.Name != "frontend-design" {
		t.Errorf("ParseReference(source:name) = %+v, %v", qualified, err)
	}
	if qualified.String() != "acme:frontend-design" || bare.String() != "frontend-design" ||
		byID.String() != sampleID {
		t.Error("a reference must render the way it was written")
	}
	for _, bad := range []string{"", ":tdd", "acme:", "acme:Bad-Name"} {
		if _, err := ParseReference(bad); err == nil {
			t.Errorf("ParseReference(%q) accepted an invalid reference", bad)
		}
	}
}

func TestLooksLikeIDRoutesReferences(t *testing.T) {
	if !LooksLikeID(sampleID) || LooksLikeID("frontend-design") || LooksLikeID("acme:tdd") {
		t.Error("LooksLikeID must separate an identity from a name")
	}
}

func TestNewIDIsUniqueAndParseable(t *testing.T) {
	first, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	second, _ := NewID()
	if first == second {
		t.Error("two identities must differ")
	}
	if _, err := ParseID(string(first)); err != nil {
		t.Errorf("a generated id must parse: %v", err)
	}
	if len(first.Short()) != 8 {
		t.Errorf("Short() = %q", first.Short())
	}
}

const otherID = "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512"

func TestManifestValidate(t *testing.T) {
	base := func() Manifest {
		return Manifest{
			SchemaVersion: ManifestSchemaVersion,
			ID:            sampleID,
			Name:          "example",
			Type:          TypeSkill,
			Version:       "1.0.0",
			Files:         []string{"SKILL.md"},
		}
	}
	valid := base()
	valid.SchemaVersion = 0
	if err := valid.Validate(); err != nil {
		t.Errorf("a missing schema_version should default, got %v", err)
	}

	cases := map[string]func(*Manifest){
		"unsupported schema": func(m *Manifest) { m.SchemaVersion = 99 },
		"missing id":         func(m *Manifest) { m.ID = "" },
		"name as id":         func(m *Manifest) { m.ID = "example" },
		"missing name":       func(m *Manifest) { m.Name = "" },
		"qualified name":     func(m *Manifest) { m.Name = "backend/example" },
		"bad type":           func(m *Manifest) { m.Type = "tool" },
		"bad version":        func(m *Manifest) { m.Version = "1.0" },
		"no files":           func(m *Manifest) { m.Files = nil },
		"self dependency":    func(m *Manifest) { m.Dependencies = []Dependency{{ID: sampleID}} },
		"duplicate dep": func(m *Manifest) {
			m.Dependencies = []Dependency{{ID: otherID}, {ID: otherID}}
		},
		"bad constraint": func(m *Manifest) {
			m.Dependencies = []Dependency{{ID: otherID, Version: ">=1 <2"}}
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

// The human label is presentation only; the install name is the fallback.
func TestDisplayName(t *testing.T) {
	plain := Manifest{Name: "frontend-design"}
	if plain.DisplayName() != "frontend-design" {
		t.Errorf("DisplayName = %q", plain.DisplayName())
	}
	titled := Manifest{Name: "frontend-design", Title: "Frontend Design"}
	if titled.DisplayName() != "Frontend Design" {
		t.Errorf("DisplayName = %q", titled.DisplayName())
	}
}

func TestDependencyUnmarshalAcceptsBothForms(t *testing.T) {
	var shorthand Dependency
	if err := json.Unmarshal([]byte(`"`+sampleID+`"`), &shorthand); err != nil {
		t.Fatalf("shorthand form returned %v", err)
	}
	if shorthand.ID != ID(sampleID) || shorthand.Version != "" {
		t.Errorf("shorthand = %+v", shorthand)
	}

	var explicit Dependency
	if err := json.Unmarshal([]byte(`{"id":"`+sampleID+`","name":"tdd","version":"^1.0.0"}`),
		&explicit); err != nil {
		t.Fatalf("explicit form returned %v", err)
	}
	if explicit.ID != ID(sampleID) || explicit.Name != "tdd" || explicit.Version != "^1.0.0" {
		t.Errorf("explicit = %+v", explicit)
	}
	// The recorded name is for humans; the label falls back to the identity without one.
	if explicit.Label() != "tdd" || shorthand.Label() != sampleID[:8] {
		t.Errorf("labels = %q, %q", explicit.Label(), shorthand.Label())
	}

	var invalid Dependency
	if err := json.Unmarshal([]byte(`{"id":"tdd"}`), &invalid); err == nil {
		t.Error("a name was accepted as an identity")
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
	const idA, idB = sampleID, otherID
	lock := NewLock("claude-code")
	lock.Upsert(LockResource{
		ID: idB, Name: "b", Type: TypeSkill, Version: "1.0.0", Checksum: "sha256:x",
		Files: []LockFile{{Path: ".agents/skills/b/SKILL.md", Checksum: "sha256:y"}},
	})
	lock.Upsert(LockResource{
		ID: idA, Name: "a", Type: TypeSkill, Version: "1.0.0", Checksum: "sha256:z",
	})

	// The lockfile is ordered by name, so its diff reads like a diff of the project.
	if lock.Resources[0].Name != "a" {
		t.Errorf("Upsert must keep resources sorted by name, got %s first", lock.Resources[0].Name)
	}
	owner, checksum, ok := lock.FileOwner(".agents/skills/b/SKILL.md")
	if !ok || owner != ID(idB) || checksum != "sha256:y" {
		t.Errorf("FileOwner = %s, %s, %v", owner, checksum, ok)
	}
	if record, found := lock.FindByName("b"); !found || record.ID != ID(idB) {
		t.Errorf("FindByName = %+v, %v", record, found)
	}
	if _, found := lock.FindByName("nothing"); found {
		t.Error("FindByName invented a record")
	}

	clone := lock.Clone()
	clone.Resources[0].Version = "2.0.0"
	if lock.Resources[0].Version != "1.0.0" {
		t.Error("Clone must not share resource storage")
	}

	if !lock.Delete(idA) || lock.Delete(idA) {
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

func TestNewLockUsesTheCurrentSchema(t *testing.T) {
	if NewLock("agents").SchemaVersion != LockSchemaVersion || LockSchemaVersion != 2 {
		t.Errorf("a new lock must be written at schema_version 2, got %d", NewLock("agents").SchemaVersion)
	}
}

// A v1 lockfile stays readable and is upgraded in memory, so every write produces v2.
func TestValidateAcceptsTheLegacySchemaAndUpgradeConverts(t *testing.T) {
	var lock Lock
	if err := json.Unmarshal([]byte(`{
	  "schema_version": 1,
	  "runtime": "claude-code",
	  "generated_at": "2026-05-22T10:00:00Z",
	  "resources": [{"id":"tdd","type":"skill","source":"public","version":"1.0.0",
	    "checksum":"sha256:x","requested":true,"files":[]}]
	}`), &lock); err != nil {
		t.Fatal(err)
	}
	if err := lock.Validate(); err != nil {
		t.Fatalf("a v1 lockfile must remain readable: %v", err)
	}
	if !lock.Legacy() {
		t.Error("Legacy must report the superseded schema")
	}
	lock.Upgrade()
	if lock.SchemaVersion != LockSchemaVersion || lock.Legacy() {
		t.Errorf("schema_version = %d after Upgrade", lock.SchemaVersion)
	}
	// The upgrade invents nothing: identity is assigned by an operation, not by reading.
	if lock.Project != nil {
		t.Errorf("project = %+v, want nil", lock.Project)
	}
	if len(lock.Resources) != 1 || lock.Resources[0].ID != "tdd" {
		t.Errorf("resources = %+v", lock.Resources)
	}
}

func TestValidateRejectsAnUnknownSchema(t *testing.T) {
	lock := Lock{SchemaVersion: 99}
	if err := lock.Validate(); !errs.Is(err, errs.CodeWorkspaceInvalid) {
		t.Errorf("Validate returned %v, want workspace_invalid", err)
	}
	incomplete := Lock{SchemaVersion: LockSchemaVersion, Project: &LockProject{CreatedAt: "now"}}
	if err := incomplete.Validate(); !errs.Is(err, errs.CodeWorkspaceInvalid) {
		t.Errorf("Validate returned %v for a project without an id", err)
	}
}

func TestLockCarriesProjectInstalledAtAndMigration(t *testing.T) {
	lock := NewLock("claude-code")
	lock.EnsureProject("3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512", "2026-05-22T10:00:00Z")
	lock.Project.Stack = &LockStack{Detected: []string{"go"}, Source: "user-input", Confidence: "high"}
	lock.Project.Disciplines = []string{"tdd"}
	lock.Migration = &LockMigration{
		Source: "workspace.json", SourceSchemaVersion: 2, MigratedAt: "2026-07-30T00:00:00Z",
		Extra:  map[string]json.RawMessage{"notes": json.RawMessage(`{"keep":true}`)},
		Backup: ".agents/workspace.json.migrated.bak",
	}
	lock.Upsert(LockResource{
		ID: "tdd", Type: TypeSkill, Version: "1.0.0", Checksum: "sha256:x",
		InstalledAt: "2026-05-22T10:00:00Z",
	})

	// EnsureProject never replaces an identity that already exists.
	if got := lock.EnsureProject("other", "later"); got.ID != "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512" {
		t.Errorf("EnsureProject overwrote the project identity: %+v", got)
	}

	encoded, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	var back Lock
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatal(err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("Validate returned %v", err)
	}
	if back.Project == nil || back.Project.CreatedAt != "2026-05-22T10:00:00Z" ||
		back.Project.Stack == nil || back.Project.Stack.Confidence != "high" ||
		len(back.Project.Disciplines) != 1 {
		t.Errorf("project did not round trip: %+v", back.Project)
	}
	if back.Resources[0].InstalledAt != "2026-05-22T10:00:00Z" {
		t.Errorf("installed_at did not round trip: %+v", back.Resources[0])
	}
	if back.Migration == nil || string(back.Migration.Extra["notes"]) != `{"keep":true}` {
		t.Errorf("migration did not round trip: %+v", back.Migration)
	}
}

// A proposal keeps the project's identity and history but starts with no resources, so
// planning an install can never silently drop the state the lockfile owns.
func TestProposalKeepsIdentityAndDeepCopies(t *testing.T) {
	lock := NewLock("claude-code")
	lock.EnsureProject("id-1", "2026-05-22T10:00:00Z")
	lock.Project.Disciplines = []string{"tdd"}
	lock.Migration = &LockMigration{Source: "workspace.json", Backup: ".agents/workspace.json.migrated.bak"}
	lock.Upsert(LockResource{ID: "tdd", Type: TypeSkill, Version: "1.0.0"})

	proposal := lock.Proposal("agents")
	if len(proposal.Resources) != 0 || proposal.Runtime != "agents" {
		t.Errorf("proposal = %+v", proposal)
	}
	if proposal.Project == nil || proposal.Project.ID != "id-1" || proposal.Migration == nil {
		t.Fatalf("proposal lost project state: %+v", proposal)
	}
	proposal.Project.Disciplines[0] = "mutated"
	proposal.Migration.Backup = "elsewhere"
	if lock.Project.Disciplines[0] != "tdd" || lock.Migration.Backup == "elsewhere" {
		t.Error("Proposal must not share storage with the lock it derives from")
	}

	clone := lock.Clone()
	clone.Project.Stack = &LockStack{Detected: []string{"go"}}
	clone.Migration.Extra = map[string]json.RawMessage{"x": json.RawMessage(`1`)}
	if lock.Project.Stack != nil || lock.Migration.Extra != nil {
		t.Error("Clone must not share project or migration storage")
	}
}

func TestNewProjectIDIsAUniqueUUID(t *testing.T) {
	first, err := NewProjectID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewProjectID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("two project ids must differ")
	}
	if len(first) != 36 || first[14] != '4' {
		t.Errorf("NewProjectID = %q, want a UUID v4", first)
	}
}
