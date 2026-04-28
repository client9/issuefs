package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickg/issuefs/internal/issue"
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

// --- Sync semantics: directory and frontmatter should always converge ---

// gitMove simulates `git mv` (file moves, frontmatter doesn't change).
func gitMove(t *testing.T, dir, fromState, toState, name string) string {
	t.Helper()
	src := filepath.Join(dir, "issues", fromState, name)
	dstDir := filepath.Join(dir, "issues", toState)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, name)
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

func parseIssueFile(t *testing.T, path string) *issue.Issue {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	iss, err := issue.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return iss
}

func countMovedEvents(iss *issue.Issue) int {
	n := 0
	for _, e := range iss.Events {
		if e.Type == issue.EventMoved {
			n++
		}
	}
	return n
}

// Row 1: standard move — fm and dir both match source state, target differs.
// Already covered by TestMove_BacklogToActive but re-asserting under sync semantics.
func TestMoveSync_Row1_Standard(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "row 1")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	root := newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	newPath := filepath.Join(dir, "issues", "active", name)
	iss := parseIssueFile(t, newPath)
	if iss.State != "active" {
		t.Errorf("frontmatter state: got %q want active", iss.State)
	}
	if countMovedEvents(iss) != 1 {
		t.Errorf("expected 1 moved event, got %d", countMovedEvents(iss))
	}
}

// Row 2: git-mv-then-ifs-move — frontmatter stale, directory matches target.
// Should sync the frontmatter and append the missing event.
func TestMoveSync_Row2_GitMoveFirst_FrontmatterRepair(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "row 2")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	gitMove(t, dir, "backlog", "active", name)

	root := newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// File still in active.
	finalPath := filepath.Join(dir, "issues", "active", name)
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("file should remain in active: %v", err)
	}
	iss := parseIssueFile(t, finalPath)
	if iss.State != "active" {
		t.Errorf("frontmatter state: got %q want active", iss.State)
	}
	if countMovedEvents(iss) != 1 {
		t.Errorf("expected 1 moved event after fm repair, got %d", countMovedEvents(iss))
	}
	// The event's `from` must be the frontmatter's previous state (backlog),
	// not the directory (active). Records the actual transition.
	for _, e := range iss.Events {
		if e.Type == issue.EventMoved {
			if e.From != "backlog" || e.To != "active" {
				t.Errorf("event from/to: got %q→%q want backlog→active", e.From, e.To)
			}
		}
	}
}

// Row 3: wrong-target after partial git mv. fm=backlog, dir=done, target=active.
// Both differ from target. Event from must be `iss.State` (backlog), not `m.State` (done).
func TestMoveSync_Row3_WrongTargetAfterGitMv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "row 3")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	gitMove(t, dir, "backlog", "done", name)

	root := newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	finalPath := filepath.Join(dir, "issues", "active", name)
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("file should be in active: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "issues", "done", name)); !os.IsNotExist(err) {
		t.Errorf("file should be gone from done: err=%v", err)
	}
	iss := parseIssueFile(t, finalPath)
	if iss.State != "active" {
		t.Errorf("frontmatter state: got %q want active", iss.State)
	}
	// One moved event total. From must be "backlog" (frontmatter), not "done" (directory).
	if countMovedEvents(iss) != 1 {
		t.Errorf("expected exactly 1 moved event, got %d", countMovedEvents(iss))
	}
	for _, e := range iss.Events {
		if e.Type == issue.EventMoved {
			if e.From != "backlog" {
				t.Errorf("event from must be 'backlog' (frontmatter), got %q (directory contamination?)", e.From)
			}
			if e.To != "active" {
				t.Errorf("event to: got %q want active", e.To)
			}
		}
	}
}

// Row 4: true no-op. fm and dir both equal target. No event. Stderr notice.
func TestMoveSync_Row4_TrueNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "row 4")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	root := newRoot()
	root.SetArgs([]string{"move", short, "backlog"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	iss := parseIssueFile(t, filepath.Join(dir, "issues", "backlog", name))
	if iss.State != "backlog" {
		t.Errorf("state changed unexpectedly: %q", iss.State)
	}
	if countMovedEvents(iss) != 0 {
		t.Errorf("true no-op should append no moved event, got %d", countMovedEvents(iss))
	}
}

// Row 5: directory-only repair. fm=active, dir=backlog, target=active.
// Move the file; do NOT append an event (no transition happened — the directory was the bug).
func TestMoveSync_Row5_DirectoryOnlyRepair(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "row 5")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	// First, move properly to active so frontmatter says active and a moved event exists.
	root := newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	beforeRepair := parseIssueFile(t, filepath.Join(dir, "issues", "active", name))
	movedCountBefore := countMovedEvents(beforeRepair)

	// Now sabotage: git mv it back to backlog without telling ifs.
	gitMove(t, dir, "active", "backlog", name)

	// Repair: ask ifs to move it to where the frontmatter says it should be.
	root = newRoot()
	root.SetArgs([]string{"move", short, "active"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// File back in active.
	finalPath := filepath.Join(dir, "issues", "active", name)
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("file should be back in active: %v", err)
	}
	after := parseIssueFile(t, finalPath)
	if after.State != "active" {
		t.Errorf("frontmatter state should still be active: got %q", after.State)
	}
	// Critical: no NEW moved event. The directory was the bug, not a transition.
	if countMovedEvents(after) != movedCountBefore {
		t.Errorf("directory-only repair should NOT append a moved event; before=%d after=%d",
			movedCountBefore, countMovedEvents(after))
	}
}

// After every operation, the file should pass `ifs verify`.
func TestMoveSync_AllRowsVerifyClean(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	createOne(t, "verify after sync")
	entries, _ := os.ReadDir(filepath.Join(dir, "issues", "backlog"))
	name := entries[0].Name()
	short := strings.Split(name, "-")[1]

	gitMove(t, dir, "backlog", "active", name) // simulate git mv
	root := newRoot()
	root.SetArgs([]string{"move", short, "active"}) // sync repair
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root = newRoot()
	root.SetArgs([]string{"verify", filepath.Join(dir, "issues", "active", name)})
	if err := root.Execute(); err != nil {
		t.Errorf("verify failed after sync: %v", err)
	}
}
