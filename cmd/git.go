package cmd

import (
	"github.com/adam-stokes/orcai/internal/gitui"
	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git [path]",
	Short: "Interactive git UI — browse changes, branches, stage, and commit",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd := "."
		if len(args) > 0 {
			cwd = args[0]
		}
		return gitui.Run(cwd)
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
}
