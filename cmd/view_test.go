package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runView_(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func setupOneIssue(t *testing.T, title string, labels ...string) (dir, short string) {
	t.Helper()
	return setupOneIssueWithBody(t, title, "", labels...)
}

func setupOneIssueWithBody(t *testing.T, title, body string, labels ...string) (dir, short string) {
	t.Helper()
	dir = t.TempDir()
	t.Chdir(dir)
	args := []string{"create", "-t", title}
	for _, l := range labels {
		args = append(args, "-l", l)
	}
	if body != "" {
		args = append(args, "-b", body)
	}
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	entries := mdEntries(t, filepath.Join(dir, "issues", "backlog"))
	if len(entries) != 1 {
		t.Fatalf("expected 1 backlog entry; len=%d", len(entries))
	}
	short = strings.Split(entries[0].Name(), "-")[1]
	return dir, short
}

func TestView_RawMd_RendersMetadataTable(t *testing.T) {
	_, short := setupOneIssue(t, "fix rocket", "bug")
	out, err := runView_(t, "view", short, "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"| Field | Value |",
		"| Title | fix rocket |",
		"| State | backlog |",
		"| Labels | bug |",
		"| Assignees | _(none)_ |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("raw-md output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestView_Ascii_HasNoANSIEscapes(t *testing.T) {
	_, short := setupOneIssue(t, "no ansi please")
	out, err := runView_(t, "view", short, "--format", "ascii")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("ascii output should contain no ANSI escapes, got:\n%q", out)
	}
	if !strings.Contains(out, "no ansi please") {
		t.Errorf("ascii output should contain title, got:\n%s", out)
	}
}

func TestView_Auto_DefaultsToAsciiInStage1(t *testing.T) {
	_, short := setupOneIssue(t, "auto check")
	out, err := runView_(t, "view", short)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("auto (stage 1) should produce no ANSI escapes, got:\n%q", out)
	}
}

func TestView_PipeInTitleIsEscaped(t *testing.T) {
	_, short := setupOneIssue(t, "title|with|pipes")
	out, err := runView_(t, "view", short, "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	// Cell value should be backslash-escaped.
	if !strings.Contains(out, `title\|with\|pipes`) {
		t.Errorf("pipe should be escaped:\n%s", out)
	}
}

func TestView_NewlineInTitleIsNormalized(t *testing.T) {
	// Titles with literal newlines aren't really filed via CLI, but ensure
	// the escape helper handles them if they appear.
	dir, _ := setupOneIssue(t, "single line")
	// Hand-edit to inject a newline, then re-view.
	entries := mdEntries(t, filepath.Join(dir, "issues", "backlog"))
	path := filepath.Join(dir, "issues", "backlog", entries[0].Name())
	data, _ := os.ReadFile(path)
	updated := strings.Replace(string(data), `"title": "single line"`, `"title": "two\nlines"`, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	short := strings.Split(entries[0].Name(), "-")[1]
	out, err := runView_(t, "view", short, "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "two<br>lines") {
		t.Errorf("newline should be normalized to <br>:\n%s", out)
	}
	// And the cell shouldn't be split across rows.
	if strings.Contains(out, "lines |\n| ") && !strings.Contains(out, "two<br>lines |") {
		t.Errorf("newline broke the table row:\n%s", out)
	}
}

func TestView_RefNotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"view", "deadbeef"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error for unresolvable ref")
	}
}

func TestView_BadFormat(t *testing.T) {
	_, short := setupOneIssue(t, "x")
	if _, err := runView_(t, "view", short, "--format", "yaml"); err == nil {
		t.Errorf("expected error for invalid --format")
	}
}

// --- Stage 2: body, ANSI, TTY detection ---

func TestView_BodyRendered_RawMd(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "with body", "## Hello\n\nThis is the body content.")
	out, err := runView_(t, "view", short, "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n---\n\n## Hello") {
		t.Errorf("body not appended after horizontal rule:\n%s", out)
	}
	if !strings.Contains(out, "This is the body content.") {
		t.Errorf("body content missing:\n%s", out)
	}
}

func TestView_EmptyBody_NoBodySection(t *testing.T) {
	// With Stage 3, the document is meta + (body) + timeline. An empty body
	// should not produce a body section between meta and timeline — meta is
	// followed directly by the timeline separator, not two separators.
	_, short := setupOneIssue(t, "no body here")
	out, err := runView_(t, "view", short, "--no-events", "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	// With no body and --no-events, only metadata should remain — no separators at all.
	if strings.Contains(out, "\n---\n") {
		t.Errorf("metadata-only output should have no separators:\n%s", out)
	}
}

func TestView_BodyRendered_Ascii(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "ascii body", "Some plain prose.")
	out, err := runView_(t, "view", short, "--format", "ascii")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Some plain prose.") {
		t.Errorf("body should appear in ascii output:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("ascii should contain no ANSI escapes:\n%q", out)
	}
}

