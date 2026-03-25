package cmd

import (
	"github.com/adam-stokes/orcai/internal/picker"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pickerCmd)
	pickerCmd.Flags().String("bus-socket", "", "Path to orcai bus Unix socket")
}

var pickerCmd = &cobra.Command{
	Use:   "picker",
	Short: "Open the session picker",
	RunE: func(cmd *cobra.Command, args []string) error {
		picker.Run()
		return nil
	},
}
