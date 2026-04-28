package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkIssue(t *testing.T, root, state, name string) string {
	t.Helper()
	dir := filepath.Join(root, state)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolver_PrefixUnique(t *testing.T) {
	root := t.TempDir()
	want := mkIssue(t, root, "backlog", "20260428T004315Z-9f2a4b7c-fix-rockets.md")
	mkIssue(t, root, "active", "20260428T010000Z-aaaa1111-other.md")
	mkIssue(t, root, "done", "20260428T020000Z-bbbb2222-third.md")

	r, err := NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"9f2a", "9f2a4b7c", "9F2A", "20260428T004315Z-9f2a4b7c"} {
		m, err := r.Lookup(ref)
		if err != nil {
			t.Errorf("Lookup(%q): %v", ref, err)
			continue
		}
		if m.Path != want {
			t.Errorf("Lookup(%q) = %s, want %s", ref, m.Path, want)
		}
	}
}

func TestResolver_Ambiguous(t *testing.T) {
	root := t.TempDir()
	mkIssue(t, root, "backlog", "20260428T004315Z-9f2a0001-a.md")
	mkIssue(t, root, "active", "20260428T010000Z-9f2a0002-b.md")

	r, err := NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Lookup("9f2a")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguity error, got %v", err)
	}
	if !strings.Contains(err.Error(), "9f2a0001") || !strings.Contains(err.Error(), "9f2a0002") {
		t.Errorf("error should list both shorts: %v", err)
	}
	// Disambiguating with one more char works.
	if _, err := r.Lookup("9f2a000"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("'9f2a000' should still be ambiguous, got %v", err)
	}
	if m, err := r.Lookup("9f2a0001"); err != nil || m.State != "backlog" {
		t.Errorf("got %+v err=%v", m, err)
	}
}

func TestResolver_NotFound(t *testing.T) {
	r, err := NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Lookup("dead"); err == nil {
		t.Errorf("expected not-found")
	}
}

func TestResolver_PathRef(t *testing.T) {
	root := t.TempDir()
	want := mkIssue(t, root, "backlog", "20260428T004315Z-9f2a4b7c-x.md")
	r, err := NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	m, err := r.Lookup(want)
	if err != nil || m.Path != want {
		t.Errorf("got %+v err=%v", m, err)
	}
}

func TestResolver_FilenameRef(t *testing.T) {
	root := t.TempDir()
	mkIssue(t, root, "active", "20260428T004315Z-9f2a4b7c-x.md")
	r, err := NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	m, err := r.Lookup("20260428T004315Z-9f2a4b7c-x.md")
	if err != nil {
		t.Fatal(err)
	}
	if m.State != "active" {
		t.Errorf("state: %s", m.State)
	}
}

func TestResolver_SkipsMalformed(t *testing.T) {
	root := t.TempDir()
	mkIssue(t, root, "backlog", "README.md")
	mkIssue(t, root, "backlog", "not-a-timestamp-xxxx.md")
	mkIssue(t, root, "backlog", "20260428T004315Z-9f2a4b7c-good.md")
	r, err := NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(r.All()); got != 1 {
		t.Errorf("want 1 indexed file, got %d: %+v", got, r.All())
	}
}
