// Package catalog turns sources into a queryable view of canonical resources.
//
// Two source layouts are supported and may coexist in the same source: the native layout
// (per-resource agent-kit.json) and the inherited Markdown layout (D-026). Both produce
// the same in-memory model, so nothing downstream knows which one a resource came from.
package catalog

import (
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// Catalog is an aggregated, uniqueness-checked view of resources.
type Catalog struct {
	byID        map[model.ID]*model.Resource
	diagnostics []model.Diagnostic
}

// New returns an empty catalog.
func New() *Catalog {
	return &Catalog{byID: map[model.ID]*model.Resource{}}
}

// Diagnostics returns non-fatal findings collected while loading.
func (c *Catalog) Diagnostics() []model.Diagnostic {
	out := make([]model.Diagnostic, len(c.diagnostics))
	copy(out, c.diagnostics)
	return out
}

func (c *Catalog) warn(code errs.Code, ref, message string) {
	c.diagnostics = append(c.diagnostics, model.Diagnostic{Code: code, Ref: ref, Message: message})
}

// add inserts a resource, refusing to let a second resource claim the same canonical ID.
// This is RF-03: a duplicate is an integrity error, never a candidate for precedence.
func (c *Catalog) add(res *model.Resource) error {
	if existing, clash := c.byID[res.ID]; clash {
		return errs.New(errs.CodeRegistryIntegrity,
			"canonical id %s is declared by two resources: %s/%s and %s/%s",
			res.ID, existing.Source, existing.Type, res.Source, res.Type).
			With("id", string(res.ID)).
			With("sources", []string{existing.Source, res.Source}).
			Hint("ids must be globally unique (D-006); rename one resource or retire it")
	}
	c.byID[res.ID] = res
	return nil
}

// All returns every resource, ordered by type then id.
func (c *Catalog) All() []*model.Resource {
	out := make([]*model.Resource, 0, len(c.byID))
	for _, res := range c.byID {
		out = append(out, res)
	}
	sortResources(out)
	return out
}

func sortResources(list []*model.Resource) {
	rank := func(t model.Type) int {
		for i, candidate := range model.Types() {
			if candidate == t {
				return i
			}
		}
		return len(model.Types())
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Type != list[j].Type {
			return rank(list[i].Type) < rank(list[j].Type)
		}
		return list[i].ID < list[j].ID
	})
}

// Len reports how many resources the catalog holds.
func (c *Catalog) Len() int { return len(c.byID) }

// Get returns the resource with an exact canonical id.
func (c *Catalog) Get(id model.ID) (*model.Resource, bool) {
	res, ok := c.byID[id]
	return res, ok
}

// Lookup resolves a user or agent reference to exactly one resource.
//
// An exact canonical id always wins, because it is unambiguous by construction. Only when
// the reference is a bare name does the catalog consider owned resources; if more than one
// answers to that name the lookup fails with ambiguous_id and lists the candidates. No
// tie is ever broken by source order (D-006).
func (c *Catalog) Lookup(ref string) (*model.Resource, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return nil, errs.New(errs.CodeUsage, "empty resource reference")
	}
	if res, ok := c.byID[model.ID(trimmed)]; ok {
		return res, nil
	}
	var matches []*model.Resource
	for _, res := range c.byID {
		if res.ID.Matches(trimmed) {
			matches = append(matches, res)
		}
	}
	switch len(matches) {
	case 0:
		return nil, errs.New(errs.CodeNotFound, "no resource matches %q", trimmed).
			Hint("run `agent-kits search %s` to list candidates", trimmed)
	case 1:
		return matches[0], nil
	}
	sortResources(matches)
	candidates := make([]string, 0, len(matches))
	for _, res := range matches {
		candidates = append(candidates, string(res.ID))
	}
	return nil, errs.New(errs.CodeAmbiguousID,
		"%q matches %d resources: %s", trimmed, len(matches), strings.Join(candidates, ", ")).
		With("candidates", candidates).
		Hint("use a fully qualified id")
}

// Query filters the catalog. An empty query matches everything.
type Query struct {
	Text   string
	Type   model.Type
	Source string
}

// Search returns the resources matching q, ordered deterministically.
func (c *Catalog) Search(q Query) []*model.Resource {
	needle := strings.ToLower(strings.TrimSpace(q.Text))
	var out []*model.Resource
	for _, res := range c.byID {
		if q.Type != "" && res.Type != q.Type {
			continue
		}
		if q.Source != "" && res.Source != q.Source {
			continue
		}
		if needle != "" && !matchesText(res, needle) {
			continue
		}
		out = append(out, res)
	}
	sortResources(out)
	return out
}

func matchesText(res *model.Resource, needle string) bool {
	for _, field := range []string{string(res.ID), res.Name, res.Description} {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

// Loader reads catalogs from resolved source checkouts.
type Loader struct {
	Limits security.Limits
}

// NewLoader returns a loader with the default security limits.
func NewLoader() *Loader { return &Loader{Limits: security.DefaultLimits()} }

// LoadCheckout reads every resource a single source exposes.
func (l *Loader) LoadCheckout(checkout source.Checkout) (*Catalog, error) {
	cat := New()
	native, err := l.loadNative(checkout, cat)
	if err != nil {
		return nil, err
	}
	legacy, err := l.loadLegacy(checkout, cat)
	if err != nil {
		return nil, err
	}
	if !native && !legacy {
		cat.warn(errs.CodeSourceUnavailable, checkout.Source.Name,
			"source exposes no recognised catalog layout")
	}
	return cat, nil
}

// Load resolves and reads every configured source, returning the aggregated catalog.
//
// Sources that cannot be read are reported as diagnostics rather than aborting: an
// unreachable private source must not make the public catalog unusable. A duplicate
// canonical id, by contrast, invalidates the whole view.
func (l *Loader) Load(store *source.Store) (*Catalog, error) {
	checkouts, failures := store.ResolveAll()
	aggregate := New()
	for _, failure := range failures {
		aggregate.warn(errs.CodeOf(failure), "", failure.Error())
	}
	for _, checkout := range checkouts {
		cat, err := l.LoadCheckout(checkout)
		if err != nil {
			return nil, err
		}
		aggregate.diagnostics = append(aggregate.diagnostics, cat.diagnostics...)
		for _, res := range cat.All() {
			if err := aggregate.add(res); err != nil {
				return nil, err
			}
		}
	}
	return aggregate, nil
}
