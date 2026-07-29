package semver

import "testing"

func TestParse(t *testing.T) {
	valid := map[string]Version{
		"1.2.3":      {Major: 1, Minor: 2, Patch: 3},
		"v0.1.0":     {Minor: 1},
		"0.0.0":      {},
		"1.0.0-rc.1": {Major: 1, Pre: "rc.1"},
		"10.20.30":   {Major: 10, Minor: 20, Patch: 30},
	}
	for input, want := range valid {
		got, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) returned %v", input, err)
		}
		if got != want {
			t.Errorf("Parse(%q) = %+v, want %+v", input, got, want)
		}
	}

	invalid := []string{"", "1", "1.2", "1.2.3.4", "1.2.x", "01.2.3", "1.2.3+build", "-1.2.3", "1.2.3-"}
	for _, input := range invalid {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) accepted an invalid version", input)
		}
	}
}

func TestCompareOrdersPrereleasesFirst(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc.1", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1},
		{"1.0.0-rc.2", "1.0.0-rc.10", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
	}
	for _, tc := range cases {
		got := Compare(MustParse(tc.a), MustParse(tc.b))
		if got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestConstraintMatch(t *testing.T) {
	cases := []struct {
		constraint string
		version    string
		want       bool
	}{
		{"", "9.9.9", true},
		{"*", "0.0.1", true},
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"=1.2.3", "1.2.3", true},

		// Caret keeps the left-most non-zero component, which is what the 0.x versions in
		// the inherited catalog need.
		{"^1.2.0", "1.9.0", true},
		{"^1.2.0", "2.0.0", false},
		{"^1.2.0", "1.1.0", false},
		{"^0.3.0", "0.3.9", true},
		{"^0.3.0", "0.4.0", false},
		{"^0.0.3", "0.0.3", true},
		{"^0.0.3", "0.0.4", false},

		{"~1.2.0", "1.2.9", true},
		{"~1.2.0", "1.3.0", false},
		{"~1.2.3", "1.2.2", false},
	}
	for _, tc := range cases {
		constraint, err := ParseConstraint(tc.constraint)
		if err != nil {
			t.Fatalf("ParseConstraint(%q) returned %v", tc.constraint, err)
		}
		if got := constraint.Match(MustParse(tc.version)); got != tc.want {
			t.Errorf("%q.Match(%s) = %v, want %v", tc.constraint, tc.version, got, tc.want)
		}
	}
}

func TestParseConstraintRejectsCompoundRanges(t *testing.T) {
	for _, input := range []string{">=1.0.0 <2.0.0", "1.0.0 || 2.0.0", "1.0.0,2.0.0"} {
		if _, err := ParseConstraint(input); err == nil {
			t.Errorf("ParseConstraint(%q) accepted a compound range", input)
		}
	}
}
