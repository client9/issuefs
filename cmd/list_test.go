package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runList_ executes `ifs list <args>` against a fresh root and returns the captured output.
func runList_(t *testing.T, args ...string) string {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"list"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

// runListErr returns the error from `ifs list <args>`.
func runListErr(t *testing.T, args ...string) error {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"list"}, args...))
	return root.Execute()
}

// seedIssues files several issues with controlled labels and assignees.
func seedIssues(t *testing.T) {
	t.Helper()
	type spec struct {
		title     string
		labels    []string
		assignees []string
		state     string // optional: defaults backlog
	}
	specs := []spec{
		{title: "Fix bug A", labels: []string{"bug"}},
		{title: "Add feature X", labels: []string{"feature"}},
		{title: "Bug and feature", labels: []string{"bug", "feature"}, assignees: []string{"alice"}},
		{title: "Assigned only", assignees: []string{"alice"}},
		{title: "Done already", labels: []string{"feature"}, state: "done"},
	}
	for _, s := range specs {
		args := []string{"create", "-t", s.title}
		for _, l := range s.labels {
			args = append(args, "-l", l)
		}
		for _, a := range s.assignees {
			args = append(args, "-a", a)
		}
		root := newRoot()
		var buf strings.Builder
		root.SetOut(&buf)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if s.state != "" && s.state != "backlog" {
			// move to the desired state via short ref
			path := strings.TrimSpace(buf.String())
			short := strings.Split(filepath.Base(path), "-")[1]
			root := newRoot()
			var mv strings.Builder
			root.SetOut(&mv)
			root.SetArgs([]string{"move", short, s.state})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestList_DefaultExcludesDone(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--format", "raw-md")
	if strings.Contains(out, "Done already") {
		t.Errorf("default list should exclude done issues:\n%s", out)
	}
	for _, want := range []string{"Fix bug A", "Add feature X", "Bug and feature", "Assigned only"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in default list:\n%s", want, out)
		}
	}
}

func TestList_StateFilter(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-s", "done", "--format", "raw-md")
	if !strings.Contains(out, "Done already") {
		t.Errorf("done-state filter should include 'Done already':\n%s", out)
	}
	if strings.Contains(out, "Fix bug A") {
		t.Errorf("done-state filter should not include backlog issues:\n%s", out)
	}
}

func TestList_StateAll(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-s", "all", "--format", "raw-md")
	for _, want := range []string{"Fix bug A", "Done already"} {
		if !strings.Contains(out, want) {
			t.Errorf("--state all should include %q:\n%s", want, out)
		}
	}
}

func TestList_LabelFilter_AND(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	// -l bug -l feature should match only "Bug and feature" (AND semantics).
	out := runList_(t, "-l", "bug", "-l", "feature", "--format", "raw-md")
	if !strings.Contains(out, "Bug and feature") {
		t.Errorf("expected 'Bug and feature':\n%s", out)
	}
	for _, exclude := range []string{"Fix bug A", "Add feature X"} {
		if strings.Contains(out, exclude) {
			t.Errorf("AND label filter should not include %q:\n%s", exclude, out)
		}
	}
}

func TestList_AssigneeFilter(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-a", "alice", "--format", "raw-md")
	for _, want := range []string{"Bug and feature", "Assigned only"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Fix bug A") {
		t.Errorf("Fix bug A is unassigned, should be excluded:\n%s", out)
	}
}

func TestList_FormatJSON(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-l", "bug", "--format", "json")
	if !json.Valid([]byte(out)) {
		t.Fatalf("invalid JSON:\n%s", out)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatal(err)
	}
	for _, raw := range arr {
		labels, _ := raw["labels"].([]any)
		hasBug := false
		for _, l := range labels {
			if l == "bug" {
				hasBug = true
			}
		}
		if !hasBug {
			t.Errorf("filtered list should only contain bug-labeled issues, got: %v", raw["labels"])
		}
	}
	if len(arr) == 0 {
		t.Errorf("expected at least one bug-labeled issue")
	}
}

func TestList_JSONAlias(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--json")
	if !json.Valid([]byte(out)) {
		t.Fatalf("--json alias should produce JSON:\n%s", out)
	}
}

func TestList_Limit(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-L", "2", "--format", "json")
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Errorf("--limit 2 should yield 2 entries, got %d", len(arr))
	}
}

func TestList_SortCreatedDesc(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--format", "json")
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatal(err)
	}
	// Created times should be non-increasing (desc).
	prev := time.Now().Add(time.Hour) // way in the future as initial bound
	for _, raw := range arr {
		ts, _ := time.Parse(time.RFC3339, raw["created"].(string))
		if ts.After(prev) {
			t.Errorf("not descending: %s after %s", ts, prev)
		}
		prev = ts
	}
}

