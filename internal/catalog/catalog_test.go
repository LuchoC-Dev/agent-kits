package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

func TestLoadNativeLayout(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir,
		internaltest.Resource{
			Name: "frontend-design", Type: model.TypeSkill, Version: "1.2.0",
			Description: "Design frontend interfaces",
			Files:       map[string]string{"SKILL.md": "# design\n", "references/a.md": "a\n"},
		},
		internaltest.Resource{
			Name: "frontend-designer", Type: model.TypeAgent, Version: "0.1.0",
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
	skill, ok := cat.Get(internaltest.IDOf("frontend-design"))
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
		`{"schema_version":1,"id":"9f2c1b7a-9d84-4e11-b6f2-77c1a9e0d512","name":"x",
			`+`"type":"skill","version":"1.0.0"}`)

	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	res, ok := cat.Get("9f2c1b7a-9d84-4e11-b6f2-77c1a9e0d512")
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
		`{"schema_version":1,"id":"9f2c1b7a-9d84-4e11-b6f2-77c1a9e0d512","name":"x",
			`+`"type":"skill","version":"1.0.0","files":["gone.md"]}`)

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

// RF-03: the same identity claimed by two resources invalidates the aggregated view.
func TestAggregateRejectsADuplicatedIdentity(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	shared := internaltest.Resource{Name: "shared", Type: model.TypeSkill, Version: "1.0.0"}
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
	t.Fatal("a duplicated identity was accepted")
}

// The three reference forms of D-036, and the rule that a name is unique per source.
func TestLookupResolvesByIdentityNameAndSource(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	internaltest.WriteNativeSource(t, first,
		internaltest.Resource{Name: "frontend-design", Type: model.TypeSkill, Version: "1.0.0"},
		internaltest.Resource{Name: "only-here", Type: model.TypeSkill, Version: "1.0.0"},
	)
	internaltest.WriteNativeSource(t, second,
		// Same name, different resource: a distinct identity is what makes them distinct.
		internaltest.Resource{
			Name: "frontend-design", ID: "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
			Type: model.TypeSkill, Version: "2.0.0",
		},
	)

	loader := NewLoader()
	cat := New()
	for _, entry := range []struct{ name, dir string }{{"public", first}, {"acme", second}} {
		loaded, err := loader.LoadCheckout(internaltest.PublicCheckout(entry.name, entry.dir))
		if err != nil {
			t.Fatalf("LoadCheckout(%s) returned %v", entry.name, err)
		}
		for _, res := range loaded.All() {
			// Two sources offering the same name is legitimate: they are different resources
			// with different identities.
			if err := cat.add(res); err != nil {
				t.Fatalf("add(%s) returned %v", res.Qualified(), err)
			}
		}
	}

	// A name only one source offers resolves on its own.
	if res, err := cat.Lookup("only-here"); err != nil || res.Source != "public" {
		t.Errorf("Lookup(only-here) = %+v, %v", res, err)
	}

	// A name two sources offer is ambiguous, and the candidates are qualified.
	_, err := cat.Lookup("frontend-design")
	if errs.CodeOf(err) != errs.CodeAmbiguousID {
		t.Fatalf("err = %v, want ambiguous_id", err)
	}
	var coded *errs.Error
	if !errors.As(err, &coded) {
		t.Fatalf("err = %v", err)
	}
	candidates, _ := coded.Details["candidates"].([]string)
	if len(candidates) != 2 || candidates[0] != "acme:frontend-design" {
		t.Errorf("candidates = %v", candidates)
	}

	// Qualifying by source resolves it.
	res, err := cat.Lookup("acme:frontend-design")
	if err != nil || res.Version != "2.0.0" {
		t.Errorf("Lookup(acme:frontend-design) = %+v, %v", res, err)
	}

	// The identity always resolves, with no qualification needed.
	byID, err := cat.Lookup(string(internaltest.IDOf("only-here")))
	if err != nil || byID.Name != "only-here" {
		t.Errorf("Lookup(uuid) = %+v, %v", byID, err)
	}

	if _, err := cat.Lookup("does-not-exist"); errs.CodeOf(err) != errs.CodeNotFound {
		t.Errorf("err = %v, want not_found", err)
	}
	if _, err := cat.Lookup("acme:does-not-exist"); errs.CodeOf(err) != errs.CodeNotFound {
		t.Errorf("err = %v, want not_found", err)
	}
}

