package store

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func gitkeepExists(t *testing.T, root, state string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, state, ".gitkeep"))
	return err == nil
}

func TestScaffold_FreshRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "issues")
	created, err := Scaffold(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"backlog", "active", "done"} {
		if _, err := os.Stat(filepath.Join(root, state)); err != nil {
			t.Errorf("state dir %q missing: %v", state, err)
		}
		if !gitkeepExists(t, root, state) {
			t.Errorf(".gitkeep missing in %s/", state)
		}
	}
	// 1 root + 3 dirs + 3 .gitkeeps.
	if len(created) != 7 {
		t.Errorf("expected 7 created paths (root + 3 dirs + 3 .gitkeep), got %d: %v", len(created), created)
	}
}

func TestScaffold_Idempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "issues")
	if _, err := Scaffold(root); err != nil {
		t.Fatal(err)
	}
	created2, err := Scaffold(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(created2) != 0 {
		t.Errorf("second Scaffold should create nothing, got %v", created2)
	}
}

func TestScaffold_PartialState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "issues")
	// Manually create only backlog/ with a file (no .gitkeep).
	if err := os.MkdirAll(filepath.Join(root, "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backlog", "fake.md"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := Scaffold(root)
	if err != nil {
		t.Fatal(err)
	}
	// Should NOT recreate backlog/ (already exists), but SHOULD add backlog/.gitkeep,
	// active/, active/.gitkeep, done/, done/.gitkeep.
	wantContains := []string{
		filepath.Join(root, "backlog", ".gitkeep"),
		filepath.Join(root, "active"),
		filepath.Join(root, "active", ".gitkeep"),
		filepath.Join(root, "done"),
		filepath.Join(root, "done", ".gitkeep"),
	}
	for _, w := range wantContains {
		if !slices.Contains(created, w) {
			t.Errorf("expected created to contain %q, got %v", w, created)
		}
	}
	if slices.Contains(created, filepath.Join(root, "backlog")) {
		t.Errorf("should not have re-created existing backlog/, got %v", created)
	}
}

func TestScaffold_FullySetUp(t *testing.T) {
	root := filepath.Join(t.TempDir(), "issues")
	for _, state := range []string{"backlog", "active", "done"} {
		dir := filepath.Join(root, state)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	created, err := Scaffold(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("fully set-up tree should create nothing, got %v", created)
	}
}

func TestScaffold_GitkeepPreservedWithFiles(t *testing.T) {
	// .gitkeep stays even when the directory has real .md files. Always-present
	// policy means we don't manage it based on contents.
	root := filepath.Join(t.TempDir(), "issues")
	if _, err := Scaffold(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backlog", "20260428T000000Z-aabbccdd-x.md"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := Scaffold(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("populated tree should still scaffold no-op, got %v", created)
	}
	if !gitkeepExists(t, root, "backlog") {
		t.Errorf(".gitkeep should not have been removed when a real file is present")
	}
}

func TestScaffold_RootIsFile(t *testing.T) {
	// Edge: root path exists but is a regular file. Must error, not silently
	// proceed.
	root := filepath.Join(t.TempDir(), "issues")
	if err := os.WriteFile(root, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Scaffold(root); err == nil {
		t.Errorf("expected error when root is a regular file")
	}
}
