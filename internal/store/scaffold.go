package store

import (
	"errors"
	"os"
	"path/filepath"
)

// gitKeep is the conventional placeholder filename used to make an otherwise
// empty directory git-trackable.
const gitKeep = ".gitkeep"

// scaffoldStates is the set of state subdirectories Scaffold creates. Mirrors
// the canonical state list. Kept here (not imported from internal/issue) to
// avoid an import cycle; the values must stay in sync.
var scaffoldStates = []string{"backlog", "active", "done"}

// Scaffold ensures the standard issuefs layout exists at root: every state
// subdirectory present, each with a .gitkeep file so git tracks the directory
// even when it contains no .md files. Idempotent — safe to run on a fresh
// tree, a partially-initialized one, or a fully-set-up one.
//
// Returns the list of paths Scaffold actually created (directories and .gitkeep
// files). An empty slice means nothing was missing. Caller decides whether to
// print.
func Scaffold(root string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var created []string

	// Ensure the root itself exists.
	if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return created, err
		}
		created = append(created, abs)
	} else if err != nil {
		return created, err
	}

	for _, state := range scaffoldStates {
		dir := filepath.Join(abs, state)
		dirCreated, err := ensureDir(dir)
		if err != nil {
			return created, err
		}
		if dirCreated {
			created = append(created, dir)
		}
		keep := filepath.Join(dir, gitKeep)
		keepCreated, err := ensureEmptyFile(keep)
		if err != nil {
			return created, err
		}
		if keepCreated {
			created = append(created, keep)
		}
	}
	return created, nil
}

// ensureDir creates dir if missing. Returns true if it was created.
func ensureDir(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return false, &os.PathError{Op: "scaffold", Path: dir, Err: errors.New("path exists and is not a directory")}
		}
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	return true, nil
}

// ensureEmptyFile creates path as a zero-byte file if missing. Returns true
// if it was created. If the file exists (any size, any contents), leaves it.
func ensureEmptyFile(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return false, err
	}
	return true, f.Close()
}
