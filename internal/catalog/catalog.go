// Package catalog turns sources into a queryable view of canonical resources.
//
// A source declares its resources with a per-resource agent-kit.json (D-017). The adapter
// that synthesised manifests from the inherited Markdown layout was retired once the whole
// catalog became native (D-032, D-034), so there is exactly one way to describe a resource.
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
	byName      map[sourceName]*model.Resource
	diagnostics []model.Diagnostic
}

// New returns an empty catalog.
func New() *Catalog {
	return &Catalog{
		byID:   map[model.ID]*model.Resource{},
		byName: map[sourceName]*model.Resource{},
	}
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

// add inserts a resource, enforcing the two uniqueness rules the catalog rests on.
//
// An identity may appear only once (RF-03, D-006): a repeated UUID means two different
// resources claim to be the same one, which is an integrity error and never a candidate
// for precedence. A name may appear only once *within a source* (D-036): two sources may
// each offer a `frontend-design`, and a caller disambiguates with `<source>:<name>`.
func (c *Catalog) add(res *model.Resource) error {
	if existing, clash := c.byID[res.ID]; clash {
		return errs.New(errs.CodeRegistryIntegrity,
			"identity %s is claimed by two resources: %s and %s",
			res.ID, existing.Qualified(), res.Qualified()).
			With("id", string(res.ID)).
			With("sources", []string{existing.Source, res.Source}).
			Hint("an identity is assigned once and never reused (D-035); " +
				"give one of them a new id or retire it")
	}
	key := sourceName{source: res.Source, name: res.Name}
	if existing, clash := c.byName[key]; clash {
		return errs.New(errs.CodeRegistryIntegrity,
			"source %s offers two resources named %s: %s and %s",
			res.Source, res.Name, existing.ID.Short(), res.ID.Short()).
			With("name", res.Name).
			With("source", res.Source).
			With("ids", []string{string(existing.ID), string(res.ID)}).
			Hint("a name is unique within a source (D-036); rename one of them")
	}
	c.byID[res.ID] = res
	c.byName[key] = res
	return nil
}

// sourceName keys a resource by the pair that must be unique: its source and its name.
type sourceName struct{ source, name string }

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
		if list[i].Name != list[j].Name {
			return list[i].Name < list[j].Name
		}
		if list[i].Source != list[j].Source {
			return list[i].Source < list[j].Source
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

// Lookup resolves a user or agent reference to exactly one resource (D-036).
//
// A UUID always resolves directly: it is unambiguous by construction. A `<source>:<name>`
// reference resolves within that source, where names are unique. A bare name is searched
// across every configured source, and if more than one answers, the lookup fails with
// ambiguous_id and lists the qualified candidates. No tie is ever broken by source order.
func (c *Catalog) Lookup(ref string) (*model.Resource, error) {
	reference, err := model.ParseReference(ref)
	if err != nil {
		return nil, err
	}
	if reference.ID != "" {
		res, ok := c.byID[reference.ID]
		if !ok {
			return nil, errs.New(errs.CodeNotFound,
				"no resource has the identity %s", reference.ID)
		}
		return res, nil
	}
	if reference.Qualified() {
		res, ok := c.byName[sourceName{source: reference.Source, name: reference.Name}]
		if !ok {
			return nil, errs.New(errs.CodeNotFound,
				"source %s offers no resource named %s", reference.Source, reference.Name).
				Hint("run `agent-kits search %s` to list candidates", reference.Name)
		}
		return res, nil
	}

	var matches []*model.Resource
	for key, res := range c.byName {
		if key.name == reference.Name {
			matches = append(matches, res)
		}
	}
	switch len(matches) {
	case 0:
		return nil, errs.New(errs.CodeNotFound, "no resource is named %q", reference.Name).
			Hint("run `agent-kits search %s` to list candidates", reference.Name)
	case 1:
		return matches[0], nil
	}
	sortResources(matches)
	candidates := make([]string, 0, len(matches))
	for _, res := range matches {
		candidates = append(candidates, res.Qualified())
	}
	return nil, errs.New(errs.CodeAmbiguousID,
		"%d sources offer a resource named %q: %s",
		len(matches), reference.Name, strings.Join(candidates, ", ")).
		With("candidates", candidates).
		Hint("qualify the reference as <source>:%s, or use its id", reference.Name)
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
	for _, field := range []string{res.Name, res.Title, res.Description, string(res.ID)} {
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
	if !native {
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
