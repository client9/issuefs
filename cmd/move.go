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
the 8-hex random suffix. Updates the frontmatter "state" field, appends a
"moved" event to the log, then renames the file across directories. The
basename is unchanged so git's rename detection pairs the move automatically
when the user later runs ` + "`git add`" + `.

A move to the same state is a no-op (with a notice on stderr). Pure file
operation; never runs git.`,
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

	if m.State == target {
		fmt.Fprintf(stderr, "%s is already in %s\n", m.Short, target)
		fmt.Fprintln(stdout, m.Path)
		return nil
	}

	iss, err := readIssue(m.Path)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Truncate(time.Second)
	iss.Events = append(iss.Events, issue.NewMoved(now, m.State, target))
	iss.State = target

	data, err := issue.Marshal(iss)
	if err != nil {
		return err
	}

	// Edit-then-rename: write updated content in place first so a crash
	// leaves the file in the source dir with state matching the source dir.
	// Recoverable; no data loss.
	if err := os.WriteFile(m.Path, data, 0o644); err != nil {
		return err
	}

	targetDir, err := store.EnsureSubdir(root, target)
	if err != nil {
		return err
	}
	newPath := filepath.Join(targetDir, m.Name)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("destination already exists: %s", newPath)
	}
	if err := os.Rename(m.Path, newPath); err != nil {
		return err
	}
	fmt.Fprintln(stdout, newPath)
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
