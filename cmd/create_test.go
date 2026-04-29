package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_Integration(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	root := newRoot()
	root.SetArgs([]string{
		"create",
		"--title", "Fix space rocket thrusters",
		"--label", "bug",
		"--label", "urgent",
		"--assignee", "@me",
		"--body", "investigate the leak",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	backlog := filepath.Join(dir, "issues", "backlog")
	entries := mdEntries(t, backlog)
	if len(entries) != 1 {
		t.Fatalf("expected 1 .md file in %s, got %d", backlog, len(entries))
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, "-fix-space-rocket-thrusters.md") {
		t.Errorf("filename %q missing slug suffix", name)
	}

	data, err := os.ReadFile(filepath.Join(backlog, name))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.NewDecoder(strings.NewReader(string(data))).Decode(&raw); err != nil {
		t.Fatalf("decode frontmatter: %v\n%s", err, data)
	}
	if raw["title"] != "Fix space rocket thrusters" {
		t.Errorf("title: %v", raw["title"])
	}
	if raw["state"] != "backlog" {
		t.Errorf("state: %v", raw["state"])
	}
	labels, _ := raw["labels"].([]any)
	if len(labels) != 2 || labels[0] != "bug" {
		t.Errorf("labels: %v", raw["labels"])
	}
	assignees, _ := raw["assignees"].([]any)
	if len(assignees) != 1 || assignees[0] != "@me" {
		t.Errorf("assignees: %v", raw["assignees"])
	}
	if !strings.Contains(string(data), "investigate the leak") {
		t.Errorf("body missing from file:\n%s", data)
	}
}

func TestCreate_RequiresTitle(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	root.SetArgs([]string{"create"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error when --title is missing")
	}
}

func TestCreate_BodyFromStdin(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	root.SetIn(strings.NewReader("body from stdin\n"))
	root.SetArgs([]string{"create", "--title", "stdin test", "--body-file", "-"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestCreate_InvalidState(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	root.SetArgs([]string{"create", "--title", "x", "--state", "wip"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error for invalid state")
	}
}
