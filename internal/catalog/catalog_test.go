package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

func TestLoadNativeLayout(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir,
		internaltest.Resource{
			ID: "frontend-design", Type: model.TypeSkill, Version: "1.2.0",
			Description: "Design frontend interfaces",
			Files:       map[string]string{"SKILL.md": "# design\n", "references/a.md": "a\n"},
		},
		internaltest.Resource{
			ID: "frontend-designer", Type: model.TypeAgent, Version: "0.1.0",
			Dependencies: []model.Dependency{internaltest.Dep("frontend-design", "^1.0.0")},
		},
	)

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	if cat.Len() != 2 {
		t.Fatalf("Len = %d, want 2", cat.Len())
	}
	skill, ok := cat.Get("frontend-design")
	if !ok {
		t.Fatal("frontend-design was not loaded")
	}
	if skill.Version != "1.2.0" || len(skill.Files) != 2 {
		t.Errorf("skill = %+v", skill.Manifest)
	}
	if skill.Source != "public" || skill.Access != model.AccessPublic {
		t.Errorf("provenance = %s %s", skill.Source, skill.Access)
	}
}

func TestLoadNativeDiscoversFilesWhenManifestOmitsThem(t *testing.T) {
	dir := t.TempDir()
	resourceDir := filepath.Join(dir, "skills", "x")
	internaltest.WriteFile(t, resourceDir, "SKILL.md", "# x\n")
	internaltest.WriteFile(t, resourceDir, "extra/notes.md", "notes\n")
	internaltest.WriteFile(t, resourceDir, model.ManifestFilename,
		`{"schema_version":1,"id":"x","type":"skill","version":"1.0.0"}`)

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	res, ok := cat.Get("x")
	if !ok {
		t.Fatal("x was not loaded")
	}
	if len(res.Files) != 2 {
		t.Errorf("files = %v, want SKILL.md and extra/notes.md", res.Files)
	}
	for _, file := range res.Files {
		if file == model.ManifestFilename {
			t.Error("the manifest itself must not be installed")
		}
	}
}

func TestLoadRejectsMissingAndInvalidManifests(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteFile(t, dir, "skills/x/"+model.ManifestFilename,
		`{"schema_version":1,"id":"x","type":"skill","version":"1.0.0","files":["gone.md"]}`)

	_, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err == nil || errs.CodeOf(err) != errs.CodeInvalidManifest {
		t.Fatalf("err = %v", err)
	}

	other := t.TempDir()
	internaltest.WriteFile(t, other, "skills/y/"+model.ManifestFilename, `{"nope":true}`)
	if _, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", other)); err == nil {
		t.Error("an unknown manifest field was accepted")
	}
}

// RF-03: a duplicated canonical id invalidates the aggregated view.
func TestAggregateRejectsDuplicateIDs(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	shared := internaltest.Resource{ID: "shared", Type: model.TypeSkill, Version: "1.0.0"}
	internaltest.WriteNativeSource(t, first, shared)
	internaltest.WriteNativeSource(t, second, shared)

	loader := NewLoader()
	aggregate := New()
	for name, dir := range map[string]string{"public": first, "personal": second} {
		cat, err := loader.LoadCheckout(internaltest.PublicCheckout(name, dir))
		if err != nil {
			t.Fatalf("LoadCheckout(%s) returned %v", name, err)
		}
		for _, res := range cat.All() {
			err = aggregate.add(res)
			if err == nil {
				continue
			}
			if errs.CodeOf(err) != errs.CodeRegistryIntegrity {
				t.Fatalf("code = %s, want %s", errs.CodeOf(err), errs.CodeRegistryIntegrity)
			}
			return
		}
	}
	t.Fatal("a duplicated id was accepted")
}

func TestLookupPrefersExactIDAndFailsOnAmbiguity(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir,
		internaltest.Resource{ID: "backend-design", Type: model.TypeKit, Version: "1.0.0"},
		internaltest.Resource{ID: "backend-design/backend-design", Type: model.TypeWorkflow, Version: "1.0.0"},
		internaltest.Resource{ID: "backend/feature-development", Type: model.TypeWorkflow, Version: "1.0.0"},
		internaltest.Resource{ID: "frontend/feature-development", Type: model.TypeWorkflow, Version: "1.0.0"},
	)
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}

	// An exact canonical id wins over a bare-name match on an owned resource.
	res, err := cat.Lookup("backend-design")
	if err != nil {
		t.Fatalf("Lookup(backend-design) returned %v", err)
	}
	if res.Type != model.TypeKit {
		t.Errorf("Lookup(backend-design) resolved to %s", res.Type)
	}

	// A bare name owned by two kits is ambiguous and must not pick one.
	_, err = cat.Lookup("feature-development")
	if err == nil || errs.CodeOf(err) != errs.CodeAmbiguousID {
		t.Fatalf("err = %v, want ambiguous_id", err)
	}

	// The qualified form resolves.
	if _, err := cat.Lookup("backend/feature-development"); err != nil {
		t.Errorf("qualified lookup returned %v", err)
	}

	if _, err := cat.Lookup("does-not-exist"); errs.CodeOf(err) != errs.CodeNotFound {
		t.Errorf("err = %v, want not_found", err)
	}
}

