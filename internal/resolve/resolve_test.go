package resolve

import (
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// load builds a single-source catalog from the given resources.
func load(t *testing.T, access model.Access, resources ...internaltest.Resource) *catalog.Catalog {
	t.Helper()
	dir := t.TempDir()
	internaltest.WriteNativeSource(t, dir, resources...)
	cat, err := catalog.NewLoader().LoadCheckout(
		internaltest.Checkout("source", dir, access, model.TrustTrusted))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	return cat
}

// merge builds a catalog spanning two sources with different visibility.
func merge(t *testing.T, public, private []internaltest.Resource) *catalog.Catalog {
	t.Helper()
	publicDir, privateDir := t.TempDir(), t.TempDir()
	internaltest.WriteNativeSource(t, publicDir, public...)
	internaltest.WriteNativeSource(t, privateDir, private...)

	home := t.TempDir()
	t.Setenv(source.HomeEnv, home)
	store, err := source.Open()
	if err != nil {
		t.Fatalf("source.Open returned %v", err)
	}
	for _, src := range []source.Source{
		{Name: "public", URL: publicDir, Access: model.AccessPublic, Trust: model.TrustTrusted},
		{Name: "personal", URL: privateDir, Access: model.AccessPrivate, Trust: model.TrustTrusted},
	} {
		if err := store.Add(src); err != nil {
			t.Fatalf("Add(%s) returned %v", src.Name, err)
		}
	}
	cat, err := catalog.NewLoader().Load(store)
	if err != nil {
		t.Fatalf("Load returned %v", err)
	}
	return cat
}

// ids lists the resolved resources by name: an order of UUIDs tells a reader nothing.
func ids(result *Result) []string {
	out := make([]string, 0, len(result.Order))
	for _, res := range result.Order {
		out = append(out, res.Name)
	}
	return out
}

func TestResolveIncludesTransitiveDependenciesInOrder(t *testing.T) {
	cat := load(t, model.AccessPublic,
		internaltest.Resource{Name: "leaf", Type: model.TypeSkill, Version: "1.0.0"},
		internaltest.Resource{
			Name: "middle", Type: model.TypeSkill, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("leaf")},
		},
		internaltest.Resource{
			Name: "root", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("middle")},
		},
	)
	result, err := New(cat, "agents").Resolve([]string{"root"})
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
	got := ids(result)
	want := []string{"leaf", "middle", "root"}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want dependencies first: %v", got, want)
		}
	}
	if !result.Requested[internaltest.IDOf("root")] || result.Requested[internaltest.IDOf("leaf")] {
		t.Errorf("Requested = %+v", result.Requested)
	}
}

func TestResolveDeduplicatesSharedDependencies(t *testing.T) {
	cat := load(t, model.AccessPublic,
		internaltest.Resource{Name: "shared", Type: model.TypeSkill, Version: "1.0.0"},
		internaltest.Resource{
			Name: "a", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("shared")},
		},
		internaltest.Resource{
			Name: "b", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("shared")},
		},
	)
	result, err := New(cat, "agents").Resolve([]string{"a", "b"})
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
	if len(result.Order) != 3 {
		t.Errorf("order = %v, want three unique resources", ids(result))
	}
}

