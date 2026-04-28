package cmd

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Build-time injection via -ldflags. Empty when not stamped; debug.ReadBuildInfo
// is consulted as a fallback so `go install` users still get useful output.
var (
	version = ""
	commit  = ""
	date    = ""
)

func newVersion() *cobra.Command {
	var short bool
	c := &cobra.Command{
		Use:   "version",
		Short: "Print the ifs version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd.OutOrStdout(), short)
		},
	}
	c.Flags().BoolVar(&short, "short", false, "print just the version number")
	return c
}

func runVersion(stdout io.Writer, short bool) error {
	v, c, d := resolveVersion()
	if short {
		fmt.Fprintln(stdout, v)
		return nil
	}
	if c == "" && d == "" {
		fmt.Fprintf(stdout, "ifs %s\n", v)
		return nil
	}
	parts := []string{}
	if c != "" {
		parts = append(parts, c)
	}
	if d != "" {
		parts = append(parts, d)
	}
	fmt.Fprintf(stdout, "ifs %s (%s)\n", v, strings.Join(parts, ", "))
	return nil
}

// resolveVersion returns version, commit, date — preferring -ldflags values,
// falling back to debug.ReadBuildInfo, finally returning "(devel)" with
// empty commit/date when nothing is available.
func resolveVersion() (v, c, d string) {
	v, c, d = version, commit, date
	if v != "" {
		return v, c, d
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)", "", ""
	}
	v = info.Main.Version
	if v == "" || v == "(devel)" {
		v = "(devel)"
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if c == "" && len(s.Value) >= 7 {
				c = s.Value[:7]
			} else if c == "" {
				c = s.Value
			}
		case "vcs.time":
			if d == "" {
				// vcs.time is RFC3339; trim to date for compactness.
				if len(s.Value) >= 10 {
					d = s.Value[:10]
				} else {
					d = s.Value
				}
			}
		}
	}
	return v, c, d
}
