package catalog

import (
	"path/filepath"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

// This file guards the repository's own catalog, which is also a source. It is what proves
// the inherited Markdown layout is no longer needed: the catalog loads entirely through
// native manifests, with the exact inventory D-034 approved.

// repositoryRoot is this package's directory walked back up to the module root.
const repositoryRoot = "../.."

func loadRepositoryCatalog(t *testing.T) *Catalog {
	t.Helper()
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := NewLoader().LoadCheckout(internaltest.PublicCheckout("public", root))
	if err != nil {
		t.Fatalf("the repository catalog does not load: %v", err)
	}
	return cat
}

// D-034 approved keeping every resource: 50 skills, 11 agents, 7 workflows and 7 kits.
func TestRepositoryCatalogMatchesTheApprovedInventory(t *testing.T) {
	cat := loadRepositoryCatalog(t)

	counts := map[model.Type]int{}
	for _, res := range cat.All() {
		counts[res.Type]++
	}
	want := map[model.Type]int{
		model.TypeSkill: 50, model.TypeAgent: 11, model.TypeWorkflow: 7, model.TypeKit: 7,
	}
	for typ, expected := range want {
		if counts[typ] != expected {
			t.Errorf("%s count = %d, want %d", typ, counts[typ], expected)
		}
	}
	if cat.Len() != 75 {
		t.Errorf("catalog holds %d resources, want 75", cat.Len())
	}
	if diagnostics := cat.Diagnostics(); len(diagnostics) != 0 {
		t.Errorf("loading the catalog reported %d diagnostics: %+v", len(diagnostics), diagnostics)
	}
}

// Every resource declares a real version: the synthetic 0.0.0 the retired loader assigned
// is gone, so `update` can tell one release from another (D-034).
func TestRepositoryCatalogDeclaresRealVersions(t *testing.T) {
	for _, res := range loadRepositoryCatalog(t).All() {
		if res.Version == "0.0.0" {
			t.Errorf("%s still carries the synthetic legacy version", res.ID)
		}
		if len(res.Files) == 0 {
			t.Errorf("%s declares no files", res.ID)
		}
	}
}

// D-028: two resources may not install to the same path. This is what the two
// feature-development workflows used to violate.
func TestRepositoryCatalogHasNoDestinationCollisions(t *testing.T) {
	runtime, err := adapter.Get("agents")
	if err != nil {
		t.Fatal(err)
	}
	owner := map[string]model.ID{}
	for _, res := range loadRepositoryCatalog(t).All() {
		for _, rel := range res.Files {
			dest, err := runtime.Destination(res, rel)
			if err != nil {
				t.Fatalf("%s: %v", res.ID, err)
			}
			if previous, taken := owner[dest]; taken {
				t.Errorf("%s and %s both install %s", previous, res.ID, dest)
				continue
			}
			owner[dest] = res.ID
		}
	}
}

// Every dependency resolves to a resource the catalog actually holds.
func TestRepositoryCatalogHasNoDanglingDependencies(t *testing.T) {
	cat := loadRepositoryCatalog(t)
	for _, res := range cat.All() {
		for _, dep := range res.Dependencies {
			if _, ok := cat.Get(dep.ID); !ok {
				t.Errorf("%s depends on %s, which no resource declares", res.ID, dep.ID)
			}
		}
	}
}
