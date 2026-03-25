package cmd

import (
	"github.com/adam-stokes/orcai/internal/welcome"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(welcomeCmd)
	welcomeCmd.Flags().String("bus-socket", "", "Path to orcai bus Unix socket")
}

var welcomeCmd = &cobra.Command{
	Use:   "welcome",
	Short: "Open the ABS welcome dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		busSocket, _ := cmd.Flags().GetString("bus-socket")
		return welcome.Run(busSocket)
	},
}
