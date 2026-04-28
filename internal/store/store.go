package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DirName is the top-level directory that holds all issues.
const DirName = "issues"

// Resolve walks upward from start looking for an existing "issues/" directory.
// If none is found, it returns (filepath.Join(start, "issues"), false, nil) so
// the caller can decide whether to create it.
func Resolve(start string) (root string, found bool, err error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", false, err
	}
	dir := abs
	for {
		candidate := filepath.Join(dir, DirName)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(abs, DirName), false, nil
		}
		dir = parent
	}
}

// EnsureSubdir makes sure root/sub exists, creating root and sub if needed.
func EnsureSubdir(root, sub string) (string, error) {
	target := filepath.Join(root, sub)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

// WriteNew writes data to dir/name, refusing to overwrite an existing file.
func WriteNew(dir, name string, data []byte) (string, error) {
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return path, nil
}
