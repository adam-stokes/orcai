package cmd

import (
	"testing"
)

func TestPickerCmd_Registered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "picker" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("picker subcommand not registered in rootCmd")
	}
}

func TestPickerCmd_BusSocketFlag(t *testing.T) {
	f := pickerCmd.Flags().Lookup("bus-socket")
	if f == nil {
		t.Fatal("--bus-socket flag not registered on pickerCmd")
	}
}
