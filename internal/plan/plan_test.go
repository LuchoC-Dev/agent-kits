package plan

import (
	"os"
	"testing"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/resolve"
)

// fixture is a catalog, a project and a planner wired together.
type fixture struct {
	t       *testing.T
	catalog *catalog.Catalog
	project string
	adapter adapter.Adapter
	lock    *model.Lock
}

func newFixture(t *testing.T, resources ...internaltest.Resource) *fixture {
	t.Helper()
	sourceDir := t.TempDir()
	internaltest.WriteNativeSource(t, sourceDir, resources...)
	cat, err := catalog.NewLoader().LoadCheckout(internaltest.PublicCheckout("public", sourceDir))
	if err != nil {
		t.Fatalf("LoadCheckout returned %v", err)
	}
	runtime, err := adapter.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{
		t: t, catalog: cat, project: t.TempDir(),
		adapter: runtime, lock: model.NewLock(runtime.Name()),
	}
}

func (f *fixture) planner() *Planner {
	p := New(f.adapter, f.project, f.lock)
	p.Now = func() time.Time { return time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC) }
	return p
}

func (f *fixture) install(refs ...string) *model.Plan {
	f.t.Helper()
	result, err := resolve.New(f.catalog, f.adapter.Name()).Resolve(refs)
	if err != nil {
		f.t.Fatalf("Resolve returned %v", err)
	}
	built, err := f.planner().Install(result)
	if err != nil {
		f.t.Fatalf("Install returned %v", err)
	}
	return built
}

func actionOf(p *model.Plan, path string) model.FileAction {
	for _, change := range p.Changes {
		if change.Path == path {
			return change.Action
		}
	}
	return ""
}

func TestPlanCreatesFilesAtRuntimeDestinations(t *testing.T) {
	f := newFixture(t,
		internaltest.Resource{
			ID: "problem-framing", Type: model.TypeSkill, Version: "1.0.0",
			Files: map[string]string{"SKILL.md": "# skill\n", "references/a.md": "a\n"},
		},
		internaltest.Resource{
			ID: "context/context-builder", Type: model.TypeAgent, Version: "1.0.0",
			Files: map[string]string{"context-builder.md": "# agent\n"},
		},
		internaltest.Resource{
			ID: "context/context-building", Type: model.TypeWorkflow, Version: "1.0.0",
			Files: map[string]string{"context-building.md": "# workflow\n"},
		},
		internaltest.Resource{
			ID: "context", Type: model.TypeKit, Version: "1.0.0",
			Files: map[string]string{"pack.md": "# pack\n"},
			Dependencies: []model.Dependency{
				internaltest.Dep("problem-framing"),
				internaltest.Dep("context/context-builder"),
				internaltest.Dep("context/context-building"),
			},
		},
	)
	built := f.install("context")

	// The layout must match what the inherited kits-init flow produces.
	want := []string{
		".agents/skills/problem-framing/SKILL.md",
		".agents/skills/problem-framing/references/a.md",
		".agents/agents/context-builder.md",
		".agents/workflows/context-building.md",
		".agents/packs/context/pack.md",
	}
	for _, path := range want {
		if got := actionOf(built, path); got != model.ActionCreate {
			t.Errorf("%s action = %q, want create", path, got)
		}
	}
	if len(built.Changes) != len(want) {
		t.Errorf("changes = %d, want %d", len(built.Changes), len(want))
	}
	if built.Empty() || built.Blocked() {
		t.Errorf("empty = %v blocked = %v", built.Empty(), built.Blocked())
	}
	if built.Lock == nil || len(built.Lock.Resources) != 4 {
		t.Fatalf("proposed lock = %+v", built.Lock)
	}
	if len(built.Metadata) != 2 {
		t.Errorf("metadata = %+v", built.Metadata)
	}
}

func TestPlanIsDeterministic(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0",
		Files: map[string]string{"one.md": "1\n", "two.md": "2\n", "nested/three.md": "3\n"},
	})
	first := f.install("a")
	second := f.install("a")
	if len(first.Changes) != len(second.Changes) {
		t.Fatalf("change counts differ: %d vs %d", len(first.Changes), len(second.Changes))
	}
	for i := range first.Changes {
		if first.Changes[i] != second.Changes[i] {
			t.Fatalf("change %d differs: %+v vs %+v", i, first.Changes[i], second.Changes[i])
		}
	}
}

