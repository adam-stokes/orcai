package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adam-stokes/orcai/internal/picker"
)

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentRunCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agent operations",
}

// agentRunCmd invokes a provider agent headlessly (no TTY / tmux required).
// The target format is "<provider>[/<model>]", e.g. "opencode/ollama/llama3.2:latest".
// The provider binary is resolved via the provider registry; pipeline.args from
// the provider's YAML are prepended so the binary enters headless/run mode.
var agentRunCmd = &cobra.Command{
	Use:   "run <provider[/model]>",
	Short: "Run an agent headlessly (for cron / scripted use)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Parse "<provider>" or "<provider>/<model>" from the target.
		providerID, model, _ := strings.Cut(target, "/")
		if providerID == "" {
			return fmt.Errorf("agent run: empty provider in target %q", target)
		}

		// Find the provider definition.
		providers := picker.BuildProviders()
		var found *picker.ProviderDef
		for i, p := range providers {
			if p.ID == providerID {
				found = &providers[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("agent run: provider %q not found (available: %s)",
				providerID, joinProviderIDs(providers))
		}

		binary := found.Command
		if binary == "" {
			binary = found.ID
		}

		// Build the argument list: pipeline args first, then --model if provided.
		invokeArgs := make([]string, 0, len(found.PipelineArgs)+2)
		invokeArgs = append(invokeArgs, found.PipelineArgs...)
		if model != "" {
			invokeArgs = append(invokeArgs, "--model", model)
		}

		c := exec.Command(binary, invokeArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin

		if err := c.Run(); err != nil {
			return fmt.Errorf("agent run: %s exited: %w", binary, err)
		}
		return nil
	},
}

func joinProviderIDs(providers []picker.ProviderDef) string {
	ids := make([]string, 0, len(providers))
	for _, p := range providers {
		ids = append(ids, p.ID)
	}
	return strings.Join(ids, ", ")
}
