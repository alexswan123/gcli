package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alexandraswan/gcli/internal/auth"
	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage account authentication",
	Long:  `Add, remove, and manage Google account authentication.`,
}

var authAddCmd = &cobra.Command{
	Use:   "add <account-name>",
	Short: "Add and authenticate a new account",
	Long: `Add a new Google account and authenticate via OAuth.

You need to provide your Google OAuth client credentials. You can get these from
the Google Cloud Console (https://console.cloud.google.com):

1. Create a new project or select an existing one
2. Enable the Gmail API and Google Calendar API
3. Create OAuth 2.0 credentials (Desktop app type)
4. Add http://localhost:8085/callback as an authorized redirect URI

Example:
  gcli auth add personal --client-id YOUR_CLIENT_ID --client-secret YOUR_CLIENT_SECRET`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]
		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		calendarID, _ := cmd.Flags().GetString("calendar-id")

		// Load existing config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if account already exists
		if _, exists := cfg.Accounts[accountName]; exists {
			return fmt.Errorf("account '%s' already exists. Use 'gcli auth remove %s' first", accountName, accountName)
		}

		// Prompt for credentials if not provided
		if clientID == "" {
			fmt.Print("Enter Google Client ID: ")
			reader := bufio.NewReader(os.Stdin)
			clientID, _ = reader.ReadString('\n')
			clientID = strings.TrimSpace(clientID)
		}
		if clientSecret == "" {
			fmt.Print("Enter Google Client Secret: ")
			reader := bufio.NewReader(os.Stdin)
			clientSecret, _ = reader.ReadString('\n')
			clientSecret = strings.TrimSpace(clientSecret)
		}

		if clientID == "" || clientSecret == "" {
			return fmt.Errorf("client ID and client secret are required")
		}

		// Create account config
		account := config.AccountConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			CalendarID:   calendarID,
		}

		// Add account to config
		if err := cfg.AddAccount(accountName, account); err != nil {
			return err
		}

		// Perform OAuth authentication
		if err := auth.AuthenticateAccount(accountName, account); err != nil {
			// Remove the account if authentication failed
			cfg.RemoveAccount(accountName)
			return fmt.Errorf("authentication failed: %w", err)
		}

		output.PrintSuccess("Account '%s' added and authenticated successfully!", accountName)
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var accounts []output.AccountInfo
		for name, acc := range cfg.Accounts {
			accounts = append(accounts, output.AccountInfo{
				Name:       name,
				IsDefault:  name == cfg.DefaultAccount,
				HasToken:   auth.TokenExists(name),
				CalendarID: acc.CalendarID,
			})
		}

		output.PrintAccountList(accounts)
		return nil
	},
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove <account-name>",
	Short: "Remove an account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.RemoveAccount(accountName); err != nil {
			return err
		}

		output.PrintSuccess("Account '%s' removed successfully!", accountName)
		return nil
	},
}

var authDefaultCmd = &cobra.Command{
	Use:   "default <account-name>",
	Short: "Set the default account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.SetDefault(accountName); err != nil {
			return err
		}

		output.PrintSuccess("Default account set to '%s'", accountName)
		return nil
	},
}

var authReauthCmd = &cobra.Command{
	Use:   "reauth <account-name>",
	Short: "Re-authenticate an existing account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		_, account, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		// Remove existing token
		auth.RemoveToken(accountName)

		// Re-authenticate
		if err := auth.AuthenticateAccount(accountName, account); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		output.PrintSuccess("Account '%s' re-authenticated successfully!", accountName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRemoveCmd)
	authCmd.AddCommand(authDefaultCmd)
	authCmd.AddCommand(authReauthCmd)

	authAddCmd.Flags().String("client-id", "", "Google OAuth Client ID")
	authAddCmd.Flags().String("client-secret", "", "Google OAuth Client Secret")
	authAddCmd.Flags().String("calendar-id", "", "Calendar ID to use (default: primary)")
}
