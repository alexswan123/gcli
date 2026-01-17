package cmd

import (
	"github.com/alexandraswan/gcli/internal/output"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "gcli",
	Short: "Gmail and Google Calendar CLI tool",
	Long: `gcli is a command-line tool for managing Gmail and Google Calendar.

It supports multiple accounts (e.g., work and personal) and allows you to:
- Read, draft, send, and schedule emails
- List, create, update, and delete calendar events
- Query a single account or all accounts at once

Get started by adding an account:
  gcli auth add personal

Then authenticate:
  gcli auth add personal --client-id YOUR_ID --client-secret YOUR_SECRET`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.JSONOutput = jsonOutput
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
}
