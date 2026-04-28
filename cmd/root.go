package cmd

import "github.com/spf13/cobra"

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "ifs",
		Short:         "Issue File System — issues as files alongside code",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newCreate())
	root.AddCommand(newMove())
	root.AddCommand(newVerify())
	root.AddCommand(newVersion())
	return root
}

func Execute() error {
	return newRoot().Execute()
}
