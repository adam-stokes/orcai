package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/adam-stokes/orcai/internal/picker"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Open a new session from a picker selection (reads ORCAI_PICKER_SELECTION)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("TMUX") == "" {
			return fmt.Errorf("orcai new: must be run inside a tmux session")
		}

		selJSON := os.Getenv("ORCAI_PICKER_SELECTION")
		if selJSON == "" {
			// No selection — open a plain shell window.
			return tmuxNewWindow("", "$SHELL")
		}

		var item picker.PickerItem
		if err := json.Unmarshal([]byte(selJSON), &item); err != nil {
			return fmt.Errorf("orcai new: malformed ORCAI_PICKER_SELECTION: %w", err)
		}

		return launchItem(item)
	},
}

// launchItem opens a new tmux window for the given PickerItem.
func launchItem(item picker.PickerItem) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("orcai new: resolve binary: %w", err)
	}

	switch item.Kind {
	case "pipeline":
		arg := item.Name
		if item.PipelineFile != "" {
			arg = item.PipelineFile
		}
		windowName := "pipeline-" + item.Name
		shellCmd := fmt.Sprintf("%s pipeline run %s; exec $SHELL", self, arg)
		return tmuxNewWindow(windowName, shellCmd)

	case "provider":
		if item.ProviderID == "" {
			return fmt.Errorf("orcai new: provider item missing providerID")
		}
		windowName := item.ProviderID
		shellCmd := item.ProviderID
		if item.ModelID != "" {
			shellCmd = fmt.Sprintf("%s --model %s", item.ProviderID, item.ModelID)
		}
		return tmuxNewWindow(windowName, shellCmd)

	case "session":
		// Focus an existing window rather than opening a new one.
		if item.SessionIndex != "" {
			exec.Command("tmux", "select-window", "-t", "orcai:"+item.SessionIndex).Run() //nolint:errcheck
		}
		return nil

	default:
		// Fallback: open a plain shell window.
		return tmuxNewWindow("", "$SHELL")
	}
}

// tmuxNewWindow creates a new tmux window in the orcai session running shellCmd.
// windowName may be empty (tmux chooses a name).
func tmuxNewWindow(windowName, shellCmd string) error {
	args := []string{"new-window", "-t", "orcai"}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	args = append(args, shellCmd)
	if out, err := exec.Command("tmux", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("orcai new: tmux new-window: %w\n%s", err, out)
	}
	return nil
}
