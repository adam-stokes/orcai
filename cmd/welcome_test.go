package cmd

import (
	"testing"
)

func TestWelcomeCmd_Registered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "welcome" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("welcome subcommand not registered in rootCmd")
	}
}

func TestWelcomeCmd_BusSocketFlag(t *testing.T) {
	f := welcomeCmd.Flags().Lookup("bus-socket")
	if f == nil {
		t.Fatal("--bus-socket flag not registered on welcomeCmd")
	}
}
