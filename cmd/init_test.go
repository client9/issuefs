package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runInit_(t *testing.T, args ...string) string {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"init"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestInit_FreshRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out := runInit_(t)
	for _, want := range []string{
		"created issues",
		"created issues/backlog",
		"created issues/backlog/.gitkeep",
		"created issues/active/.gitkeep",
		"created issues/done/.gitkeep",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q\nfull output:\n%s", want, out)
		}
	}
	for _, state := range []string{"backlog", "active", "done"} {
		if _, err := os.Stat(filepath.Join(dir, "issues", state, ".gitkeep")); err != nil {
			t.Errorf(".gitkeep missing in %s/: %v", state, err)
		}
	}
}

func TestInit_AlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runInit_(t)
	out := runInit_(t)
	if strings.TrimSpace(out) != "already initialized" {
		t.Errorf("expected 'already initialized', got %q", out)
	}
}

func TestInit_PartialRepair(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// Pre-create only backlog/ (no .gitkeep, no other states).
	if err := os.MkdirAll(filepath.Join(dir, "issues", "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := runInit_(t)
	// Should report adding the missing pieces but NOT report re-creating backlog/.
	if strings.Contains(out, "created issues/backlog\n") || strings.Contains(out, "created issues/backlog ") {
		// The "issues/backlog/.gitkeep" line is OK; the bare "issues/backlog" line is not.
		// Be flexible: only fail if the bare backlog dir line appears.
		lines := strings.Split(out, "\n")
		for _, l := range lines {
			if l == "created issues/backlog" {
				t.Errorf("should not re-create existing backlog/:\n%s", out)
			}
		}
	}
	if !strings.Contains(out, "created issues/active") {
		t.Errorf("should report creating active/:\n%s", out)
	}
	if !strings.Contains(out, "created issues/backlog/.gitkeep") {
		t.Errorf("should add missing .gitkeep to existing backlog/:\n%s", out)
	}
}

func TestCreate_ScaffoldsAllStates(t *testing.T) {
	// Regression: `ifs create` on a fresh repo should now produce all three
	// state dirs with .gitkeep, not just backlog/.
	dir := t.TempDir()
	t.Chdir(dir)
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", "scaffold check"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"backlog", "active", "done"} {
		if _, err := os.Stat(filepath.Join(dir, "issues", state)); err != nil {
			t.Errorf("state dir %q missing after create: %v", state, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "issues", state, ".gitkeep")); err != nil {
			t.Errorf(".gitkeep missing in %s/ after create: %v", state, err)
		}
	}
}

func TestReadOnlyVerbsDoNotScaffold(t *testing.T) {
	// Negative test for the "scaffold from write-verbs only" rule.
	// view/list/verify on a tempdir without issues/ must not create issues/.
	for _, verb := range []struct {
		name string
		args []string
	}{
		{"list", []string{"list"}},
		{"verify", []string{"verify", "/dev/null"}}, // verify on a non-issue file
	} {
		t.Run(verb.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			root := newRoot()
			var buf strings.Builder
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(verb.args)
			_ = root.Execute() // may error (e.g. verify on /dev/null), don't care
			if _, err := os.Stat(filepath.Join(dir, "issues")); !os.IsNotExist(err) {
				t.Errorf("read-only verb %q created issues/ dir (err=%v)", verb.name, err)
			}
		})
	}
}
