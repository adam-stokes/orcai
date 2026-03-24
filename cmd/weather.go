package cmd

import (
	"github.com/adam-stokes/orcai/internal/weather"
	"github.com/spf13/cobra"
)

var weatherCity string

var weatherCmd = &cobra.Command{
	Use:   "weather",
	Short: "Current weather, forecast, and air quality",
	RunE: func(cmd *cobra.Command, args []string) error {
		return weather.Run(weatherCity)
	},
}

func init() {
	weatherCmd.Flags().StringVar(&weatherCity, "city", "", "City name (auto-detected from IP if not specified)")
	rootCmd.AddCommand(weatherCmd)
}
