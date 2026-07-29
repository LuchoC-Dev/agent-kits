// Package frontmatter parses the YAML subset used by the inherited Markdown catalog.
//
// It is deliberately not a general YAML implementation. It covers exactly what the
// legacy catalog uses — scalars, quoted scalars, block scalars (| and >), nested
// mappings, sequences of scalars, sequences of mappings and empty flow collections —
// and reports anything else as an error instead of guessing.
package frontmatter

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind discriminates a parsed value.
type Kind int

const (
	// KindScalar is a string value.
	KindScalar Kind = iota
	// KindSeq is an ordered list.
	KindSeq
	// KindMap is a keyed mapping.
	KindMap
)

// Value is a parsed frontmatter node.
type Value struct {
	Kind Kind
	Str  string
	Seq  []*Value

	entries map[string]*Value
	keys    []string
}

// Delimiter is the frontmatter fence.
const Delimiter = "---"

// Split separates the frontmatter block from the document body. It reports ok=false
// when the content does not open with a fence.
func Split(content []byte) (frontmatter string, body string, ok bool) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, Delimiter+"\n") {
		return "", text, false
	}
	rest := text[len(Delimiter)+1:]
	lines := strings.Split(rest, "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == Delimiter {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n"), true
		}
	}
	return "", text, false
}

var keyPattern = regexp.MustCompile(`^("[^"]*"|'[^']*'|[A-Za-z_][A-Za-z0-9_.\-]*)[ \t]*:([ \t]|$)`)

type logicalLine struct {
	indent int
	text   string
	number int
	// literal keeps the untouched line, needed to preserve block scalar indentation.
	literal string
}

// Parse reads a frontmatter block into a mapping value.
func Parse(frontmatter string) (*Value, error) {
	lines := scan(frontmatter)
	if len(lines) == 0 {
		return newMap(), nil
	}
	value, next, err := parseBlock(lines, 0, lines[0].indent)
	if err != nil {
		return nil, err
	}
	if next != len(lines) {
		return nil, fmt.Errorf("line %d: unexpected indentation", lines[next].number)
	}
	if value.Kind != KindMap {
		return nil, fmt.Errorf("frontmatter must be a mapping")
	}
	return value, nil
}

// scan turns raw text into logical lines, dropping blanks and comments and splitting
// inline sequence items so that every sequence entry is a bare marker followed by a
// child block. That normalisation is what keeps the parser small.
func scan(text string) []logicalLine {
	var out []logicalLine
	for i, raw := range strings.Split(text, "\n") {
		expanded := strings.ReplaceAll(raw, "\t", "    ")
		trimmed := strings.TrimLeft(expanded, " ")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(expanded) - len(trimmed)
		content := strings.TrimRight(trimmed, " ")

		if content == "-" {
			out = append(out, logicalLine{indent: indent, text: "-", number: i + 1, literal: raw})
			continue
		}
		if rest, ok := strings.CutPrefix(content, "- "); ok {
			out = append(out,
				logicalLine{indent: indent, text: "-", number: i + 1, literal: raw},
				logicalLine{indent: indent + 2, text: strings.TrimSpace(rest), number: i + 1, literal: raw},
			)
			continue
		}
		out = append(out, logicalLine{indent: indent, text: content, number: i + 1, literal: raw})
	}
	return out
}

func parseBlock(lines []logicalLine, i, indent int) (*Value, int, error) {
	if i >= len(lines) {
		return scalar(""), i, nil
	}
	if lines[i].text == "-" {
		return parseSeq(lines, i, indent)
	}
	if keyPattern.MatchString(lines[i].text) {
		return parseMap(lines, i, indent)
	}
	return parsePlainScalar(lines, i, indent)
}

func parseSeq(lines []logicalLine, i, indent int) (*Value, int, error) {
	value := &Value{Kind: KindSeq}
	for i < len(lines) && lines[i].indent == indent && lines[i].text == "-" {
		i++
		if i >= len(lines) || lines[i].indent <= indent {
			value.Seq = append(value.Seq, scalar(""))
			continue
		}
		child, next, err := parseBlock(lines, i, lines[i].indent)
		if err != nil {
			return nil, 0, err
		}
		value.Seq = append(value.Seq, child)
		i = next
	}
	return value, i, nil
}

