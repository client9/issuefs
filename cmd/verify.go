package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/nickg/issuefs/internal/issue"
	"github.com/spf13/cobra"
)

func newVerify() *cobra.Command {
	return &cobra.Command{
		Use:   "verify <file>...",
		Short: "Verify that one or more issue files are well-formed",
		Long: `Verify that each file:
  - parses as bare-JSON frontmatter followed by an optional body
  - separates frontmatter from body by exactly one blank line (or ends at the JSON)
  - contains no unknown frontmatter fields
  - has all required fields (title, id, state, created) with a valid state

Silent on success. Prints "<path>: <error>" for each failure and exits non-zero
if any file failed.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
}

func runVerify(_ io.Writer, stderr io.Writer, paths []string) error {
	failed := 0
	for _, p := range paths {
		if err := verifyFile(p); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", p, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d file(s) failed verification", failed)
	}
	return nil
}

func verifyFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = issue.Verify(f)
	return err
}