func TestView_AnsiHasEscapes(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "") // clear so ansiStyle() falls back to "dark"
	_, short := setupOneIssueWithBody(t, "ansi body", "## Heading\n\nProse with **bold**.")
	out, err := runView_(t, "view", short, "--format", "ansi")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("ansi output should contain ANSI escapes:\n%q", out)
	}
}

func TestView_AutoNoTTYDefaultsToAscii(t *testing.T) {
	// Cobra writes to a strings.Builder in tests, which isn't a *os.File, so
	// isTerminal() returns false and auto picks ascii.
	_, short := setupOneIssueWithBody(t, "auto check", "body line")
	out, err := runView_(t, "view", short)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("auto with no TTY should produce no ANSI escapes:\n%q", out)
	}
	if !strings.Contains(out, "body line") {
		t.Errorf("body should still render in ascii:\n%s", out)
	}
}

func TestView_GlamourStyleEnvOverride(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "ascii")
	_, short := setupOneIssueWithBody(t, "env override", "body")
	out, err := runView_(t, "view", short, "--format", "ansi")
	if err != nil {
		t.Fatal(err)
	}
	// With GLAMOUR_STYLE=ascii, --format ansi should respect the env and emit no escapes.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("GLAMOUR_STYLE=ascii should suppress ANSI escapes even with --format ansi:\n%q", out)
	}
}

// --- Stage 3: timeline + remaining flags ---

func TestView_TimelineRendered(t *testing.T) {
	dir, short := setupOneIssue(t, "timeline test")
	t.Chdir(dir)
	// Move it once so we have both a `filed` and a `moved` event.
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out, err := runView_(t, "view", short, "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "## Timeline") {
		t.Errorf("timeline heading missing:\n%s", out)
	}
	if !strings.Contains(out, "| filed |") {
		t.Errorf("filed event row missing:\n%s", out)
	}
	if !strings.Contains(out, "| moved |") {
		t.Errorf("moved event row missing:\n%s", out)
	}
	if !strings.Contains(out, "backlog → active") {
		t.Errorf("event details (from → to) missing:\n%s", out)
	}
}

func TestView_NoMeta(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "skip meta", "the body")
	out, err := runView_(t, "view", short, "--no-meta", "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "| Field | Value |") {
		t.Errorf("--no-meta should suppress metadata table:\n%s", out)
	}
	if !strings.Contains(out, "the body") {
		t.Errorf("body should still render:\n%s", out)
	}
	if !strings.Contains(out, "## Timeline") {
		t.Errorf("timeline should still render:\n%s", out)
	}
}

func TestView_NoEvents(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "skip events", "the body")
	out, err := runView_(t, "view", short, "--no-events", "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "## Timeline") {
		t.Errorf("--no-events should suppress timeline:\n%s", out)
	}
	if !strings.Contains(out, "| Field | Value |") {
		t.Errorf("metadata should still render:\n%s", out)
	}
}

func TestView_NoMetaNoEvents_BodyOnly(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "body only", "just the body")
	out, err := runView_(t, "view", short, "--no-meta", "--no-events", "--format", "raw-md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "| Field |") || strings.Contains(out, "## Timeline") {
		t.Errorf("only the body should render:\n%s", out)
	}
	if !strings.Contains(out, "just the body") {
		t.Errorf("body missing:\n%s", out)
	}
	// No leading separator.
	if strings.HasPrefix(out, "\n---") {
		t.Errorf("body-only output should not start with a separator:\n%s", out)
	}
}

func TestView_FormatJSON_RoundTrips(t *testing.T) {
	_, short := setupOneIssueWithBody(t, "json round trip", "with body")
	out, err := runView_(t, "view", short, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("not valid JSON:\n%s", out)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["title"] != "json round trip" {
		t.Errorf("title field wrong: %v", raw["title"])
	}
	for _, k := range []string{"id", "state", "created", "labels", "events"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("missing field %q in JSON output", k)
		}
	}
}

func TestView_FormatRaw_BytesIdentical(t *testing.T) {
	dir, short := setupOneIssueWithBody(t, "raw test", "raw body")
	out, err := runView_(t, "view", short, "--format", "raw")
	if err != nil {
		t.Fatal(err)
	}
	entries := mdEntries(t, filepath.Join(dir, "issues", "backlog"))
	want, err := os.ReadFile(filepath.Join(dir, "issues", "backlog", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if out != string(want) {
		t.Errorf("--format raw should be byte-identical to file.\ngot len=%d want len=%d", len(out), len(want))
	}
}

func TestView_TimelineNotInRawOrJSON(t *testing.T) {
	// Sanity: bypass formats shouldn't accidentally include the rendered timeline.
	_, short := setupOneIssue(t, "bypass check")
	for _, fmt := range []string{"raw", "json"} {
		out, err := runView_(t, "view", short, "--format", fmt)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "## Timeline") {
			t.Errorf("--format %s should not include rendered timeline header:\n%s", fmt, out)
		}
	}
}
