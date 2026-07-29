package model

import (
	"regexp"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

// segmentPattern is one kebab-case identifier segment.
var segmentPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ID is a canonical resource identity, in one of the two forms fixed by D-019:
//
//	<name>         a resource in the global pool
//	<kit>/<name>   a resource whose identity belongs to a kit
//
// The owner segment is the kit, never the source: moving a kit between a private and a
// public source preserves every ID it owns.
type ID string

// ParseID validates s and returns it as an ID.
func ParseID(s string) (ID, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return "", errs.New(errs.CodeUsage, "empty resource id")
	}
	parts := strings.Split(raw, "/")
	if len(parts) > 2 {
		return "", errs.New(errs.CodeUsage,
			"invalid id %q: at most one owner segment is allowed (<kit>/<name>)", s)
	}
	for _, p := range parts {
		if !segmentPattern.MatchString(p) {
			return "", errs.New(errs.CodeUsage,
				"invalid id %q: segments must be lower-case kebab-case", s)
		}
	}
	return ID(raw), nil
}

// String returns the canonical text form.
func (id ID) String() string { return string(id) }

// Qualified reports whether the id carries an owner segment.
func (id ID) Qualified() bool { return strings.ContainsRune(string(id), '/') }

// Owner returns the owning kit, or "" for global-pool resources.
func (id ID) Owner() string {
	if i := strings.IndexByte(string(id), '/'); i >= 0 {
		return string(id)[:i]
	}
	return ""
}

// Name returns the last segment, which is what adapters use to build filenames.
func (id ID) Name() string {
	if i := strings.IndexByte(string(id), '/'); i >= 0 {
		return string(id)[i+1:]
	}
	return string(id)
}

// Matches reports whether ref addresses this id. A bare reference matches both a global
// id of the same name and any owned resource with that name; the caller is responsible
// for treating multiple matches as errs.CodeAmbiguousID rather than picking one.
func (id ID) Matches(ref string) bool {
	if string(id) == ref {
		return true
	}
	return !strings.ContainsRune(ref, '/') && id.Name() == ref
}
