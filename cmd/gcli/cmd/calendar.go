package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexandraswan/gcli/internal/calendar"
	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
	"github.com/spf13/cobra"
)

var calCmd = &cobra.Command{
	Use:     "cal",
	Aliases: []string{"calendar", "c"},
	Short:   "Manage calendar events",
	Long:    `List, create, update, and delete calendar events.`,
}

var calListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Long: `List calendar events within a date range.

By default, shows events for the next 7 days.

Examples:
  gcli cal list                           # Events for next 7 days
  gcli cal list -a work                   # From work calendar
  gcli cal list --all                     # From all accounts
  gcli cal list --from 2024-01-01 --to 2024-01-31`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		allAccounts, _ := cmd.Flags().GetBool("all")
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")
		limit, _ := cmd.Flags().GetInt64("limit")

		// Parse date range
		var from, to time.Time
		var err error

		if fromStr != "" {
			from, err = parseDate(fromStr)
			if err != nil {
				return fmt.Errorf("invalid --from date: %w", err)
			}
		} else {
			from = time.Now()
		}

		if toStr != "" {
			to, err = parseDate(toStr)
			if err != nil {
				return fmt.Errorf("invalid --to date: %w", err)
			}
			// Set to end of day
			to = to.Add(24*time.Hour - time.Second)
		} else {
			to = from.Add(7 * 24 * time.Hour)
		}

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

		var allEvents []output.CalendarEventSummary
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

				client, err := calendar.NewClient(ctx, name, acc)
				if err != nil {
					errChan <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				events, err := client.ListEvents(ctx, from, to, limit)
				if err != nil {
					errChan <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				mu.Lock()
				allEvents = append(allEvents, events...)
				mu.Unlock()
			}(accName)
		}

		wg.Wait()
		close(errChan)

		// Report any errors
		for err := range errChan {
			output.PrintError("%v", err)
		}

		// Sort events by start time
		sortEventsByStart(allEvents)

		output.PrintCalendarEventList(allEvents)
		return nil
	},
}

var calGetCmd = &cobra.Command{
	Use:   "get <event-id>",
	Short: "Get event details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		eventID := args[0]
		accountName, _ := cmd.Flags().GetString("account")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := calendar.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		event, err := client.GetEvent(ctx, eventID)
		if err != nil {
			return err
		}

		output.PrintCalendarEventDetail(event)
		return nil
	},
}

var calAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new calendar event",
	Long: `Create a new calendar event.

For timed events, provide start and end times. For all-day events, use --all-day.

Examples:
  gcli cal add -s "Meeting" --start "2024-12-25T10:00" --end "2024-12-25T11:00"
  gcli cal add -s "Holiday" --start "2024-12-25" --end "2024-12-26" --all-day
  gcli cal add -s "Meeting" --start "2024-12-25T10:00" --end "2024-12-25T11:00" -l "Conference Room"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")
		summary, _ := cmd.Flags().GetString("summary")
		description, _ := cmd.Flags().GetString("description")
		location, _ := cmd.Flags().GetString("location")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		allDay, _ := cmd.Flags().GetBool("all-day")
		attendeesStr, _ := cmd.Flags().GetStringSlice("attendees")

		if summary == "" {
			return fmt.Errorf("summary is required (--summary)")
		}
		if startStr == "" {
			return fmt.Errorf("start time is required (--start)")
		}
		if endStr == "" {
			return fmt.Errorf("end time is required (--end)")
		}

		var start, end time.Time
		var err error

		if allDay {
			start, err = parseDate(startStr)
			if err != nil {
				return fmt.Errorf("invalid start date: %w", err)
			}
			end, err = parseDate(endStr)
			if err != nil {
				return fmt.Errorf("invalid end date: %w", err)
			}
		} else {
			start, err = parseDateTime(startStr)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
			end, err = parseDateTime(endStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := calendar.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		input := calendar.EventInput{
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       start,
			End:         end,
			AllDay:      allDay,
			Attendees:   attendeesStr,
		}

		eventID, err := client.CreateEvent(ctx, input)
		if err != nil {
			return err
		}

		output.PrintSuccess("Event created (ID: %s)", eventID)
		return nil
	},
}

var calUpdateCmd = &cobra.Command{
	Use:   "update <event-id>",
	Short: "Update an existing calendar event",
	Long: `Update an existing calendar event.

Only provided fields will be updated.

