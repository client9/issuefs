package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Match describes a single issue file located by the resolver.
type Match struct {
	Path    string // repo-relative path, e.g. "issues/done/1234-bug.md"
	AbsPath string // absolute path to the file on disk
	State   string // backlog | active | done
	Name    string // basename, e.g. "20260428T004315Z-9f2a4b7c-slug.md"
	ID      string // "<timestamp>-<8hex>"
	Short   string // the 8-hex random suffix
}

// Resolver indexes every issue file under a root and resolves refs to them.
type Resolver struct {
	root  string
	all   []Match
	byKey map[string][]Match // lower-cased prefix index keys; values may overlap
}

// NewResolver scans root for issue files. States other than the three known
// names are skipped.
func NewResolver(root string) (*Resolver, error) {
	r := &Resolver{root: root, byKey: map[string][]Match{}}
	for _, state := range []string{"backlog", "active", "done"} {
		dir := filepath.Join(root, state)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			m, ok := matchFromName(state, dir, e.Name())
			if !ok {
				continue
			}
			r.all = append(r.all, m)
		}
	}
	for _, m := range r.all {
		r.byKey[strings.ToLower(m.Name)] = append(r.byKey[strings.ToLower(m.Name)], m)
		r.byKey[strings.ToLower(strings.TrimSuffix(m.Name, ".md"))] = append(r.byKey[strings.ToLower(strings.TrimSuffix(m.Name, ".md"))], m)
		r.byKey[strings.ToLower(m.ID)] = append(r.byKey[strings.ToLower(m.ID)], m)
	}
	return r, nil
}

// matchFromName parses a filename like "<ts>-<hex>-<slug>.md" or "<ts>-<hex>.md"
// into a Match. Returns ok=false if the name doesn't fit the expected shape.
func matchFromName(state, dir, name string) (Match, bool) {
	base := strings.TrimSuffix(name, ".md")
	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 2 {
		return Match{}, false
	}
	ts, hex := parts[0], parts[1]
	if len(ts) != len("20060102T150405Z") || !strings.HasSuffix(ts, "Z") {
		return Match{}, false
	}
	if !isHex(hex) {
		return Match{}, false
	}
	return Match{
		Path:    filepath.ToSlash(filepath.Join("issues", state, name)),
		AbsPath: filepath.Join(dir, name),
		State:   state,
		Name:    name,
		ID:      ts + "-" + hex,
		Short:   hex,
	}, true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// Lookup resolves a ref to a single Match. A ref may be:
//   - a path (contains a separator or ends in ".md") — opened directly
//   - a full filename (with or without ".md")
//   - a full ID ("<ts>-<hex>")
//   - any prefix of the random hex suffix (case-insensitive)
//
// Returns an error if no match is found, or a git-style ambiguity error if
// more than one issue matches.
func (r *Resolver) Lookup(ref string) (Match, error) {
	if ref == "" {
		return Match{}, fmt.Errorf("empty ref")
	}
	if strings.ContainsRune(ref, filepath.Separator) || strings.HasSuffix(ref, ".md") {
		return r.lookupPath(ref)
	}
	key := strings.ToLower(ref)
	if ms, ok := r.byKey[key]; ok && len(ms) == 1 {
		return ms[0], nil
	}
	// Prefix-match against the random hex suffix.
	var hits []Match
	for _, m := range r.all {
		if strings.HasPrefix(strings.ToLower(m.Short), key) {
			hits = append(hits, m)
		}
	}
	switch len(hits) {
	case 0:
		return Match{}, fmt.Errorf("no issue matching %q", ref)
	case 1:
		return hits[0], nil
	default:
		return Match{}, ambiguous(ref, hits)
	}
}

func ambiguous(ref string, hits []Match) error {
	shorts := make([]string, len(hits))
	for i, m := range hits {
		shorts[i] = m.Short
	}
	sort.Strings(shorts)
	return fmt.Errorf("ambiguous ref %q: matches %s", ref, strings.Join(shorts, ", "))
}

func (r *Resolver) lookupPath(ref string) (Match, error) {
	abs, err := filepath.Abs(ref)
	if err != nil {
		return Match{}, err
	}
	for _, m := range r.all {
		if m.AbsPath == abs {
			return m, nil
		}
	}
	// Fall back to filename match if the path didn't resolve under the root.
	name := filepath.Base(ref)
	if ms, ok := r.byKey[strings.ToLower(name)]; ok && len(ms) == 1 {
		return ms[0], nil
	}
	return Match{}, fmt.Errorf("no issue at %s", ref)
}

// All returns every indexed match (for `list`-style verbs).
func (r *Resolver) All() []Match { return append([]Match(nil), r.all...) }
