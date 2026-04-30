package cmd

import (
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

func writeTemplate(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
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

func TestList_DefaultTemplate(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t)
	if out == "" {
		t.Fatalf("expected default list output")
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("default template should render as plain ascii on a buffer:\n%q", out)
	}
	for _, want := range []string{"Fix bug A", "Add feature X", "Bug and feature", "Assigned only"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in default list:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Done already") {
		t.Errorf("default list should exclude done issues:\n%s", out)
	}
}

func TestList_StateFilter(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-s", "done")
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
	out := runList_(t, "-s", "all")
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
	out := runList_(t, "-l", "bug", "-l", "feature")
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
	out := runList_(t, "-a", "alice")
	for _, want := range []string{"Bug and feature", "Assigned only"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Fix bug A") {
		t.Errorf("Fix bug A is unassigned, should be excluded:\n%s", out)
	}
}

func TestList_Limit(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	templatePath := writeTemplate(t, "count.tmpl", "count={{ len .Entries }}")
	out := strings.TrimSpace(runList_(t, "-L", "2", "--template", templatePath))
	if out != "count=2" {
		t.Errorf("--limit 2 should yield 2 entries, got %q", out)
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

	templatePath := writeTemplate(t, "titles.tmpl", "{{ range .Entries }}{{ .Issue.Title }}\n{{ end }}")
	out := runList_(t, "--sort", "updated", "--template", templatePath)
	iA := strings.Index(out, "A first")
	iB := strings.Index(out, "B second")
	if iA == -1 || iB == -1 {
		t.Fatalf("expected both issues in output:\n%s", out)
	}
	if iA > iB {
		t.Errorf("--sort updated should put most-recently-updated first:\n%s", out)
	}
}

func TestList_EmptyResult(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "-l", "nonexistent")
	if out != "" {
		t.Errorf("no matches should produce no output, got:\n%s", out)
	}
}

func TestList_NoIssuesDir(t *testing.T) {
	t.Chdir(t.TempDir())
	out := runList_(t)
	if out != "" {
		t.Errorf("no issues dir should produce empty output, got: %q", out)
	}
}

func TestList_Since_ISO(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	out := runList_(t, "--since", tomorrow)
	if out != "" {
		t.Errorf("--since tomorrow should exclude all of today's issues, got:\n%s", out)
	}
}

func TestList_Since_Adhoc(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	out := runList_(t, "--since", "yesterday")
	if len(out) == 0 {
		t.Errorf("--since yesterday should include today's issues")
	}
}

func TestList_Since_Bad(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "--since", "completely not a date xyz123"); err == nil {
		t.Errorf("expected --since parse error")
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

func TestList_TemplateFileMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runListErr(t, "--template", "missing.tmpl"); err == nil {
		t.Fatalf("expected missing template file error")
	}
}

func TestList_TemplateParseError(t *testing.T) {
	t.Chdir(t.TempDir())
	path := writeTemplate(t, "bad.tmpl", "{{")
	if err := runListErr(t, "--template", path); err == nil {
		t.Fatalf("expected template parse error")
	}
}

func TestList_TemplateExecError(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	path := writeTemplate(t, "bad.tmpl", "{{ .Nope }}")
	if err := runListErr(t, "--template", path); err == nil {
		t.Fatalf("expected template execution error")
	}
}

func TestList_MarkdownTemplateRenders(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	raw := "**Report**\n{{ len .Entries }} issues\n"
	path := writeTemplate(t, "report.md", raw)
	out := runList_(t, "--template", path)
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("markdown template should render without ANSI in a buffer:\n%q", out)
	}
	if out == raw {
		t.Fatalf("markdown template should be rendered, not emitted raw:\n%s", out)
	}
	if !strings.Contains(out, "Report") || !strings.Contains(out, "4 issues") {
		t.Fatalf("markdown template output missing expected content:\n%s", out)
	}
}

func TestList_PlainTemplateBypassesMarkdown(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	path := writeTemplate(t, "report.tmpl", "**Report**\n{{ len .Entries }} issues\n")
	out := runList_(t, "--template", path)
	want := "**Report**\n4 issues\n"
	if out != want {
		t.Fatalf("plain-text template should bypass markdown rendering:\nwant %q\ngot  %q", want, out)
	}
}

func TestList_EscapeCell(t *testing.T) {
	t.Chdir(t.TempDir())
	seedIssues(t)
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"create", "-t", "title|with|pipes"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	path := writeTemplate(t, "escape.tmpl", "{{ range .Entries }}{{ mdCell .Issue.Title }}\n{{ end }}")
	out := runList_(t, "--template", path)
	if !strings.Contains(out, `title\|with\|pipes`) {
		t.Errorf("pipes should be escaped in template helper output:\n%s", out)
	}
}

func TestList_MarkdownExtensionDetection(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{name: "report.md", want: true},
		{name: "report.markdown", want: true},
		{name: "report.mdown", want: true},
		{name: "report.txt", want: false},
		{name: "report", want: false},
	} {
		if got := isMarkdownTemplate(tc.name); got != tc.want {
			t.Errorf("isMarkdownTemplate(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
