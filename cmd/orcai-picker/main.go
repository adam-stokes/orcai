// Binary orcai-picker is the ABS provider/model picker widget.
//
// It is launched via tmux display-popup (typically from orcai-welcome or a
// chord binding) and lets the user select an AI provider and model to start
// a new session.
package main

import "github.com/adam-stokes/orcai/internal/picker"

func main() {
	picker.Run()
}
