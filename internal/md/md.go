// Package md assembles markdown documents from structured data. Output is
// then handed to a renderer (typically charmbracelet/glamour) per the
// project's rendering convention; see CLAUDE.md.
package md

import "strings"

// EscapeCell escapes a single markdown table cell value. Replaces "|" with
// "\|" so the bar doesn't terminate the cell, and replaces newlines with
// "<br>" so the row stays on one line (glamour renders <br> as a soft break).
// Apply to every value before assembling a table.
func EscapeCell(s string) string {
	if s == "" {
		return ""
	}
	r := strings.NewReplacer(
		`|`, `\|`,
		"\r\n", "<br>",
		"\n", "<br>",
	)
	return r.Replace(s)
}

// Table renders a markdown table from headers and rows. Caller is responsible
// for cell escaping (use EscapeCell). Empty cells render as a single space so
// the column structure is preserved.
//
// Returns "" if headers is empty. Rows shorter than headers are padded with
// blanks; rows longer are truncated.
func Table(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder

	// Header row.
	b.WriteString("|")
	for _, h := range headers {
		b.WriteString(" ")
		b.WriteString(cellOrSpace(h))
		b.WriteString(" |")
	}
	b.WriteString("\n")

	// Separator row.
	b.WriteString("|")
	for range headers {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	// Data rows.
	for _, row := range rows {
		b.WriteString("|")
		for i := range headers {
			b.WriteString(" ")
			if i < len(row) {
				b.WriteString(cellOrSpace(row[i]))
			} else {
				b.WriteString(" ")
			}
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func cellOrSpace(s string) string {
	if s == "" {
		return " "
	}
	return s
}
