package format

import (
	"strings"
	"unicode/utf8"
)

// SplitCells splits one Markdown table line into its trimmed cells.
func SplitCells(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// IsSeparatorRow reports whether the cells form a table separator row.
func IsSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

// PadTable rewrites the lines of one Markdown table into the canonical padded
// form: every cell padded to its column width, separator cells spanning the
// column width, and no alignment markers.
func PadTable(lines []string) []string {
	rows := make([][]string, len(lines))
	cols := 0
	for i, l := range lines {
		rows[i] = SplitCells(l)
		if len(rows[i]) > cols {
			cols = len(rows[i])
		}
	}
	for i := range rows {
		for len(rows[i]) < cols {
			rows[i] = append(rows[i], "")
		}
	}
	widths := make([]int, cols)
	for i := range widths {
		widths[i] = 3
	}
	for _, r := range rows {
		if IsSeparatorRow(r) {
			continue
		}
		for i, c := range r {
			if w := utf8.RuneCountInString(c); w > widths[i] {
				widths[i] = w
			}
		}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		var b strings.Builder
		b.WriteString("| ")
		sep := IsSeparatorRow(r)
		for j := 0; j < cols; j++ {
			if j > 0 {
				b.WriteString(" | ")
			}
			if sep {
				b.WriteString(strings.Repeat("-", widths[j]))
			} else {
				b.WriteString(r[j])
				b.WriteString(strings.Repeat(" ", widths[j]-utf8.RuneCountInString(r[j])))
			}
		}
		b.WriteString(" |")
		out[i] = b.String()
	}
	return out
}

// IsFence reports whether a line opens or closes a fenced code block.
func IsFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

// PadDocument rewrites every table in a Markdown document into the canonical
// padded form and leaves every other line untouched. Lines inside fenced code
// blocks are never treated as table rows.
func PadDocument(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	i := 0
	for i < len(lines) {
		if IsFence(lines[i]) {
			inFence = !inFence
		}
		if inFence || !strings.HasPrefix(lines[i], "|") {
			out = append(out, lines[i])
			i++
			continue
		}
		j := i
		for j < len(lines) && strings.HasPrefix(lines[j], "|") {
			j++
		}
		out = append(out, PadTable(lines[i:j])...)
		i = j
	}
	return strings.Join(out, "\n")
}
