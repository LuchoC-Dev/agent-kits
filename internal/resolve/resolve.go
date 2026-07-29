// Package resolve expands a set of references into the complete, ordered set of
// resources an operation must install.
//
// Resolution is read-only and total: it either returns a closed dependency set with every
// constraint satisfied, or it fails. It never drops an unsatisfiable dependency to make
// progress (RF-05).
package resolve

import (
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

// Result is a closed dependency set, ordered dependencies-first where the graph allows.
type Result struct {
	// Order lists resources dependencies-first, so installing in order never leaves a
	// resource on disk before what it needs.
	Order []*model.Resource
	// Requested marks the resources named explicitly by the caller.
	Requested map[model.ID]bool
	// Refs is the original reference list, preserved for plan output.
	Refs []string
	// Diagnostics reports non-fatal findings, notably mutual references.
	Diagnostics []model.Diagnostic
}

// IDs returns the resolved ids in installation order.
func (r *Result) IDs() []model.ID {
	out := make([]model.ID, 0, len(r.Order))
	for _, res := range r.Order {
		out = append(out, res.ID)
	}
	return out
}

// Resolver expands references against a catalog.
type Resolver struct {
	Catalog *catalog.Catalog
	// Runtime, when set, rejects resources that declare incompatibility with it.
	Runtime string
}

// New returns a resolver over cat.
func New(cat *catalog.Catalog, runtime string) *Resolver {
	return &Resolver{Catalog: cat, Runtime: runtime}
}

type mark int

const (
	unvisited mark = iota
	visiting
	visited
)

// Resolve expands refs into a closed dependency set.
func (r *Resolver) Resolve(refs []string) (*Result, error) {
	result := &Result{Requested: map[model.ID]bool{}, Refs: refs}

	roots := make([]*model.Resource, 0, len(refs))
	for _, ref := range refs {
		res, err := r.Catalog.Lookup(ref)
		if err != nil {
			return nil, err
		}
		if result.Requested[res.ID] {
			continue
		}
		result.Requested[res.ID] = true
		roots = append(roots, res)
	}

	marks := map[model.ID]mark{}
	var order []*model.Resource
	var path []model.ID

	var visit func(res *model.Resource) error
	visit = func(res *model.Resource) error {
		switch marks[res.ID] {
		case visited:
			return nil
		case visiting:
			// A cycle is reported, not rejected. Resolution computes a *set* of resources
			// and installation writes independent files, so nothing here depends on
			// ordering. Mutual references are also legitimate in the inherited catalog: a
			// workflow names its orchestrating agent and that agent names the workflow it
			// runs (D-027).
			result.Diagnostics = append(result.Diagnostics, cycleDiagnostic(append(path, res.ID)))
			return nil
		}
		marks[res.ID] = visiting
		path = append(path, res.ID)

		if err := r.checkRuntime(res); err != nil {
			return err
		}
		for _, dep := range sortedDeps(res.Dependencies) {
			child, ok := r.Catalog.Get(dep.ID)
			if !ok {
				return errs.New(errs.CodeDependencyMissing,
					"%s depends on %s, which is not in any configured source", res.ID, dep.ID).
					With("resource", string(res.ID)).
					With("dependency", string(dep.ID)).
					Hint("sync the source that provides %s, or retire the dependency", dep.ID)
			}
			if err := checkConstraint(res, child, dep); err != nil {
				return err
			}
			if err := checkVisibility(res, child); err != nil {
				return err
			}
			if err := visit(child); err != nil {
				return err
			}
		}
		path = path[:len(path)-1]
		marks[res.ID] = visited
		order = append(order, res)
		return nil
	}

	for _, root := range roots {
		if err := visit(root); err != nil {
			return nil, err
		}
	}
	result.Order = order
	return result, nil
}

func sortedDeps(deps []model.Dependency) []model.Dependency {
	out := make([]model.Dependency, len(deps))
	copy(out, deps)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// cycleDiagnostic describes a reference cycle, trimmed to the cycle itself.
func cycleDiagnostic(path []model.ID) model.Diagnostic {
	last := path[len(path)-1]
	start := 0
	for i, id := range path {
		if id == last {
			start = i
			break
		}
	}
	labels := make([]string, 0, len(path)-start)
	for _, id := range path[start:] {
		labels = append(labels, string(id))
	}
	return model.Diagnostic{
		Code:    errs.CodeDependencyCycle,
		Ref:     string(last),
		Message: "mutual reference: " + strings.Join(labels, " -> "),
	}
}

// checkConstraint validates the parent's version requirement against the single resource
// that owns the dependency's id. Because ids are globally unique there is no version to
// select, so an unsatisfied constraint is a hard conflict rather than a search failure.
func checkConstraint(parent, child *model.Resource, dep model.Dependency) error {
	constraint, err := dep.Constraint()
	if err != nil {
		return errs.Wrap(errs.CodeInvalidManifest, err,
			"%s declares an invalid constraint for %s", parent.ID, dep.ID)
	}
	if constraint.IsAny() {
		return nil
	}
	if constraint.Match(child.SemVersion()) {
		return nil
	}
	return errs.New(errs.CodeVersionConflict,
		"%s requires %s %s, but the catalog provides %s",
		parent.ID, dep.ID, constraint, child.Version).
		With("resource", string(parent.ID)).
		With("dependency", string(dep.ID)).
		With("required", constraint.String()).
		With("available", child.Version)
}

// checkVisibility enforces the rule from 02-architecture-direction.md §3: a private
// source may depend on a public one, never the reverse. A public resource that needed
// private content would be uninstallable for anyone without those credentials.
func checkVisibility(parent, child *model.Resource) error {
	if parent.Access == model.AccessPublic && child.Access == model.AccessPrivate {
		return errs.New(errs.CodeVisibilityViolation,
			"public resource %s depends on private resource %s", parent.ID, child.ID).
			With("resource", string(parent.ID)).
			With("dependency", string(child.ID)).
			Hint("move %s to a public source or make %s private", child.ID, parent.ID)
	}
	return nil
}

func (r *Resolver) checkRuntime(res *model.Resource) error {
	if r.Runtime == "" || res.SupportsRuntime(r.Runtime) {
		return nil
	}
	return errs.New(errs.CodeRuntimeUnsupported,
		"%s does not support runtime %q (declares %s)",
		res.ID, r.Runtime, strings.Join(res.Runtimes, ", ")).
		With("resource", string(res.ID)).
		With("runtime", r.Runtime)
}