// D-023: the three-way comparison decides create, unchanged, update or divergent.
func TestPlanThreeWayClassification(t *testing.T) {
	const content = "new content\n"
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0",
		Files: map[string]string{"SKILL.md": content},
	})
	path := ".agents/skills/a/SKILL.md"
	newSum := fsutil.ChecksumBytes([]byte(content))

	track := func(checksum string) {
		f.lock = model.NewLock(f.adapter.Name())
		f.lock.Upsert(model.LockResource{
			ID: "a", Type: model.TypeSkill, Version: "1.0.0",
			Files: []model.LockFile{{Path: path, Checksum: checksum}},
		})
	}

	// Nothing on disk, nothing tracked.
	if got := actionOf(f.install("a"), path); got != model.ActionCreate {
		t.Errorf("absent file action = %q, want create", got)
	}

	// Identical content, already tracked.
	internaltest.WriteFile(t, f.project, path, content)
	track(newSum)
	if got := actionOf(f.install("a"), path); got != model.ActionUnchanged {
		t.Errorf("identical tracked file action = %q, want unchanged", got)
	}

	// Identical content, untracked: adopt it rather than rewrite it.
	f.lock = model.NewLock(f.adapter.Name())
	if got := actionOf(f.install("a"), path); got != model.ActionAdopt {
		t.Errorf("identical untracked file action = %q, want adopt", got)
	}

	// Tracked, untouched since install, new content available.
	const previous = "old content\n"
	internaltest.WriteFile(t, f.project, path, previous)
	track(fsutil.ChecksumBytes([]byte(previous)))
	if got := actionOf(f.install("a"), path); got != model.ActionUpdate {
		t.Errorf("tracked pristine file action = %q, want update", got)
	}

	// Tracked but locally modified: blocked.
	track("sha256:recorded-at-install-time")
	built := f.install("a")
	if got := actionOf(built, path); got != model.ActionDivergent {
		t.Errorf("locally modified file action = %q, want divergent", got)
	}
	if !built.Blocked() || built.Blockers[0].Code != errs.CodeLocalDivergence {
		t.Errorf("blockers = %+v", built.Blockers)
	}

	// An untracked file with different content is also a conflict, never a silent
	// overwrite of something the project already had.
	f.lock = model.NewLock(f.adapter.Name())
	if built = f.install("a"); !built.Blocked() {
		t.Error("an unmanaged file in the way must block")
	}
}

func TestPlanForceConvertsDivergenceToUpdate(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0",
		Files: map[string]string{"SKILL.md": "new\n"},
	})
	path := ".agents/skills/a/SKILL.md"
	internaltest.WriteFile(t, f.project, path, "local edit\n")

	result, err := resolve.New(f.catalog, f.adapter.Name()).Resolve([]string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	planner := f.planner()
	planner.Force = true
	built, err := planner.Install(result)
	if err != nil {
		t.Fatalf("Install returned %v", err)
	}
	if built.Blocked() {
		t.Fatalf("--force must not block: %+v", built.Blockers)
	}
	if got := actionOf(built, path); got != model.ActionUpdate {
		t.Errorf("action = %q, want update", got)
	}
	if len(built.Warnings) == 0 {
		t.Error("--force must warn that local changes are being overwritten")
	}
}

// Two resources that write the same destination is an integrity problem, not a race.
func TestPlanDetectsDestinationConflict(t *testing.T) {
	f := newFixture(t,
		internaltest.Resource{
			ID: "backend/feature-development", Type: model.TypeWorkflow, Version: "1.0.0",
			Files: map[string]string{"feature-development.md": "# backend\n"},
		},
		internaltest.Resource{
			ID: "frontend/feature-development", Type: model.TypeWorkflow, Version: "1.0.0",
			Files: map[string]string{"feature-development.md": "# frontend\n"},
		},
	)
	built := f.install("backend/feature-development", "frontend/feature-development")
	if !built.Blocked() {
		t.Fatal("a destination collision must block")
	}
	var found bool
	for _, blocker := range built.Blockers {
		if blocker.Code == errs.CodeDestinationConflict {
			found = true
		}
	}
	if !found {
		t.Errorf("blockers = %+v", built.Blockers)
	}
}

func TestPlanBlocksOnEmbeddedCredential(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "leaky", Type: model.TypeSkill, Version: "1.0.0",
		Files: map[string]string{"SKILL.md": "key: AKIAIOSFODNN7EXAMPLE\n"},
	})
	built := f.install("leaky")
	if !built.Blocked() || built.Blockers[0].Code != errs.CodeUnsafeContent {
		t.Fatalf("blockers = %+v", built.Blockers)
	}
}

