package issue

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"fix space rocket thrusters", "fix-space-rocket-thrusters"},
		{"  Hello, World!  ", "hello-world"},
		{"already-slugged", "already-slugged"},
		{"multiple   spaces", "multiple-spaces"},
		{"---leading and trailing---", "leading-and-trailing"},
		{"UPPER case 123", "upper-case-123"},
		{"emoji 🚀 rocket", "emoji-rocket"},
		{"non-ascii café", "non-ascii-caf"},
		{"!!!", ""},
		{"", ""},
		{"a/b\\c:d", "a-b-c-d"},
	}
	for _, c := range cases {
		got := Slug(c.in)
		if got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugMaxLen(t *testing.T) {
	long := strings.Repeat("word ", 30)
	got := Slug(long)
	if len(got) > slugMaxLen {
		t.Errorf("len = %d, want <= %d (got %q)", len(got), slugMaxLen, got)
	}
	if strings.HasSuffix(got, "-") || strings.HasPrefix(got, "-") {
		t.Errorf("slug should not have leading/trailing dash: %q", got)
	}
}

func TestSlugCutOnBoundary(t *testing.T) {
	in := strings.Repeat("ab ", 25) // "ab ab ab ..." → "ab-ab-ab-..."
	got := Slug(in)
	if strings.Contains(got, "--") {
		t.Errorf("double dash in %q", got)
	}
	if len(got) > slugMaxLen {
		t.Errorf("len > max: %q", got)
	}
}