// A mutual reference is reported, not rejected: the inherited catalog has an agent and its
// workflow naming each other, and installation has no ordering requirement (D-027).
func TestResolveReportsMutualReferencesWithoutFailing(t *testing.T) {
	cat := load(t, model.AccessPublic,
		internaltest.Resource{
			Name: "agent", Type: model.TypeAgent, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("flow")},
		},
		internaltest.Resource{
			Name: "flow", Type: model.TypeWorkflow, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("agent")},
		},
	)
	result, err := New(cat, "agents").Resolve([]string{"agent"})
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
	if len(result.Order) != 2 {
		t.Errorf("order = %v, want both resources", ids(result))
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != errs.CodeDependencyCycle {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestResolveFailsOnMissingDependency(t *testing.T) {
	cat := load(t, model.AccessPublic, internaltest.Resource{
		Name: "root", Type: model.TypeKit, Version: "1.0.0",
		Dependencies: []model.Dependency{internaltest.Dep("gone")},
	})
	_, err := New(cat, "agents").Resolve([]string{"root"})
	if errs.CodeOf(err) != errs.CodeDependencyMissing {
		t.Fatalf("err = %v, want dependency_unresolved", err)
	}
}

func TestResolveFailsOnVersionConflict(t *testing.T) {
	cat := load(t, model.AccessPublic,
		internaltest.Resource{Name: "dep", Type: model.TypeSkill, Version: "1.0.0"},
		internaltest.Resource{
			Name: "root", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("dep", "^2.0.0")},
		},
	)
	_, err := New(cat, "agents").Resolve([]string{"root"})
	if errs.CodeOf(err) != errs.CodeVersionConflict {
		t.Fatalf("err = %v, want version_conflict", err)
	}
}

func TestResolveAcceptsSatisfiedConstraint(t *testing.T) {
	cat := load(t, model.AccessPublic,
		internaltest.Resource{Name: "dep", Type: model.TypeSkill, Version: "1.4.2"},
		internaltest.Resource{
			Name: "root", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("dep", "^1.2.0")},
		},
	)
	if _, err := New(cat, "agents").Resolve([]string{"root"}); err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
}

// A private source may depend on a public one; the reverse would be uninstallable for
// anyone without credentials.
func TestResolveVisibilityRules(t *testing.T) {
	privateOK := merge(t,
		[]internaltest.Resource{{Name: "open", Type: model.TypeSkill, Version: "1.0.0"}},
		[]internaltest.Resource{{
			Name: "closed", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("open")},
		}},
	)
	if _, err := New(privateOK, "agents").Resolve([]string{"closed"}); err != nil {
		t.Errorf("private -> public must be allowed, got %v", err)
	}

	publicBad := merge(t,
		[]internaltest.Resource{{
			Name: "open", Type: model.TypeKit, Version: "1.0.0",
			Dependencies: []model.Dependency{internaltest.Dep("closed")},
		}},
		[]internaltest.Resource{{Name: "closed", Type: model.TypeSkill, Version: "1.0.0"}},
	)
	_, err := New(publicBad, "agents").Resolve([]string{"open"})
	if errs.CodeOf(err) != errs.CodeVisibilityViolation {
		t.Fatalf("err = %v, want visibility_violation", err)
	}
}

func TestResolveRejectsIncompatibleRuntime(t *testing.T) {
	cat := load(t, model.AccessPublic, internaltest.Resource{
		Name: "only-claude", Type: model.TypeSkill, Version: "1.0.0",
		Runtimes: []string{"claude-code"},
	})
	if _, err := New(cat, "opencode").Resolve([]string{"only-claude"}); errs.CodeOf(err) != errs.CodeRuntimeUnsupported {
		t.Fatalf("err = %v, want runtime_unsupported", err)
	}
	if _, err := New(cat, "claude-code").Resolve([]string{"only-claude"}); err != nil {
		t.Errorf("the declared runtime must be accepted, got %v", err)
	}
}

func TestResolvePropagatesLookupFailures(t *testing.T) {
	// Ambiguity now lives between sources: within one, a name identifies one resource.
	cat := merge(t,
		[]internaltest.Resource{{Name: "dup", Type: model.TypeWorkflow, Version: "1.0.0"}},
		[]internaltest.Resource{{
			Name: "dup", ID: "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
			Type: model.TypeWorkflow, Version: "1.0.0",
		}},
	)
	if _, err := New(cat, "agents").Resolve([]string{"dup"}); errs.CodeOf(err) != errs.CodeAmbiguousID {
		t.Fatalf("err = %v, want ambiguous_id", err)
	}
	// Qualifying it by source resolves it.
	if _, err := New(cat, "agents").Resolve([]string{"public:dup"}); err != nil {
		t.Fatalf("a qualified reference returned %v", err)
	}
	if _, err := New(cat, "agents").Resolve([]string{"nope"}); errs.CodeOf(err) != errs.CodeNotFound {
		t.Fatalf("err = %v, want not_found", err)
	}
}