func parseMap(lines []logicalLine, i, indent int) (*Value, int, error) {
	value := newMap()
	for i < len(lines) && lines[i].indent == indent {
		line := lines[i]
		if line.text == "-" {
			break
		}
		match := keyPattern.FindStringSubmatch(line.text)
		if match == nil {
			return nil, 0, fmt.Errorf("line %d: expected `key: value`, got %q", line.number, line.text)
		}
		key := unquote(match[1])
		rest := strings.TrimSpace(line.text[len(match[0])-len(match[2]):])
		i++

		switch {
		case isBlockScalarHeader(rest):
			text, next := readBlockScalar(lines, i, indent, rest)
			value.set(key, scalar(text))
			i = next

		case rest != "":
			// A plain scalar may still continue on more-indented following lines.
			text := rest
			for i < len(lines) && lines[i].indent > indent &&
				lines[i].text != "-" && !keyPattern.MatchString(lines[i].text) {
				text += " " + lines[i].text
				i++
			}
			parsed, err := parseInline(text)
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %w", line.number, err)
			}
			value.set(key, parsed)

		case i < len(lines) && lines[i].indent > indent:
			child, next, err := parseBlock(lines, i, lines[i].indent)
			if err != nil {
				return nil, 0, err
			}
			value.set(key, child)
			i = next

		case i < len(lines) && lines[i].indent == indent && lines[i].text == "-":
			// A sequence written at the same indentation as its key.
			child, next, err := parseSeq(lines, i, indent)
			if err != nil {
				return nil, 0, err
			}
			value.set(key, child)
			i = next

		default:
			value.set(key, scalar(""))
		}
	}
	return value, i, nil
}

func parsePlainScalar(lines []logicalLine, i, indent int) (*Value, int, error) {
	text := lines[i].text
	i++
	for i < len(lines) && lines[i].indent >= indent &&
		lines[i].text != "-" && !keyPattern.MatchString(lines[i].text) {
		text += " " + lines[i].text
		i++
	}
	parsed, err := parseInline(text)
	if err != nil {
		return nil, 0, err
	}
	return parsed, i, nil
}

func isBlockScalarHeader(rest string) bool {
	if rest == "" {
		return false
	}
	if rest[0] != '|' && rest[0] != '>' {
		return false
	}
	return strings.Trim(rest[1:], "-+0123456789") == ""
}

// readBlockScalar consumes an indented block introduced by | or >.
func readBlockScalar(lines []logicalLine, i, indent int, header string) (string, int) {
	literal := header[0] == '|'
	chomp := byte('c')
	if len(header) > 1 {
		switch header[1] {
		case '-':
			chomp = '-'
		case '+':
			chomp = '+'
		}
	}

	var collected []string
	for i < len(lines) && lines[i].indent > indent {
		collected = append(collected, lines[i].text)
		i++
	}

	var text string
	if literal {
		text = strings.Join(collected, "\n")
	} else {
		var parts []string
		for _, line := range collected {
			if line == "" {
				parts = append(parts, "\n")
				continue
			}
			parts = append(parts, line)
		}
		text = strings.Join(parts, " ")
		text = strings.ReplaceAll(text, " \n ", "\n\n")
	}
	switch chomp {
	case '-':
		text = strings.TrimRight(text, "\n ")
	case '+':
		// keep as collected
	default:
		text = strings.TrimRight(text, "\n ")
	}
	return text, i
}

// parseInline handles quoted scalars and flow collections. The legacy catalog writes
// workflow steps as inline mappings that nest flow sequences, so both forms recurse.
func parseInline(text string) (*Value, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return scalar(""), nil
	}
	if trimmed[0] != '[' && trimmed[0] != '{' {
		return scalar(unquote(trimmed)), nil
	}
	value, rest, err := parseFlow(trimmed)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rest) != "" {
		return nil, fmt.Errorf("trailing content after flow collection: %q", rest)
	}
	return value, nil
}

// parseFlow parses one flow node and returns the unconsumed remainder.
func parseFlow(s string) (*Value, string, error) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return nil, "", fmt.Errorf("unexpected end of flow collection")
	}
	switch s[0] {
	case '[':
		return parseFlowSeq(s[1:])
	case '{':
		return parseFlowMap(s[1:])
	}
	text, rest := readFlowScalar(s)
	return scalar(unquote(text)), rest, nil
}

func parseFlowSeq(s string) (*Value, string, error) {
	value := &Value{Kind: KindSeq}
	rest := strings.TrimLeft(s, " \t")
	if strings.HasPrefix(rest, "]") {
		return value, rest[1:], nil
	}
	for {
		item, next, err := parseFlow(rest)
		if err != nil {
			return nil, "", err
		}
		value.Seq = append(value.Seq, item)
		next = strings.TrimLeft(next, " \t")
		if next == "" {
			return nil, "", fmt.Errorf("unterminated flow sequence")
		}
		switch next[0] {
		case ',':
			rest = strings.TrimLeft(next[1:], " \t")
		case ']':
			return value, next[1:], nil
		default:
			return nil, "", fmt.Errorf("unexpected %q in flow sequence", next[0])
		}
	}
}