Examples:
  gcli cal update EVENT_ID -s "New Title"
  gcli cal update EVENT_ID --start "2024-12-25T14:00" --end "2024-12-25T15:00"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		eventID := args[0]
		accountName, _ := cmd.Flags().GetString("account")
		summary, _ := cmd.Flags().GetString("summary")
		description, _ := cmd.Flags().GetString("description")
		location, _ := cmd.Flags().GetString("location")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		allDay, _ := cmd.Flags().GetBool("all-day")
		attendeesStr, _ := cmd.Flags().GetStringSlice("attendees")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := calendar.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		input := calendar.EventInput{
			Summary:     summary,
			Description: description,
			Location:    location,
			AllDay:      allDay,
			Attendees:   attendeesStr,
		}

		if startStr != "" {
			if allDay {
				input.Start, err = parseDate(startStr)
			} else {
				input.Start, err = parseDateTime(startStr)
			}
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
		}

		if endStr != "" {
			if allDay {
				input.End, err = parseDate(endStr)
			} else {
				input.End, err = parseDateTime(endStr)
			}
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		if err := client.UpdateEvent(ctx, eventID, input); err != nil {
			return err
		}

		output.PrintSuccess("Event updated")
		return nil
	},
}

var calDeleteCmd = &cobra.Command{
	Use:   "delete <event-id>",
	Short: "Delete a calendar event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		eventID := args[0]
		accountName, _ := cmd.Flags().GetString("account")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := calendar.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		if err := client.DeleteEvent(ctx, eventID); err != nil {
			return err
		}

		output.PrintSuccess("Event deleted")
		return nil
	},
}

var calCalendarsCmd = &cobra.Command{
	Use:   "calendars",
	Short: "List available calendars",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accountName, _ := cmd.Flags().GetString("account")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name, acc, err := cfg.GetAccount(accountName)
		if err != nil {
			return err
		}

		client, err := calendar.NewClient(ctx, name, acc)
		if err != nil {
			return err
		}

		calendars, err := client.ListCalendars(ctx)
		if err != nil {
			return err
		}

		if output.JSONOutput {
			output.PrintJSON(calendars)
			return nil
		}

		fmt.Printf("Calendars for account '%s':\n\n", name)
		for _, cal := range calendars {
			primary := ""
			if cal.Primary {
				primary = " (primary)"
			}
			fmt.Printf("  ID: %s%s\n", cal.ID, primary)
			fmt.Printf("  Name: %s\n", cal.Summary)
			if cal.Description != "" {
				fmt.Printf("  Description: %s\n", cal.Description)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(calCmd)
	calCmd.AddCommand(calListCmd)
	calCmd.AddCommand(calGetCmd)
	calCmd.AddCommand(calAddCmd)
	calCmd.AddCommand(calUpdateCmd)
	calCmd.AddCommand(calDeleteCmd)
	calCmd.AddCommand(calCalendarsCmd)

	// Common flags
	addAccountFlag := func(cmd *cobra.Command) {
		cmd.Flags().StringP("account", "a", "", "Account to use (default: default account)")
	}

	addEventFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringP("summary", "s", "", "Event title/summary")
		cmd.Flags().StringP("description", "d", "", "Event description")
		cmd.Flags().StringP("location", "l", "", "Event location")
		cmd.Flags().String("start", "", "Start time (ISO 8601 format)")
		cmd.Flags().String("end", "", "End time (ISO 8601 format)")
		cmd.Flags().Bool("all-day", false, "All-day event")
		cmd.Flags().StringSlice("attendees", nil, "Attendee email addresses")
	}

	// calListCmd flags
	addAccountFlag(calListCmd)
	calListCmd.Flags().Bool("all", false, "List from all accounts")
	calListCmd.Flags().String("from", "", "Start date (YYYY-MM-DD)")
	calListCmd.Flags().String("to", "", "End date (YYYY-MM-DD)")
	calListCmd.Flags().Int64P("limit", "n", 50, "Maximum number of events")

	// calGetCmd flags
	addAccountFlag(calGetCmd)

	// calAddCmd flags
	addAccountFlag(calAddCmd)
	addEventFlags(calAddCmd)

	// calUpdateCmd flags
	addAccountFlag(calUpdateCmd)
	addEventFlags(calUpdateCmd)

	// calDeleteCmd flags
	addAccountFlag(calDeleteCmd)

	// calCalendarsCmd flags
	addAccountFlag(calCalendarsCmd)
}

// parseDate parses a date string
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"02-01-2006",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s (use YYYY-MM-DD format)", s)
}

// sortEventsByStart sorts events by start time
func sortEventsByStart(events []output.CalendarEventSummary) {
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Start.Before(events[i].Start) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}
