package cmd

import (
	"github.com/adam-stokes/orcai/internal/sidebar"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sysopCmd)
	sysopCmd.Flags().String("bus-socket", "", "Path to orcai bus Unix socket")
}

var sysopCmd = &cobra.Command{
	Use:   "sysop",
	Short: "Open the ABS sysop panel",
	RunE: func(cmd *cobra.Command, args []string) error {
		sidebar.Run()
		return nil
	},
}
