package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

// table renders aligned columns without padding the last one, so output stays copyable.
type table struct {
	header []string
	rows   [][]string
}

func newTable(header ...string) *table { return &table{header: header} }

func (t *table) add(cells ...string) { t.rows = append(t.rows, cells) }

func (t *table) render(w io.Writer) {
	if len(t.rows) == 0 {
		return
	}
	widths := make([]int, len(t.header))
	for i, cell := range t.header {
		widths[i] = len(cell)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	writeRow := func(cells []string) {
		var line strings.Builder
		for i, cell := range cells {
			if i == len(cells)-1 {
				line.WriteString(cell)
				break
			}
			line.WriteString(cell)
			line.WriteString(strings.Repeat(" ", widths[i]-len(cell)+2))
		}
		fmt.Fprintln(w, strings.TrimRight(line.String(), " "))
	}
	writeRow(t.header)
	for _, row := range t.rows {
		writeRow(row)
	}
}

// truncate shortens text to max characters. It counts runes, not bytes, so a description
// in Spanish is never cut in the middle of an accented character.
func truncate(text string, max int) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	runes := []rune(collapsed)
	if len(runes) <= max {
		return collapsed
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}

// wrap breaks text into lines no wider than width, measured in runes.
func wrap(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var (
		lines   []string
		current []string
		length  int
	)
	flush := func() {
		if len(current) > 0 {
			lines = append(lines, strings.Join(current, " "))
			current, length = nil, 0
		}
	}
	for _, word := range words {
		size := len([]rune(word))
		if length > 0 && length+1+size > width {
			flush()
		}
		if length > 0 {
			length++
		}
		current = append(current, word)
		length += size
	}
	flush()
	return strings.Join(lines, "\n")
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	if commit == "" {
		return "no commit"
	}
	return commit
}

// renderPlan prints a plan compactly: one summary line, per-type counts, then only the
// detail that matters — blockers and warnings.
func (a *App) renderPlan(p *model.Plan) {
	counts := p.Counts()
	byType := map[model.Type]int{}
	for _, res := range p.Resources {
		byType[res.Type]++
	}

	fmt.Fprintf(a.Stdout, "%s · runtime %s · %d resource(s)\n",
		p.Operation, p.Runtime, len(p.Resources))

	var typeParts []string
	for _, t := range model.Types() {
		if byType[t] > 0 {
			typeParts = append(typeParts, fmt.Sprintf("%d %s", byType[t], plural(string(t), byType[t])))
		}
	}
	if len(typeParts) > 0 {
		fmt.Fprintf(a.Stdout, "  %s\n", strings.Join(typeParts, " · "))
	}

	var changeParts []string
	for _, action := range []model.FileAction{
		model.ActionCreate, model.ActionUpdate, model.ActionAdopt,
		model.ActionRemove, model.ActionUnchanged, model.ActionDivergent,
	} {
		if counts[action] > 0 {
			changeParts = append(changeParts, fmt.Sprintf("%s %d", action, counts[action]))
		}
	}
	if len(changeParts) == 0 {
		changeParts = []string{"no file changes"}
	}
	fmt.Fprintf(a.Stdout, "  %s\n", strings.Join(changeParts, " · "))

	if len(p.Metadata) > 0 {
		paths := make([]string, 0, len(p.Metadata))
		for _, change := range p.Metadata {
			paths = append(paths, change.Path)
		}
		fmt.Fprintf(a.Stdout, "  metadata %s\n", strings.Join(paths, ", "))
	}

	if requested := requestedList(p); len(requested) > 0 {
		fmt.Fprintf(a.Stdout, "  requested %s\n", strings.Join(requested, ", "))
	}

	renderDiagnostics(a.Stdout, "blocked", p.Blockers)
	renderDiagnostics(a.Stdout, "warning", p.Warnings)
}

func requestedList(p *model.Plan) []string {
	var out []string
	for _, res := range p.Resources {
		if res.Requested {
			label := string(res.ID)
			if res.State != "new" && res.State != "adopt" {
				label += " (" + res.State + ")"
			}
			out = append(out, label)
		}
	}
	sort.Strings(out)
	return out
}

func renderDiagnostics(w io.Writer, label string, list []model.Diagnostic) {
	if len(list) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s (%d):\n", label, len(list))
	for _, diagnostic := range list {
		location := diagnostic.Path
		if location == "" {
			location = diagnostic.Ref
		}
		if location == "" {
			fmt.Fprintf(w, "  [%s] %s\n", diagnostic.Code, diagnostic.Message)
			continue
		}
		fmt.Fprintf(w, "  [%s] %s: %s\n", diagnostic.Code, location, diagnostic.Message)
	}
}

func plural(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
