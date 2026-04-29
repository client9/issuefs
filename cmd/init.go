package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

func newInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the issues/ directory layout",
		Long: `Create issues/{backlog,active,done}/ with .gitkeep files in each.

Idempotent. Safe to run on a fresh repo, on an existing repo missing some
state directories, or on a fully-set-up repo (in which case it's a no-op
that prints nothing).

Run this once after cloning a repo that uses ifs to immunize the layout
against state directories disappearing in git when the last issue moves out.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.OutOrStdout())
		},
	}
}

func runInit(stdout io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := filepath.Join(cwd, store.DirName)
	created, err := store.Scaffold(root)
	if err != nil {
		return err
	}
	if len(created) == 0 {
		fmt.Fprintln(stdout, "already initialized")
		return nil
	}
	for _, p := range created {
		// Print paths relative to cwd when possible for readable output.
		rel, relErr := filepath.Rel(cwd, p)
		if relErr != nil {
			rel = p
		}
		fmt.Fprintf(stdout, "created %s\n", rel)
	}
	return nil
}
