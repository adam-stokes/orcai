package cmd

import (
	"github.com/spf13/cobra"
)

var bridgeCmd = &cobra.Command{
	Use:    "bridge",
	Short:  "Provider bridge subcommands (internal)",
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(bridgeCmd)
}
