package cmd

import (
	"fmt"

	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and modify gcli configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if output.JSONOutput {
			output.PrintJSON(cfg)
			return nil
		}

		configDir, _ := config.GetConfigDir()
		fmt.Printf("Configuration directory: %s\n\n", configDir)

		if !cfg.HasAccounts() {
			fmt.Println("No accounts configured.")
			fmt.Println("\nRun 'gcli auth add <name>' to add an account.")
			return nil
		}

		fmt.Printf("Default account: %s\n\n", cfg.DefaultAccount)
		fmt.Println("Accounts:")
		for name, acc := range cfg.Accounts {
			calID := acc.CalendarID
			if calID == "" {
				calID = "primary"
			}
			defaultMarker := ""
			if name == cfg.DefaultAccount {
				defaultMarker = " (default)"
			}
			fmt.Printf("  %s%s\n", name, defaultMarker)
			fmt.Printf("    Calendar ID: %s\n", calID)
			fmt.Printf("    Client ID: %s...\n", truncateString(acc.ClientID, 20))
		}

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  default-account <name>    Set the default account
  <account>.calendar-id <id>  Set calendar ID for an account

Examples:
  gcli config set default-account work
  gcli config set work.calendar-id "work@company.com"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		switch key {
		case "default-account":
			if err := cfg.SetDefault(value); err != nil {
				return err
			}
			output.PrintSuccess("Default account set to '%s'", value)

		default:
			// Check for account.property format
			var accountName, property string
			for i := len(key) - 1; i >= 0; i-- {
				if key[i] == '.' {
					accountName = key[:i]
					property = key[i+1:]
					break
				}
			}

			if accountName == "" {
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			acc, exists := cfg.Accounts[accountName]
			if !exists {
				return fmt.Errorf("account '%s' does not exist", accountName)
			}

			switch property {
			case "calendar-id":
				acc.CalendarID = value
				if err := cfg.UpdateAccount(accountName, acc); err != nil {
					return err
				}
				output.PrintSuccess("Calendar ID for '%s' set to '%s'", accountName, value)

			default:
				return fmt.Errorf("unknown property '%s' for account '%s'", property, accountName)
			}
		}

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.GetConfigPath()
		if err != nil {
			return err
		}

		if output.JSONOutput {
			output.PrintJSON(map[string]string{
				"config_path": configPath,
			})
			return nil
		}

		fmt.Println(configPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