func TestSearchFiltersByTypeAndText(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir,
		internaltest.Resource{ID: "alpha", Type: model.TypeSkill, Version: "1.0.0", Description: "about widgets"},
		internaltest.Resource{ID: "beta", Type: model.TypeAgent, Version: "1.0.0", Description: "about widgets"},
	)
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	if got := cat.Search(Query{}); len(got) != 2 {
		t.Errorf("empty query matched %d", len(got))
	}
	if got := cat.Search(Query{Text: "widgets"}); len(got) != 2 {
		t.Errorf("text query matched %d", len(got))
	}
	if got := cat.Search(Query{Type: model.TypeAgent}); len(got) != 1 || got[0].ID != "beta" {
		t.Errorf("type query matched %v", got)
	}
	if got := cat.Search(Query{Text: "alpha"}); len(got) != 1 || got[0].ID != "alpha" {
		t.Errorf("id query matched %v", got)
	}
	if got := cat.Search(Query{Source: "other"}); len(got) != 0 {
		t.Errorf("source filter matched %d", len(got))
	}
}

// writeLegacyCatalog builds a miniature version of the inherited Markdown layout.
func writeLegacyCatalog(t *testing.T, dir string) {
	t.Helper()
	internaltest.WriteFile(t, dir, "catalog-index.md", "---\nversion: \"0.1.0\"\n---\n# index\n")

	internaltest.WriteFile(t, dir, "skills/problem-framing/SKILL.md",
		"---\nname: problem-framing\ndescription: Define the real problem\n---\n# body\n")
	internaltest.WriteFile(t, dir, "skills/problem-framing/references/five-whys.md", "notes\n")
	internaltest.WriteFile(t, dir, "skills/tdd/SKILL.md",
		"---\nname: tdd\ndescription: Test first\ndiscipline: true\ncombinable: true\nmetadata:\n  version: \"2.0\"\n---\n# body\n")
	internaltest.WriteFile(t, dir, "skills/sdd/SKILL.md",
		"---\nname: sdd\ndescription: Spec first\ndiscipline: true\ncomposes:\n  - sdd-flow\n---\n# body\n")
	internaltest.WriteFile(t, dir, "skills/sdd-flow/SKILL.md",
		"---\nname: sdd-flow\ndescription: Orchestrates SDD\n---\n# body\n")

	internaltest.WriteFile(t, dir, "agents/artifact-validator.md",
		"---\nid: artifact-validator\ndescription: Validates artifacts\nclass: 2\ninvocation:\n  tool: Task\n---\n# body\n")

	internaltest.WriteFile(t, dir, "packs/context/pack.md",
		`---
id: context
version: "0.3.0"
description: Context pack
stack_hints: []
skills:
  - id: problem-framing
    description: phase one
agents:
  - id: context-builder
    description: orchestrator
workflows:
  - id: context-building
    description: the workflow
produces:
  - artifact: product-context
    path: docs/context/
consumes: []
---
# body
`)
	internaltest.WriteFile(t, dir, "packs/context/agents/context-builder.md",
		"---\nid: context-builder\ndescription: Orchestrates context\nskills:\n  - problem-framing\nuses_agents:\n  - artifact-validator\n---\n# body\n")
	internaltest.WriteFile(t, dir, "packs/context/workflows/context-building.md",
		"---\nid: context-building\ndescription: Six areas\nagent: context-builder\nsteps:\n  - { skill: problem-framing, nn: \"01\" }\n---\n# body\n")

	// Two kits owning a workflow of the same name: the collision the inherited catalog has.
	for _, kit := range []string{"backend", "frontend"} {
		internaltest.WriteFile(t, dir, "packs/"+kit+"/pack.md",
			"---\nid: "+kit+"\nversion: \"0.1.0\"\ndescription: "+kit+" pack\nskills: []\nworkflows:\n  - id: feature-development\n    description: build a feature\n---\n# body\n")
		internaltest.WriteFile(t, dir, "packs/"+kit+"/workflows/feature-development.md",
			"---\nid: feature-development\ndescription: "+kit+" flavour\n---\n# body for "+kit+"\n")
	}
}

func TestLoadLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	writeLegacyCatalog(t, dir)

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("legacy", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	if diagnostics := cat.Diagnostics(); len(diagnostics) != 0 {
		t.Errorf("diagnostics = %+v", diagnostics)
	}

	counts := map[model.Type]int{}
	for _, res := range cat.All() {
		counts[res.Type]++
		if !res.Legacy {
			t.Errorf("%s was not marked as legacy", res.ID)
		}
	}
	want := map[model.Type]int{
		model.TypeSkill: 4, model.TypeAgent: 2, model.TypeWorkflow: 3, model.TypeKit: 3,
	}
	for typ, expected := range want {
		if counts[typ] != expected {
			t.Errorf("%s count = %d, want %d", typ, counts[typ], expected)
		}
	}
}

func TestLegacyIdentityAndOwnership(t *testing.T) {
	dir := t.TempDir()
	writeLegacyCatalog(t, dir)
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("legacy", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}

	// A kit resolves its own workflow instead of colliding with the other kit's.
	for _, kit := range []string{"backend", "frontend"} {
		res, ok := cat.Get(model.ID(kit))
		if !ok {
			t.Fatalf("%s was not loaded", kit)
		}
		var found bool
		for _, dep := range res.Dependencies {
			if dep.ID == model.ID(kit+"/feature-development") {
				found = true
			}
		}
		if !found {
			t.Errorf("%s does not depend on %s/feature-development: %+v", kit, kit, res.Dependencies)
		}
	}
}

func TestLegacyMetadataIsPreserved(t *testing.T) {
	dir := t.TempDir()
	writeLegacyCatalog(t, dir)
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("legacy", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}

	tdd, _ := cat.Get("tdd")
	if !tdd.Traits["discipline"] || !tdd.Traits["combinable"] {
		t.Errorf("tdd traits = %+v", tdd.Traits)
	}
	// metadata.version "2.0" is normalised to a full semver.
	if tdd.Version != "2.0.0" {
		t.Errorf("tdd version = %s, want 2.0.0", tdd.Version)
	}

	skill, _ := cat.Get("problem-framing")
	if skill.Version != "0.0.0" {
		t.Errorf("a skill without a version should be 0.0.0, got %s", skill.Version)
	}
	if len(skill.Files) != 2 {
		t.Errorf("a skill directory must contribute every file: %v", skill.Files)
	}

	agent, _ := cat.Get("artifact-validator")
	if agent.Labels["class"] != "2" || agent.Labels["invocation.tool"] != "Task" {
		t.Errorf("agent labels = %+v", agent.Labels)
	}

	kit, _ := cat.Get("context")
	if len(kit.Produces) != 1 || kit.Produces[0].Artifact != "product-context" {
		t.Errorf("kit produces = %+v", kit.Produces)
	}

	sdd, _ := cat.Get("sdd")
	if len(sdd.Dependencies) != 1 || sdd.Dependencies[0].ID != "sdd-flow" {
		t.Errorf("composes was not mapped to dependencies: %+v", sdd.Dependencies)
	}
}

func TestLegacyLoaderSkipsDirectoriesWithNativeManifests(t *testing.T) {
	dir := t.TempDir()
	writeLegacyCatalog(t, dir)
	// A skill that carries both a SKILL.md and a native manifest must load once, natively.
	internaltest.WriteFile(t, dir, "skills/problem-framing/"+model.ManifestFilename,
		`{"schema_version":1,"id":"problem-framing","type":"skill","version":"3.0.0","files":["SKILL.md"]}`)

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("legacy", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	res, ok := cat.Get("problem-framing")
	if !ok {
		t.Fatal("problem-framing was not loaded")
	}
	if res.Version != "3.0.0" || res.Legacy {
		t.Errorf("the native manifest must win: version %s legacy %v", res.Version, res.Legacy)
	}
}

func TestLegacyReportsUnresolvedReferencesAsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteFile(t, dir, "packs/lonely/pack.md",
		"---\nid: lonely\nversion: \"0.1.0\"\ndescription: pack\nskills:\n  - id: nowhere\n---\n# body\n")

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("legacy", dir))
	if err != nil {
		t.Fatalf("a dangling reference must not fail the load: %v", err)
	}
	diagnostics := cat.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Code != errs.CodeDependencyMissing {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	kit, _ := cat.Get("lonely")
	if len(kit.Dependencies) != 0 {
		t.Errorf("an unresolved reference must not become a dependency: %+v", kit.Dependencies)
	}
}

func TestLoadCheckoutReportsUnrecognisedLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("nothing here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("empty", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	if cat.Len() != 0 {
		t.Errorf("Len = %d", cat.Len())
	}
	if len(cat.Diagnostics()) == 0 {
		t.Error("an unrecognised layout should be reported")
	}
}
