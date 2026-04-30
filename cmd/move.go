package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nickg/issuefs/internal/issue"
	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

func newMove() *cobra.Command {
	return &cobra.Command{
		Use:   "move <ref> <state>",
		Short: "Move an issue to a different state directory",
		Long: `Move an issue across state directories (backlog/active/done).

Resolves <ref> as a path, full filename, full ID, or any unique prefix of
the 8-hex random suffix.

Contract: after this command succeeds, both the file's location AND its
frontmatter "state" field equal <state>. Always. The verb is idempotent
and self-healing — if you ` + "`git mv`" + ` first to preserve git's rename detection,
running ` + "`ifs move <ref> <state>`" + ` afterward will sync the metadata. If
frontmatter and directory disagree, this verb fixes them.

A move where both directory AND frontmatter already equal <state> is a true
no-op (with a notice on stderr; no event appended). Pure file operation;
never runs git.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0], args[1])
		},
	}
}

func runMove(stdout, stderr io.Writer, ref, target string) error {
	if !issue.IsValidState(target) {
		return fmt.Errorf("--state must be one of %v", issue.ValidStates())
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, found, err := store.Resolve(cwd)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no issues directory found from %s", cwd)
	}

	r, err := store.NewResolver(root)
	if err != nil {
		return err
	}
	m, err := r.Lookup(ref)
	if err != nil {
		return err
	}

	iss, err := readIssue(m.AbsPath)
	if err != nil {
		return err
	}

	needFmUpdate := iss.State != target
	needRename := m.State != target

	// True no-op: both location and frontmatter already in sync.
	if !needFmUpdate && !needRename {
		fmt.Fprintf(stderr, "%s is already in %s\n", m.Short, target)
		fmt.Fprintln(stdout, m.AbsPath)
		return nil
	}

	// Frontmatter update: append a `moved` event whose `from` is the
	// frontmatter's previous state (not the directory). Records what the
	// issue *thought* its state was, which is the canonical history.
	if needFmUpdate {
		now := time.Now().UTC().Truncate(time.Second)
		iss.Events = append(iss.Events, issue.NewMoved(now, iss.State, target))
		iss.State = target
		data, err := issue.Marshal(iss)
		if err != nil {
			return err
		}
		// Edit-then-rename: write updated content in place first so a crash
		// leaves the file in the source dir with state matching the source dir.
		if err := os.WriteFile(m.AbsPath, data, 0o644); err != nil {
			return err
		}
	}

	// Rename: move the file across directories if it's not already in the
	// target. When this is the only change (frontmatter was already correct),
	// no event is appended — the directory was the bug, no transition happened.
	finalPath := m.AbsPath
	if needRename {
		targetDir, err := store.EnsureSubdir(root, target)
		if err != nil {
			return err
		}
		newPath := filepath.Join(targetDir, m.Name)
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("destination already exists: %s", newPath)
		}
		if err := os.Rename(m.AbsPath, newPath); err != nil {
			return err
		}
		finalPath = newPath
	}
	fmt.Fprintln(stdout, finalPath)
	return nil
}

func readIssue(path string) (*issue.Issue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return issue.Parse(f)
}