func TestPlanReportsUpdateAndDowngradeStates(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "2.0.0",
		Files: map[string]string{"SKILL.md": "v2\n"},
	})
	f.lock.Upsert(model.LockResource{ID: "a", Type: model.TypeSkill, Version: "1.0.0"})
	if state := f.install("a").Resources[0].State; state != "update" {
		t.Errorf("state = %q, want update", state)
	}

	f.lock = model.NewLock(f.adapter.Name())
	f.lock.Upsert(model.LockResource{ID: "a", Type: model.TypeSkill, Version: "3.0.0"})
	if state := f.install("a").Resources[0].State; state != "downgrade" {
		t.Errorf("state = %q, want downgrade", state)
	}
}

// A file a resource no longer ships is pruned when the resource is re-planned.
func TestPlanPrunesRemovedFiles(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "2.0.0",
		Files: map[string]string{"SKILL.md": "v2\n"},
	})
	stale := ".agents/skills/a/OLD.md"
	staleAbs := internaltest.WriteFile(t, f.project, stale, "old\n")
	f.lock.Upsert(model.LockResource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0",
		Files: []model.LockFile{{Path: stale, Checksum: checksumOfFile(t, staleAbs)}},
	})

	built := f.install("a")
	if got := actionOf(built, stale); got != model.ActionRemove {
		t.Errorf("%s action = %q, want remove", stale, got)
	}
}

func TestRemovePlanKeepsSharedDependencies(t *testing.T) {
	f := newFixture(t,
		internaltest.Resource{
			ID: "shared", Type: model.TypeSkill, Version: "1.0.0",
			Files: map[string]string{"SKILL.md": "shared\n"},
		},
		internaltest.Resource{
			ID: "solo", Type: model.TypeSkill, Version: "1.0.0",
			Files: map[string]string{"SKILL.md": "solo\n"},
		},
		internaltest.Resource{
			ID: "kit-a", Type: model.TypeKit, Version: "1.0.0",
			Files:        map[string]string{"pack.md": "a\n"},
			Dependencies: []model.Dependency{internaltest.Dep("shared"), internaltest.Dep("solo")},
		},
		internaltest.Resource{
			ID: "kit-b", Type: model.TypeKit, Version: "1.0.0",
			Files:        map[string]string{"pack.md": "b\n"},
			Dependencies: []model.Dependency{internaltest.Dep("shared")},
		},
	)
	// Pretend both kits are installed by recording the plan they would produce.
	built := f.install("kit-a", "kit-b")
	f.lock = built.Lock
	for _, change := range built.Changes {
		internaltest.WriteFile(t, f.project, change.Path, readSource(t, change.SourcePath))
	}

	removal, err := f.planner().Remove([]string{"kit-a"}, f.catalog)
	if err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	removed := map[string]bool{}
	for _, change := range removal.Changes {
		if change.Action == model.ActionRemove {
			removed[change.Path] = true
		}
	}
	if !removed[".agents/packs/kit-a/pack.md"] {
		t.Error("the requested kit must be removed")
	}
	if !removed[".agents/skills/solo/SKILL.md"] {
		t.Error("a dependency nothing else needs must be removed")
	}
	if removed[".agents/skills/shared/SKILL.md"] {
		t.Error("a dependency another installed kit still needs must be kept")
	}
	if removed[".agents/packs/kit-b/pack.md"] {
		t.Error("an unrelated kit must be kept")
	}
}

func TestRemovePlanRejectsUnknownResource(t *testing.T) {
	f := newFixture(t, internaltest.Resource{ID: "a", Type: model.TypeSkill, Version: "1.0.0"})
	_, err := f.planner().Remove([]string{"a"}, f.catalog)
	if errs.CodeOf(err) != errs.CodeNotInstalled {
		t.Fatalf("err = %v, want not_installed", err)
	}
}

func TestRemovePlanBlocksOnModifiedFile(t *testing.T) {
	f := newFixture(t, internaltest.Resource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0",
		Files: map[string]string{"SKILL.md": "content\n"},
	})
	path := ".agents/skills/a/SKILL.md"
	internaltest.WriteFile(t, f.project, path, "edited locally\n")
	f.lock.Upsert(model.LockResource{
		ID: "a", Type: model.TypeSkill, Version: "1.0.0", Requested: true,
		Files: []model.LockFile{{Path: path, Checksum: "sha256:stale"}},
	})

	removal, err := f.planner().Remove([]string{"a"}, f.catalog)
	if err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	if !removal.Blocked() {
		t.Fatal("removing a locally modified file must block")
	}
}

func checksumOfFile(t *testing.T, path string) string {
	t.Helper()
	sum, _, err := fsutil.ChecksumFile(path)
	if err != nil {
		t.Fatalf("cannot checksum %s: %v", path, err)
	}
	return sum
}

func readSource(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	return string(content)
}
