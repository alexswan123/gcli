package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/gmail"
	"github.com/alexandraswan/gcli/internal/output"
	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:     "mail",
	Aliases: []string{"m", "email"},
	Short:   "Manage emails",
	Long:    `Read, draft, send, and schedule emails.`,
}

var mailReadCmd = &cobra.Command{
	Use:   "read",
	Short: "List emails",
	Long: `List emails from one or all accounts.

Examples:
  gcli mail read                      # Read from default account
  gcli mail read -a work              # Read from work account
  gcli mail read --all                # Read from all accounts
  gcli mail read -q "is:unread"       # Filter unread emails
  gcli mail read -n 20                # Limit to 20 emails`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		allAccounts, _ := cmd.Flags().GetBool("all")
		query, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt64("limit")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.HasAccounts() {
			return fmt.Errorf("no accounts configured. Run 'gcli auth add <name>' first")
		}

		var accounts []string
		if allAccounts {
			accounts = cfg.GetAllAccounts()
		} else {
			name, _, err := cfg.GetAccount(accountName)
			if err != nil {
				return err
			}
			accounts = []string{name}
		}

		var allEmails []output.EmailSummary
		var mu sync.Mutex
		var wg sync.WaitGroup
		errChan := make(chan error, len(accounts))

		for _, accName := range accounts {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()

				_, acc, err := cfg.GetAccount(name)
				if err != nil {
					errChan <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				client, err := gmail.NewClient(ctx, name, acc)
				if err != nil {
					errChan <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				emails, err := client.ListMessages(ctx, query, limit)
				if err != nil {
					errChan <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				mu.Lock()
				allEmails = append(allEmails, emails...)
				mu.Unlock()
			}(accName)
		}

		wg.Wait()
		close(errChan)

		// Report any errors
		for err := range errChan {
			output.PrintError("%v", err)
		}

		output.PrintEmailList(allEmails)
		return nil
	},
}

var mailGetCmd = &cobra.Command{
	Use:   "get <message-id>",
	Short: "Get email details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		messageID := args[0]
		accountName, _ := cmd.Flags().GetString("account")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := gmail.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		email, err := client.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}

		output.PrintEmailDetail(email)
		return nil
	},
}

var mailDraftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Create an email draft",
	Long: `Create an email draft without sending it.

Example:
  gcli mail draft -t "user@example.com" -s "Hello" -b "Message body"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		to, _ := cmd.Flags().GetStringSlice("to")
		cc, _ := cmd.Flags().GetStringSlice("cc")
		bcc, _ := cmd.Flags().GetStringSlice("bcc")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		html, _ := cmd.Flags().GetBool("html")

		if len(to) == 0 {
			return fmt.Errorf("at least one recipient is required (--to)")
		}
		if subject == "" {
			return fmt.Errorf("subject is required (--subject)")
		}
		if body == "" {
			return fmt.Errorf("body is required (--body)")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := gmail.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		draft := gmail.DraftEmail{
			To:      to,
			CC:      cc,
			BCC:     bcc,
			Subject: subject,
			Body:    body,
			IsHTML:  html,
		}

		draftID, err := client.CreateDraft(ctx, draft)
		if err != nil {
			return err
		}

		output.PrintSuccess("Draft created (ID: %s)", draftID)
		return nil
	},
}

var mailSendCmd = &cobra.Command{
	Use:   "send <draft-id>",
	Short: "Send an existing draft",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		draftID := args[0]
		accountName, _ := cmd.Flags().GetString("account")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := gmail.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		msgID, err := client.SendDraft(ctx, draftID)
		if err != nil {
			return err
		}

		output.PrintSuccess("Email sent (Message ID: %s)", msgID)
		return nil
	},
}

var mailSendNowCmd = &cobra.Command{
	Use:   "send-now",
	Short: "Compose and send an email immediately",
	Long: `Compose and send an email directly without creating a draft first.

Example:
  gcli mail send-now -t "user@example.com" -s "Hello" -b "Message body"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		to, _ := cmd.Flags().GetStringSlice("to")
		cc, _ := cmd.Flags().GetStringSlice("cc")
		bcc, _ := cmd.Flags().GetStringSlice("bcc")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		html, _ := cmd.Flags().GetBool("html")

		if len(to) == 0 {
			return fmt.Errorf("at least one recipient is required (--to)")
		}
		if subject == "" {
			return fmt.Errorf("subject is required (--subject)")
		}
		if body == "" {
			return fmt.Errorf("body is required (--body)")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := gmail.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		email := gmail.DraftEmail{
			To:      to,
			CC:      cc,
			BCC:     bcc,
			Subject: subject,
			Body:    body,
			IsHTML:  html,
		}

		msgID, err := client.SendEmail(ctx, email)
		if err != nil {
			return err
		}

		output.PrintSuccess("Email sent (Message ID: %s)", msgID)
		return nil
	},
}

var mailScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Schedule an email to be sent later",
	Long: `Create an email draft and schedule it for later sending.

The scheduled time should be in ISO 8601 format (e.g., 2024-12-25T10:00:00).

Example:
  gcli mail schedule -t "user@example.com" -s "Hello" -b "Message" --at "2024-12-25T10:00:00"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		to, _ := cmd.Flags().GetStringSlice("to")
		cc, _ := cmd.Flags().GetStringSlice("cc")
		bcc, _ := cmd.Flags().GetStringSlice("bcc")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		html, _ := cmd.Flags().GetBool("html")
		atStr, _ := cmd.Flags().GetString("at")

		if len(to) == 0 {
			return fmt.Errorf("at least one recipient is required (--to)")
		}
		if subject == "" {
			return fmt.Errorf("subject is required (--subject)")
		}
		if body == "" {
			return fmt.Errorf("body is required (--body)")
		}
		if atStr == "" {
			return fmt.Errorf("schedule time is required (--at)")
		}

		// Parse schedule time
		scheduledAt, err := parseDateTime(atStr)
		if err != nil {
			return fmt.Errorf("invalid schedule time: %w", err)
		}

		if scheduledAt.Before(time.Now()) {
			return fmt.Errorf("schedule time must be in the future")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := gmail.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		// Create draft
		draft := gmail.DraftEmail{
			To:      to,
			CC:      cc,
			BCC:     bcc,
			Subject: subject,
			Body:    body,
			IsHTML:  html,
		}

		draftID, err := client.CreateDraft(ctx, draft)
		if err != nil {
			return err
		}

		// Schedule email
		scheduled := gmail.ScheduledEmailData{
			Account:     name,
			DraftID:     draftID,
			To:          to,
			CC:          cc,
			BCC:         bcc,
			Subject:     subject,
			Body:        body,
			IsHTML:      html,
			ScheduledAt: scheduledAt,
		}

		if err := gmail.AddScheduledEmail(scheduled); err != nil {
			return fmt.Errorf("failed to schedule email: %w", err)
		}

		output.PrintSuccess("Email scheduled for %s", scheduledAt.Format("Mon, 02 Jan 2006 15:04 MST"))
		output.PrintInfo("Draft ID: %s", draftID)
		output.PrintInfo("Run 'gcli mail scheduled send' to send scheduled emails when ready")
		return nil
	},
}

var mailScheduledCmd = &cobra.Command{
	Use:   "scheduled",
	Short: "Manage scheduled emails",
}

var mailScheduledListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName, _ := cmd.Flags().GetString("account")
		pendingOnly, _ := cmd.Flags().GetBool("pending")

		emails, err := gmail.GetScheduledEmailsByAccount(accountName)
		if err != nil {
			return err
		}

		if pendingOnly {
			var pending []output.ScheduledEmail
			for _, e := range emails {
				if !e.Sent && e.Error == "" {
					pending = append(pending, e)
				}
			}
			emails = pending
		}

		output.PrintScheduledEmails(emails)
		return nil
	},
}

var mailScheduledSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send scheduled emails that are ready",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		pending, err := gmail.GetPendingScheduledEmails(accountName)
		if err != nil {
			return err
		}

		if len(pending) == 0 {
			output.PrintInfo("No scheduled emails ready to send")
			return nil
		}

		fmt.Printf("Found %d email(s) ready to send\n\n", len(pending))

		if dryRun {
			output.PrintWarning("Dry run mode - no emails will be sent")
			for _, e := range pending {
				fmt.Printf("  - %s (to: %s)\n", e.Subject, strings.Join(e.To, ", "))
			}
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var sentCount, errorCount int
		for _, e := range pending {
			_, acc, err := cfg.GetAccount(e.Account)
			if err != nil {
				output.PrintError("[%s] %v", e.Subject, err)
				gmail.MarkScheduledEmailError(e.ID, err.Error())
				errorCount++
				continue
			}

			client, err := gmail.NewClient(ctx, e.Account, acc)
			if err != nil {
				output.PrintError("[%s] %v", e.Subject, err)
				gmail.MarkScheduledEmailError(e.ID, err.Error())
				errorCount++
				continue
			}

			msgID, err := client.SendDraft(ctx, e.DraftID)
			if err != nil {
				output.PrintError("[%s] %v", e.Subject, err)
				gmail.MarkScheduledEmailError(e.ID, err.Error())
				errorCount++
				continue
			}

			gmail.MarkScheduledEmailSent(e.ID, msgID)
			output.PrintSuccess("Sent: %s", e.Subject)
			sentCount++
		}

		fmt.Printf("\nSummary: %d sent, %d failed\n", sentCount, errorCount)
		return nil
	},
}

var mailScheduledClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear scheduled emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName, _ := cmd.Flags().GetString("account")
		sentOnly, _ := cmd.Flags().GetBool("sent")
		all, _ := cmd.Flags().GetBool("all")

		if sentOnly {
			if err := gmail.ClearSentScheduledEmails(accountName); err != nil {
				return err
			}
			output.PrintSuccess("Cleared sent scheduled emails")
		} else if all {
			if err := gmail.ClearAllScheduledEmails(accountName); err != nil {
				return err
			}
			output.PrintSuccess("Cleared all scheduled emails")
		} else {
			return fmt.Errorf("specify --sent to clear sent emails or --all to clear all")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mailCmd)
	mailCmd.AddCommand(mailReadCmd)
	mailCmd.AddCommand(mailGetCmd)
	mailCmd.AddCommand(mailDraftCmd)
	mailCmd.AddCommand(mailSendCmd)
	mailCmd.AddCommand(mailSendNowCmd)
	mailCmd.AddCommand(mailScheduleCmd)
	mailCmd.AddCommand(mailScheduledCmd)

	mailScheduledCmd.AddCommand(mailScheduledListCmd)
	mailScheduledCmd.AddCommand(mailScheduledSendCmd)
	mailScheduledCmd.AddCommand(mailScheduledClearCmd)

	// Common flags
	addAccountFlag := func(cmd *cobra.Command) {
		cmd.Flags().StringP("account", "a", "", "Account to use (default: default account)")
	}

	addEmailFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringSliceP("to", "t", nil, "Recipient email addresses")
		cmd.Flags().StringSlice("cc", nil, "CC email addresses")
		cmd.Flags().StringSlice("bcc", nil, "BCC email addresses")
		cmd.Flags().StringP("subject", "s", "", "Email subject")
		cmd.Flags().StringP("body", "b", "", "Email body")
		cmd.Flags().Bool("html", false, "Body is HTML format")
	}

	// mailReadCmd flags
	addAccountFlag(mailReadCmd)
	mailReadCmd.Flags().Bool("all", false, "Read from all accounts")
	mailReadCmd.Flags().StringP("query", "q", "", "Gmail search query")
	mailReadCmd.Flags().Int64P("limit", "n", 25, "Maximum number of emails to fetch")

	// mailGetCmd flags
	addAccountFlag(mailGetCmd)

	// mailDraftCmd flags
	addAccountFlag(mailDraftCmd)
	addEmailFlags(mailDraftCmd)

	// mailSendCmd flags
	addAccountFlag(mailSendCmd)

	// mailSendNowCmd flags
	addAccountFlag(mailSendNowCmd)
	addEmailFlags(mailSendNowCmd)

	// mailScheduleCmd flags
	addAccountFlag(mailScheduleCmd)
	addEmailFlags(mailScheduleCmd)
	mailScheduleCmd.Flags().String("at", "", "Schedule time (ISO 8601 format)")

	// mailScheduledListCmd flags
	addAccountFlag(mailScheduledListCmd)
	mailScheduledListCmd.Flags().Bool("pending", false, "Show only pending emails")

	// mailScheduledSendCmd flags
	addAccountFlag(mailScheduledSendCmd)
	mailScheduledSendCmd.Flags().Bool("dry-run", false, "Show what would be sent without sending")

	// mailScheduledClearCmd flags
	addAccountFlag(mailScheduledClearCmd)
	mailScheduledClearCmd.Flags().Bool("sent", false, "Clear sent emails only")
	mailScheduledClearCmd.Flags().Bool("all", false, "Clear all scheduled emails")
}

// parseDateTime parses a datetime string in various formats
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			// If no timezone, assume local
			if format == "2006-01-02T15:04:05" || format == "2006-01-02 15:04:05" ||
				format == "2006-01-02T15:04" || format == "2006-01-02 15:04" {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse datetime: %s", s)
}
