package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createOne runs `ifs create -t <title>` and returns the resulting path.
func createOne(t *testing.T, title string) string {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", title})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	path := strings.TrimSpace(buf.String())
	if path == "" {
		t.Fatal("create produced no path")
	}
	return path
}

func TestMove_BacklogToActive(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "fix rocket")

	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	if len(entries) != 1 {
		t.Fatalf("expected 1 backlog file, got %d", len(entries))
	}
	name := entries[0].Name()
	short := strings.Split(name, "-")[1] // 8-hex portion

	root := newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatalf("move: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "issues", "backlog", name)); !os.IsNotExist(err) {
		t.Errorf("file should be gone from backlog, err=%v", err)
	}
	newPath := filepath.Join(dir, "issues", "active", name)
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("file should be in active: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"state": "active"`) {
		t.Errorf("state field not updated:\n%s", s)
	}
	if !strings.Contains(s, `"type": "moved"`) {
		t.Errorf("moved event not appended:\n%s", s)
	}
	if !strings.Contains(s, `"from": "backlog"`) || !strings.Contains(s, `"to": "active"`) {
		t.Errorf("moved event missing from/to:\n%s", s)
	}
}

func TestMove_VerifiesAfter(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := createOne(t, "verify after move")
	short := strings.Split(filepath.Base(path), "-")[1]

	for _, target := range []string{"active", "done", "backlog"} {
		root := newRoot()
		root.SetArgs([]string{"move", short, target})
		if err := root.Execute(); err != nil {
			t.Fatalf("move %s: %v", target, err)
		}
	}

	// Find the file (it's now back in backlog) and verify.
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	if len(entries) != 1 {
		t.Fatalf("expected file back in backlog, got %d entries", len(entries))
	}
	root := newRoot()
	root.SetArgs([]string{"verify", filepath.Join(dir, "issues", "backlog", entries[0].Name())})
	if err := root.Execute(); err != nil {
		t.Errorf("verify failed after moves: %v", err)
	}
}

func TestMove_SameStateNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "no-op")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	short := strings.Split(entries[0].Name(), "-")[1]

	root := newRoot()
	root.SetArgs([]string{"move", short, "backlog"})
	if err := root.Execute(); err != nil {
		t.Errorf("same-state move should not error: %v", err)
	}
	// File still there, content unchanged in interesting ways.
	data, _ := os.ReadFile(filepath.Join(dir, "issues", "backlog", entries[0].Name()))
	if strings.Contains(string(data), `"type": "moved"`) {
		t.Errorf("no-op move should not append a moved event:\n%s", data)
	}
}

func TestMove_InvalidState(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "x")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	short := strings.Split(entries[0].Name(), "-")[1]

	root := newRoot()
	root.SetArgs([]string{"move", short, "wip"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error for invalid state")
	}
}

func TestMove_RefNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "x")

	root := newRoot()
	root.SetArgs([]string{"move", "deadbeef", "active"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected not-found error")
	}
}
