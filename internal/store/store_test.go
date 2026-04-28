package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFindsExisting(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "issues")
	if err := os.MkdirAll(filepath.Join(want, "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, found, err := Resolve(deep)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Errorf("expected found=true")
	}
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveNotFoundReturnsCWDPath(t *testing.T) {
	root := t.TempDir()
	got, found, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Errorf("expected found=false")
	}
	if got != filepath.Join(root, "issues") {
		t.Errorf("got %q", got)
	}
}

func TestWriteNewRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteNew(dir, "x.md", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteNew(dir, "x.md", []byte("b")); err == nil {
		t.Errorf("expected error on overwrite")
	}
}
