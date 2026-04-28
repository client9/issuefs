package md

import (
	"strings"
	"testing"
)

func TestEscapeCell(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"plain", "plain"},
		{"a|b", `a\|b`},
		{"a\nb", "a<br>b"},
		{"a\r\nb", "a<br>b"},
		{"a|b\nc", `a\|b<br>c`},
	}
	for _, c := range cases {
		if got := EscapeCell(c.in); got != c.want {
			t.Errorf("EscapeCell(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTable_Basic(t *testing.T) {
	got := Table([]string{"A", "B"}, [][]string{
		{"1", "2"},
		{"3", "4"},
	})
	want := "| A | B |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTable_EmptyHeaders(t *testing.T) {
	if got := Table(nil, [][]string{{"x"}}); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestTable_PadsShortRows(t *testing.T) {
	got := Table([]string{"A", "B", "C"}, [][]string{{"1"}})
	if !strings.Contains(got, "| 1 |   |   |") {
		t.Errorf("short row not padded with blanks:\n%s", got)
	}
}

func TestTable_TruncatesLongRows(t *testing.T) {
	got := Table([]string{"A"}, [][]string{{"1", "2", "3"}})
	// Should only render the first column.
	for line := range strings.SplitSeq(strings.TrimRight(got, "\n"), "\n") {
		// Each line is | <cell> | — exactly two pipes.
		if strings.Count(line, "|") != 2 {
			t.Errorf("line should have 2 pipes (1 col), got: %q", line)
		}
	}
}

func TestTable_EmptyCellsRenderAsSpace(t *testing.T) {
	got := Table([]string{"A"}, [][]string{{""}})
	if !strings.Contains(got, "|   |") {
		t.Errorf("empty cell should render as space:\n%s", got)
	}
}
