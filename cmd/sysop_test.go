package cmd

import (
	"testing"
)

func TestSysopCmd_Registered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "sysop" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sysop subcommand not registered in rootCmd")
	}
}

func TestSysopCmd_BusSocketFlag(t *testing.T) {
	f := sysopCmd.Flags().Lookup("bus-socket")
	if f == nil {
		t.Fatal("--bus-socket flag not registered on sysopCmd")
	}
}
