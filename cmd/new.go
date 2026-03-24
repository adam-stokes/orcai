package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new <provider>",
	Short: "Open a new AI session (claude, opencode, gemini, aider...)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO(Task 4): send new-session request to sidebar via tmux pipe.
		return fmt.Errorf("orcai new: not yet implemented in tmux mode")
	},
}
