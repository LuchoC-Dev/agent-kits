// Package semver implements the bounded subset of SemVer 2.0.0 approved in D-024:
// exact versions, caret, tilde and wildcard. Compound ranges are deliberately absent.
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed SemVer 2.0.0 version without build metadata.
type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Parse reads a version string. A leading "v" is tolerated; build metadata is rejected
// so that two versions never compare equal while carrying different identifiers.
func Parse(s string) (Version, error) {
	var v Version
	raw := strings.TrimSpace(s)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return v, fmt.Errorf("empty version")
	}
	if i := strings.IndexByte(raw, '+'); i >= 0 {
		return v, fmt.Errorf("build metadata is not supported: %q", s)
	}
	if i := strings.IndexByte(raw, '-'); i >= 0 {
		v.Pre = raw[i+1:]
		raw = raw[:i]
		if v.Pre == "" {
			return v, fmt.Errorf("empty prerelease in %q", s)
		}
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("expected major.minor.patch, got %q", s)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		if p == "" || (len(p) > 1 && p[0] == '0') {
			return v, fmt.Errorf("invalid numeric component %q in %q", p, s)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("invalid numeric component %q in %q", p, s)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

// MustParse is Parse for literals known to be valid.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns -1, 0 or 1. A prerelease sorts before its release (1.0.0-rc < 1.0.0).
func Compare(a, b Version) int {
	for _, pair := range [][2]int{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case a.Pre == b.Pre:
		return 0
	case a.Pre == "":
		return 1
	case b.Pre == "":
		return -1
	}
	return comparePre(a.Pre, b.Pre)
}

func comparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
		case aerr == nil:
			return -1 // numeric identifiers rank lower than alphanumeric
		case berr == nil:
			return 1
		default:
			if as[i] != bs[i] {
				if as[i] < bs[i] {
					return -1
				}
				return 1
			}
		}
	}
	switch {
	case len(as) == len(bs):
		return 0
	case len(as) < len(bs):
		return -1
	}
	return 1
}

// Operator is the kind of a constraint.
type Operator string

const (
	OpAny   Operator = "*"
	OpExact Operator = "="
	OpCaret Operator = "^"
	OpTilde Operator = "~"
)

// Constraint is a single bounded version requirement.
type Constraint struct {
	Op   Operator
	Base Version
	raw  string
}

func (c Constraint) String() string {
	if c.raw == "" {
		return string(OpAny)
	}
	return c.raw
}

// IsAny reports whether the constraint accepts every version.
func (c Constraint) IsAny() bool { return c.Op == OpAny }

// ParseConstraint reads a constraint. An empty string and "*" both mean "any", which is
// how the legacy catalog expresses dependencies that carry no version at all.
func ParseConstraint(s string) (Constraint, error) {
	raw := strings.TrimSpace(s)
	if raw == "" || raw == "*" {
		return Constraint{Op: OpAny}, nil
	}
	if strings.ContainsAny(raw, " ,|") {
		return Constraint{}, fmt.Errorf("compound ranges are not supported (D-024): %q", s)
	}
	c := Constraint{raw: raw}
	body := raw
	switch raw[0] {
	case '^':
		c.Op, body = OpCaret, raw[1:]
	case '~':
		c.Op, body = OpTilde, raw[1:]
	case '=':
		c.Op, body = OpExact, raw[1:]
	default:
		c.Op = OpExact
	}
	v, err := Parse(body)
	if err != nil {
		return Constraint{}, err
	}
	c.Base = v
	return c, nil
}

// Match reports whether v satisfies the constraint.
//
// Caret allows changes that do not modify the left-most non-zero component, so ^0.3.0
// admits 0.3.x but not 0.4.0 — the behaviour the legacy catalog's 0.x versions need.
// Tilde allows patch-level changes only.
func (c Constraint) Match(v Version) bool {
	switch c.Op {
	case OpAny:
		return true
	case OpExact:
		return Compare(v, c.Base) == 0
	case OpTilde:
		return v.Major == c.Base.Major && v.Minor == c.Base.Minor && Compare(v, c.Base) >= 0
	case OpCaret:
		if Compare(v, c.Base) < 0 {
			return false
		}
		switch {
		case c.Base.Major != 0:
			return v.Major == c.Base.Major
		case c.Base.Minor != 0:
			return v.Major == 0 && v.Minor == c.Base.Minor
		default:
			return v.Major == 0 && v.Minor == 0 && v.Patch == c.Base.Patch
		}
	}
	return false
}