func parseFlowMap(s string) (*Value, string, error) {
	value := newMap()
	rest := strings.TrimLeft(s, " \t")
	if strings.HasPrefix(rest, "}") {
		return value, rest[1:], nil
	}
	for {
		candidate := readFlowKeyCandidate(rest)
		if !strings.HasSuffix(candidate, ":") {
			return nil, "", fmt.Errorf("flow mapping entry without a key: %q", rest)
		}
		key := strings.TrimSpace(strings.TrimSuffix(candidate, ":"))
		if key == "" {
			return nil, "", fmt.Errorf("flow mapping entry with an empty key: %q", rest)
		}
		item, next, err := parseFlow(rest[len(candidate):])
		if err != nil {
			return nil, "", err
		}
		value.set(unquote(key), item)

		next = strings.TrimLeft(next, " \t")
		if next == "" {
			return nil, "", fmt.Errorf("unterminated flow mapping")
		}
		switch next[0] {
		case ',':
			rest = strings.TrimLeft(next[1:], " \t")
		case '}':
			return value, next[1:], nil
		default:
			return nil, "", fmt.Errorf("unexpected %q in flow mapping", next[0])
		}
	}
}

// readFlowKeyCandidate returns the prefix of s up to and including the first colon that
// is not inside quotes, so a quoted key containing a colon stays intact.
func readFlowKeyCandidate(s string) string {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ':':
			return s[:i+1]
		case c == ',' || c == '}' || c == ']':
			return s[:i]
		}
	}
	return s
}

// readFlowScalar consumes a scalar up to the next top-level separator.
func readFlowScalar(s string) (text string, rest string) {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ',' || c == ']' || c == '}':
			return strings.TrimSpace(s[:i]), s[i:]
		}
	}
	return strings.TrimSpace(s), ""
}

func unquote(s string) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) >= 2 {
		if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			inner := trimmed[1 : len(trimmed)-1]
			inner = strings.ReplaceAll(inner, `\"`, `"`)
			return strings.ReplaceAll(inner, `\\`, `\`)
		}
		if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			return strings.ReplaceAll(trimmed[1:len(trimmed)-1], "''", "'")
		}
	}
	return trimmed
}

func scalar(s string) *Value { return &Value{Kind: KindScalar, Str: s} }

func newMap() *Value { return &Value{Kind: KindMap, entries: map[string]*Value{}} }

func (v *Value) set(key string, child *Value) {
	if v.entries == nil {
		v.entries = map[string]*Value{}
	}
	if _, exists := v.entries[key]; !exists {
		v.keys = append(v.keys, key)
	}
	v.entries[key] = child
}

// Keys returns the mapping keys in document order.
func (v *Value) Keys() []string {
	if v == nil {
		return nil
	}
	out := make([]string, len(v.keys))
	copy(out, v.keys)
	return out
}

// Get returns a child of a mapping, or nil.
func (v *Value) Get(key string) *Value {
	if v == nil || v.Kind != KindMap {
		return nil
	}
	return v.entries[key]
}

// Has reports whether a mapping declares key.
func (v *Value) Has(key string) bool { return v.Get(key) != nil }

// String returns the scalar text, or "" for collections.
func (v *Value) String() string {
	if v == nil || v.Kind != KindScalar {
		return ""
	}
	return v.Str
}

// Text is String with surrounding whitespace collapsed, for descriptions.
func (v *Value) Text() string {
	return strings.Join(strings.Fields(v.String()), " ")
}

// Bool interprets a scalar as a YAML boolean.
func (v *Value) Bool() bool {
	switch strings.ToLower(v.String()) {
	case "true", "yes", "on":
		return true
	}
	return false
}

// Items returns the elements of a sequence. A scalar is treated as a one-element
// sequence, which is how the legacy catalog occasionally writes single dependencies.
func (v *Value) Items() []*Value {
	if v == nil {
		return nil
	}
	switch v.Kind {
	case KindSeq:
		return v.Seq
	case KindScalar:
		if v.Str == "" {
			return nil
		}
		return []*Value{v}
	}
	return nil
}

// Strings returns the sequence as scalar strings, skipping empty entries.
func (v *Value) Strings() []string {
	var out []string
	for _, item := range v.Items() {
		if s := item.String(); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// StringsOf returns, for a sequence of mappings, the value of key in each element. An
// element that is a bare scalar contributes itself, so both of these parse alike:
//
//	skills:
//	  - discovery
//	  - id: discovery
func (v *Value) StringsOf(key string) []string {
	var out []string
	for _, item := range v.Items() {
		switch item.Kind {
		case KindScalar:
			if item.Str != "" {
				out = append(out, item.Str)
			}
		case KindMap:
			if child := item.Get(key); child != nil && child.String() != "" {
				out = append(out, child.String())
			}
		}
	}
	return out
}