// D-036: within one source a name identifies exactly one resource.
func TestSourceMayNotOfferTwoResourcesWithTheSameName(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteFile(t, dir, "skills/one/"+model.ManifestFilename,
		`{"schema_version":1,"id":"9f2c1b7a-9d84-4e11-b6f2-77c1a9e0d512","name":"clash",`+
			`"type":"skill","version":"1.0.0","files":["SKILL.md"]}`)
	internaltest.WriteFile(t, dir, "skills/one/SKILL.md", "# one\n")
	internaltest.WriteFile(t, dir, "skills/two/"+model.ManifestFilename,
		`{"schema_version":1,"id":"3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512","name":"clash",`+
			`"type":"skill","version":"1.0.0","files":["SKILL.md"]}`)
	internaltest.WriteFile(t, dir, "skills/two/SKILL.md", "# two\n")

	_, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", dir))
	if errs.CodeOf(err) != errs.CodeRegistryIntegrity {
		t.Fatalf("err = %v, want registry_integrity_error", err)
	}
}

func TestSearchFiltersByTypeAndText(t *testing.T) {
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir,
		internaltest.Resource{Name: "alpha", Type: model.TypeSkill, Version: "1.0.0", Description: "about widgets"},
		internaltest.Resource{Name: "beta", Type: model.TypeAgent, Version: "1.0.0", Description: "about widgets"},
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
	if got := cat.Search(Query{Type: model.TypeAgent}); len(got) != 1 || got[0].Name != "beta" {
		t.Errorf("type query matched %v", got)
	}
	if got := cat.Search(Query{Text: "alpha"}); len(got) != 1 || got[0].Name != "alpha" {
		t.Errorf("name query matched %v", got)
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

// D-038: a published resource exists in the private source and in its public mirror, and
// sharing an identity there is the normal state — not a duplicate.
func TestAPublishedResourceIsNotADuplicate(t *testing.T) {
	privateDir, publicDir := t.TempDir(), t.TempDir()
	// The same resource: one identity, seen in both. The private copy is one version ahead,
	// which is what "published, then edited again" looks like.
	internaltest.WriteNativeSource(t, privateDir,
		internaltest.Resource{Name: "tdd", Type: model.TypeSkill, Version: "1.1.0"},
		internaltest.Resource{Name: "unpublished", Type: model.TypeSkill, Version: "1.0.0"},
	)
	internaltest.WriteNativeSource(t, publicDir,
		internaltest.Resource{Name: "tdd", Type: model.TypeSkill, Version: "1.0.0"},
	)

	t.Setenv(source.HomeEnv, t.TempDir())
	store, err := source.Open()
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range []source.Source{
		{Name: "private", URL: privateDir, Access: model.AccessPrivate, Trust: model.TrustTrusted},
		{
			Name: "public", URL: publicDir, Access: model.AccessPublic,
			Trust: model.TrustTrusted, Publishes: "private",
		},
	} {
		if err := store.Add(src); err != nil {
			t.Fatalf("Add(%s) returned %v", src.Name, err)
		}
	}

	cat, err := catalogOf(store)
	if err != nil {
		t.Fatalf("a published resource must not invalidate the catalog: %v", err)
	}
	if cat.Len() != 2 {
		t.Fatalf("catalog holds %d resources, want 2", cat.Len())
	}

	// The origin wins, because it is at least as new as what was published.
	res, err := cat.Lookup("tdd")
	if err != nil {
		t.Fatalf("Lookup(tdd) returned %v", err)
	}
	if res.Source != "private" || res.Version != "1.1.0" {
		t.Errorf("resolved %s %s, want the private origin at 1.1.0", res.Source, res.Version)
	}
	// And it appears once: a bare name is not ambiguous just because it was published.
	if got := cat.Search(Query{Text: "tdd"}); len(got) != 1 {
		t.Errorf("search matched %d resources, want 1", len(got))
	}
}

// Without a declared relationship, the same identity in two sources is still an error: the
// exception is a declaration, never an inference (D-038).
func TestUnrelatedSourcesStillConflictOnIdentity(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	shared := internaltest.Resource{Name: "tdd", Type: model.TypeSkill, Version: "1.0.0"}
	internaltest.WriteNativeSource(t, first, shared)
	internaltest.WriteNativeSource(t, second, shared)

	t.Setenv(source.HomeEnv, t.TempDir())
	store, err := source.Open()
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range []source.Source{
		{Name: "one", URL: first, Access: model.AccessPublic, Trust: model.TrustTrusted},
		{Name: "two", URL: second, Access: model.AccessPublic, Trust: model.TrustTrusted},
	} {
		if err := store.Add(src); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := catalogOf(store); errs.CodeOf(err) != errs.CodeRegistryIntegrity {
		t.Fatalf("err = %v, want registry_integrity_error", err)
	}
}

func catalogOf(store *source.Store) (*Catalog, error) { return NewLoader().Load(store) }
