package frontmatter

import (
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {
	fm, body, ok := Split([]byte("---\nid: x\n---\n# Title\n"))
	if !ok {
		t.Fatal("Split did not find a frontmatter block")
	}
	if fm != "id: x" {
		t.Errorf("frontmatter = %q", fm)
	}
	if !strings.Contains(body, "# Title") {
		t.Errorf("body = %q", body)
	}

	if _, _, ok := Split([]byte("# no frontmatter\n")); ok {
		t.Error("Split reported a block where there is none")
	}
	if _, _, ok := Split([]byte("---\nid: x\n")); ok {
		t.Error("Split accepted an unterminated block")
	}

	// CRLF input must parse identically, since the catalog is edited on Windows.
	fmCRLF, _, ok := Split([]byte("---\r\nid: x\r\n---\r\nbody\r\n"))
	if !ok || fmCRLF != "id: x" {
		t.Errorf("CRLF frontmatter = %q, ok = %v", fmCRLF, ok)
	}
}

func TestParseScalarsAndQuoting(t *testing.T) {
	value, err := Parse(`
name: plain
quoted: "has: colon"
single: 'it''s here'
empty:
number: "0.3.0"
`)
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	cases := map[string]string{
		"name":   "plain",
		"quoted": "has: colon",
		"single": "it's here",
		"empty":  "",
		"number": "0.3.0",
	}
	for key, want := range cases {
		if got := value.Get(key).String(); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestParseBlockScalars(t *testing.T) {
	value, err := Parse(`
folded: >
  first line
  second line
stripped: >-
  only line
literal: |
  line one
  line two
`)
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	if got := value.Get("folded").String(); got != "first line second line" {
		t.Errorf("folded = %q", got)
	}
	if got := value.Get("stripped").String(); got != "only line" {
		t.Errorf("stripped = %q", got)
	}
	if got := value.Get("literal").String(); got != "line one\nline two" {
		t.Errorf("literal = %q", got)
	}
}

func TestParseSequences(t *testing.T) {
	value, err := Parse(`
scalars:
  - one
  - two
empty: []
inline: [a, "b, still b", c]
maps:
  - id: first
    description: the first one
  - id: second
    description: the second one
`)
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	if got := value.Get("scalars").Strings(); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("scalars = %v", got)
	}
	if got := value.Get("empty").Items(); len(got) != 0 {
		t.Errorf("empty = %v", got)
	}
	inline := value.Get("inline").Strings()
	if len(inline) != 3 || inline[1] != "b, still b" {
		t.Errorf("inline = %v", inline)
	}
	if got := value.Get("maps").StringsOf("id"); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Errorf("maps ids = %v", got)
	}
}

func TestParseNestedMapping(t *testing.T) {
	value, err := Parse(`
metadata:
  author: someone
  version: "2.0"
invocation:
  tool: Task
`)
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	if got := value.Get("metadata").Get("author").String(); got != "someone" {
		t.Errorf("metadata.author = %q", got)
	}
	if got := value.Get("invocation").Get("tool").String(); got != "Task" {
		t.Errorf("invocation.tool = %q", got)
	}
}

// The inherited workflow definitions write steps as inline mappings that nest sequences.
func TestParseInlineMappings(t *testing.T) {
	value, err := Parse(`
steps:
  - { skill: problem-framing, nn: "01", output: docs/context/01.md, depends_on: [] }
  - { skill: vision, depends_on: [problem-framing] }
`)
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	skills := value.Get("steps").StringsOf("skill")
	if len(skills) != 2 || skills[0] != "problem-framing" || skills[1] != "vision" {
		t.Fatalf("step skills = %v", skills)
	}
	first := value.Get("steps").Items()[0]
	if got := first.Get("nn").String(); got != "01" {
		t.Errorf("steps[0].nn = %q", got)
	}
	if got := first.Get("depends_on").Items(); len(got) != 0 {
		t.Errorf("steps[0].depends_on = %v", got)
	}
	second := value.Get("steps").Items()[1]
	if got := second.Get("depends_on").Strings(); len(got) != 1 || got[0] != "problem-framing" {
		t.Errorf("steps[1].depends_on = %v", got)
	}
}

func TestStringsOfAcceptsBareScalars(t *testing.T) {
	value, err := Parse("skills:\n  - discovery\n  - id: tech-decisions\n")
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	got := value.Get("skills").StringsOf("id")
	if len(got) != 2 || got[0] != "discovery" || got[1] != "tech-decisions" {
		t.Errorf("skills = %v", got)
	}
}

func TestBool(t *testing.T) {
	value, err := Parse("discipline: true\ncombinable: yes\nother: false\n")
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	if !value.Get("discipline").Bool() || !value.Get("combinable").Bool() {
		t.Error("expected true for discipline and combinable")
	}
	if value.Get("other").Bool() {
		t.Error("expected false for other")
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	for _, input := range []string{
		"just a scalar\n",
		"steps:\n  - { skill: x\n",
		"steps:\n  - { : x }\n",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) accepted malformed input", input)
		}
	}
}

func TestTextCollapsesWhitespace(t *testing.T) {
	value, err := Parse("description: >\n  one\n  two\n")
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}
	if got := value.Get("description").Text(); got != "one two" {
		t.Errorf("Text() = %q", got)
	}
}
