package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

// ID is a resource's identity: a UUID assigned once and never changed (D-035).
//
// Nothing about a resource is encoded in it. The name, the type, the version, the source
// and the kit a resource belongs to are all attributes that may change over its life; the
// identity does not. That is what makes moving a resource between kits, renaming it, or
// publishing it from a private source to a public one non-destructive: a lockfile that
// records this id keeps pointing at the same resource afterwards.
type ID string

// uuidPattern is the canonical 8-4-4-4-12 hexadecimal form, case-insensitive.
var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ParseID validates s as a resource identity and normalises it to lower case.
func ParseID(s string) (ID, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return "", errs.New(errs.CodeUsage, "empty resource id")
	}
	if !uuidPattern.MatchString(raw) {
		return "", errs.New(errs.CodeUsage,
			"invalid resource id %q: an id is a UUID, not a name", s).
			Hint("resources are referenced by name; the id only appears in manifests and lockfiles")
	}
	return ID(strings.ToLower(raw)), nil
}

// LooksLikeID reports whether ref is shaped like an identity rather than a name. It is how
// a reference is routed: a UUID resolves directly, anything else resolves by name.
func LooksLikeID(ref string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(ref))
}

// String returns the text form.
func (id ID) String() string { return string(id) }

// Short returns an abbreviated id for messages, where the full UUID adds noise.
func (id ID) Short() string {
	if len(id) >= 8 {
		return string(id)[:8]
	}
	return string(id)
}

// NewID generates a resource identity.
func NewID() (ID, error) {
	raw, err := newUUID()
	if err != nil {
		return "", errs.Wrap(errs.CodeInternal, err, "cannot generate a resource id")
	}
	return ID(raw), nil
}

// namePattern is a kebab-case install name.
var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ParseName validates a resource's install name (D-036).
//
// The name is what a person types and where the resource lands on disk. It carries no
// namespace: no kit prefix, no source prefix. Two sources may offer the same name, and a
// reference is then qualified as `<source>:<name>`.
func ParseName(s string) (string, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return "", errs.New(errs.CodeUsage, "empty resource name")
	}
	if !namePattern.MatchString(raw) {
		return "", errs.New(errs.CodeUsage,
			"invalid resource name %q: names are lower-case kebab-case, with no prefix", s)
	}
	return raw, nil
}

// Reference is a parsed resource reference, in one of the three forms of D-036.
type Reference struct {
	// ID is set when the reference was a UUID.
	ID ID
	// Source is set when the reference was qualified as `<source>:<name>`.
	Source string
	// Name is set for both qualified and bare name references.
	Name string
}

// Qualified reports whether the reference names a source explicitly.
func (r Reference) Qualified() bool { return r.Source != "" }

// String renders the reference the way the user wrote it.
func (r Reference) String() string {
	switch {
	case r.ID != "":
		return string(r.ID)
	case r.Source != "":
		return r.Source + ":" + r.Name
	}
	return r.Name
}

// ParseReference interprets how a caller addressed a resource.
func ParseReference(ref string) (Reference, error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return Reference{}, errs.New(errs.CodeUsage, "empty resource reference")
	}
	if LooksLikeID(raw) {
		id, err := ParseID(raw)
		if err != nil {
			return Reference{}, err
		}
		return Reference{ID: id}, nil
	}
	source, name, qualified := strings.Cut(raw, ":")
	if !qualified {
		parsed, err := ParseName(raw)
		if err != nil {
			return Reference{}, err
		}
		return Reference{Name: parsed}, nil
	}
	if strings.TrimSpace(source) == "" {
		return Reference{}, errs.New(errs.CodeUsage,
			"invalid reference %q: the source is empty", ref)
	}
	parsed, err := ParseName(name)
	if err != nil {
		return Reference{}, err
	}
	return Reference{Source: strings.TrimSpace(source), Name: parsed}, nil
}

// Qualify renders a resource as `<source>:<name>`, the form that always disambiguates
// between sources.
func Qualify(source, name string) string { return fmt.Sprintf("%s:%s", source, name) }
