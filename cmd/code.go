package cmd

import "github.com/spf13/cobra"

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Open stok desktop workspace (run 'stok' without arguments)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("Run 'stok' without arguments to open the desktop app.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(codeCmd)
}
