package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/adam-stokes/orcai/internal/ollama"
)

var ollamaCmd = &cobra.Command{
	Use:   "ollama",
	Short: "Manage local Ollama models and extended-context variants",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(ollama.NewTUI(), tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(ollamaCmd)
}