func TestList_EmptyResult(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-l", "nonexistent", "--format", "raw-md")
	if out != "" {
		t.Errorf("no matches should produce no output, got:\n%s", out)
	}
}

func TestList_EmptyResultJSON(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := strings.TrimSpace(runList_(t, "-l", "nonexistent", "--json"))
	if out != "[]" {
		t.Errorf("no matches in JSON should be `[]`, got: %q", out)
	}
}

func TestList_NoIssuesDir(t *testing.T) {
	t.Chdir(t.TempDir())
	out := strings.TrimSpace(runList_(t, "--json"))
	if out != "[]" {
		t.Errorf("no issues dir should produce `[]` for JSON, got: %q", out)
	}
	if outRaw := runList_(t, "--format", "raw-md"); outRaw != "" {
		t.Errorf("no issues dir should produce empty for raw-md, got: %q", outRaw)
	}
}

func TestList_Since_ISO(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	out := runList_(t, "--since", tomorrow, "--format", "json")
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 0 {
		t.Errorf("--since tomorrow should exclude all of today's issues, got %d", len(arr))
	}
}

func TestList_Since_Adhoc(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--since", "yesterday", "--format", "json")
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("invalid JSON:\n%s\n%v", out, err)
	}
	if len(arr) == 0 {
		t.Errorf("--since yesterday should include today's issues")
	}
}

func TestList_Since_Bad(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "--since", "completely not a date xyz123"); err == nil {
		t.Errorf("expected --since parse error")
	}
}

func TestList_BadFormat(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "--format", "yaml"); err == nil {
		t.Errorf("expected --format error")
	}
}

func TestList_BadState(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "-s", "wip"); err == nil {
		t.Errorf("expected --state error")
	}
}

func TestList_BadSort(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "--sort", "alphabetical"); err == nil {
		t.Errorf("expected --sort error")
	}
}

func TestList_PipeInTitleEscaped(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", "title|with|pipes"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := runList_(t, "--format", "raw-md")
	if !strings.Contains(out, `title\|with\|pipes`) {
		t.Errorf("pipes should be escaped in markdown table cell:\n%s", out)
	}
}

func TestList_AsciiHasNoANSI(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--format", "ascii")
	if strings.Contains(out, "\x1b[") {
		t.Errorf("ascii output should contain no ANSI escapes, got:\n%q", out)
	}
}

func TestList_SortUpdated(t *testing.T) {
	// updated sort uses last-event timestamp; create then move shifts the issue's update time.
	t.Chdir(t.TempDir())

	// Create A first, then B.
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", "A first"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	pathA := strings.TrimSpace(buf.String())
	shortA := strings.Split(filepath.Base(pathA), "-")[1]

	time.Sleep(1100 * time.Millisecond) // ensure later timestamp (Created truncates to seconds)

	root = newRoot()
	buf.Reset()
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", "B second"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1100 * time.Millisecond)

	// Move A — gives A a more recent event timestamp than B.
	root = newRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"move", shortA, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// Default state filter (backlog+active) includes both.
	out := runList_(t, "--sort", "updated", "--format", "json")
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(arr))
	}
	// A was moved most recently, so it should sort first under --sort updated.
	if arr[0]["title"] != "A first" {
		t.Errorf("--sort updated should put most-recently-updated first; got %v", arr[0]["title"])
	}
}

func TestList_EmptyDir_Verify(t *testing.T) {
	// Sanity: list output validates against the temp dir we look at.
	dir := t.TempDir()
	t.Chdir(dir)
	seedIssues(t)
	if _, err := os.Stat(filepath.Join(dir, "issues", "backlog")); err != nil {
		t.Fatal("expected issues dir to be created by seeding")
	}
}
