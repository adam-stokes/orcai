package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(killCmd)
}

var killCmd = &cobra.Command{
	Use:   "kill <session-id>",
	Short: "Kill a running AI session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO(Task 4): send kill-session request to sidebar via tmux pipe.
		return fmt.Errorf("orcai kill: not yet implemented in tmux mode")
	},
}
