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
