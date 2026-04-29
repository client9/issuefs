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

// --- --install-skill ---

func TestInit_InstallSkill_Project(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out := runInit_(t, "--install-skill", "project")
	target := filepath.Join(dir, ".claude", "skills", "issuefs", "SKILL.md")
	if !strings.Contains(out, "created .claude/skills/issuefs/SKILL.md") {
		t.Errorf("expected created line for project skill:\n%s", out)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("skill not installed at %s: %v", target, err)
	}
	if len(data) == 0 {
		t.Errorf("installed skill is empty")
	}
	if !strings.Contains(string(data), "name: issuefs") {
		t.Errorf("installed skill missing expected frontmatter:\n%s", data[:200])
	}
}

func TestInit_InstallSkill_IdenticalIsNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runInit_(t, "--install-skill", "project")
	out := runInit_(t, "--install-skill", "project")
	if !strings.Contains(out, "matches bundled version") {
		t.Errorf("identical re-install should report match:\n%s", out)
	}
	if strings.Contains(out, "created .claude/skills/issuefs/SKILL.md") {
		t.Errorf("identical re-install should not print 'created':\n%s", out)
	}
}

func TestInit_InstallSkill_DriftRefuses(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runInit_(t, "--install-skill", "project")
	target := filepath.Join(dir, ".claude", "skills", "issuefs", "SKILL.md")
	// Modify the installed copy.
	if err := os.WriteFile(target, []byte("modified by user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"init", "--install-skill", "project"})
	err := root.Execute()
	if err == nil {
		t.Errorf("drift without --force should error")
	}
	// Confirm file wasn't overwritten.
	data, _ := os.ReadFile(target)
	if string(data) != "modified by user\n" {
		t.Errorf("local edits should be preserved on refusal:\n%s", data)
	}
}

func TestInit_InstallSkill_ForceOverwritesDrift(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runInit_(t, "--install-skill", "project")
	target := filepath.Join(dir, ".claude", "skills", "issuefs", "SKILL.md")
	if err := os.WriteFile(target, []byte("modified by user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runInit_(t, "--install-skill", "project", "--force")
	if !strings.Contains(out, "created .claude/skills/issuefs/SKILL.md") {
		t.Errorf("--force should overwrite and print 'created':\n%s", out)
	}
	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), "name: issuefs") {
		t.Errorf("force should restore bundled content:\n%s", data[:min(200, len(data))])
	}
}

func TestInit_InstallSkill_Global(t *testing.T) {
	// Redirect HOME to a temp dir so we can test global install in isolation.
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Chdir(dir)
	out := runInit_(t, "--install-skill", "global")
	if !strings.Contains(out, "created") || !strings.Contains(out, "/.claude/skills/issuefs/SKILL.md") {
		t.Errorf("expected global install message:\n%s", out)
	}
	target := filepath.Join(home, ".claude", "skills", "issuefs", "SKILL.md")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("global skill not installed at %s: %v", target, err)
	}
}

func TestInit_InstallSkill_BothTargets(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Chdir(dir)
	out := runInit_(t, "--install-skill", "project", "--install-skill", "global")
	if !strings.Contains(out, ".claude/skills/issuefs/SKILL.md") {
		t.Errorf("expected project install line:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "issuefs", "SKILL.md")); err != nil {
		t.Errorf("project skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "issuefs", "SKILL.md")); err != nil {
		t.Errorf("global skill missing: %v", err)
	}
}

func TestInit_InstallSkill_BadTarget(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"init", "--install-skill", "everywhere"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error for invalid --install-skill target")
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
