package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// JSONOutput determines whether to output JSON
var JSONOutput bool

// PrintJSON prints data as formatted JSON
func PrintJSON(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

// EmailSummary represents a summary of an email for display
type EmailSummary struct {
	ID       string    `json:"id"`
	Account  string    `json:"account,omitempty"`
	From     string    `json:"from"`
	Subject  string    `json:"subject"`
	Date     time.Time `json:"date"`
	Snippet  string    `json:"snippet"`
	HasAttach bool     `json:"has_attachments"`
}

// EmailDetail represents detailed email information
type EmailDetail struct {
	ID          string    `json:"id"`
	Account     string    `json:"account,omitempty"`
	ThreadID    string    `json:"thread_id"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	CC          []string  `json:"cc,omitempty"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	Body        string    `json:"body"`
	Attachments []string  `json:"attachments,omitempty"`
}

// CalendarEventSummary represents a summary of a calendar event
type CalendarEventSummary struct {
	ID          string    `json:"id"`
	Account     string    `json:"account,omitempty"`
	CalendarID  string    `json:"calendar_id,omitempty"`
	Summary     string    `json:"summary"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Location    string    `json:"location,omitempty"`
	Status      string    `json:"status"`
	AllDay      bool      `json:"all_day"`
}

// CalendarEventDetail represents detailed calendar event information
type CalendarEventDetail struct {
	ID           string    `json:"id"`
	Account      string    `json:"account,omitempty"`
	CalendarID   string    `json:"calendar_id,omitempty"`
	Summary      string    `json:"summary"`
	Description  string    `json:"description,omitempty"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Location     string    `json:"location,omitempty"`
	Status       string    `json:"status"`
	AllDay       bool      `json:"all_day"`
	Attendees    []string  `json:"attendees,omitempty"`
	Organizer    string    `json:"organizer,omitempty"`
	HtmlLink     string    `json:"html_link,omitempty"`
	Created      time.Time `json:"created,omitempty"`
	Updated      time.Time `json:"updated,omitempty"`
}

// ScheduledEmail represents a scheduled email
type ScheduledEmail struct {
	ID          string    `json:"id"`
	Account     string    `json:"account"`
	DraftID     string    `json:"draft_id"`
	To          []string  `json:"to"`
	Subject     string    `json:"subject"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
	Sent        bool      `json:"sent"`
	SentAt      time.Time `json:"sent_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// PrintEmailList prints a list of emails
func PrintEmailList(emails []EmailSummary) {
	if JSONOutput {
		PrintJSON(emails)
		return
	}

	if len(emails) == 0 {
		fmt.Println("No emails found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFROM\tSUBJECT\tDATE\tACCOUNT")
	fmt.Fprintln(w, "──\t────\t───────\t────\t───────")

	for _, email := range emails {
		from := truncate(email.From, 30)
		subject := truncate(email.Subject, 40)
		date := email.Date.Format("2006-01-02 15:04")
		account := email.Account
		if account == "" {
			account = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			truncate(email.ID, 16), from, subject, date, account)
	}
	w.Flush()
}

// PrintEmailDetail prints detailed email information
func PrintEmailDetail(email EmailDetail) {
	if JSONOutput {
		PrintJSON(email)
		return
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("ID:      %s\n", email.ID)
	if email.Account != "" {
		fmt.Printf("Account: %s\n", email.Account)
	}
	fmt.Printf("From:    %s\n", email.From)
	fmt.Printf("To:      %s\n", strings.Join(email.To, ", "))
	if len(email.CC) > 0 {
		fmt.Printf("CC:      %s\n", strings.Join(email.CC, ", "))
	}
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Printf("Date:    %s\n", email.Date.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	if len(email.Attachments) > 0 {
		fmt.Printf("Attachments: %s\n", strings.Join(email.Attachments, ", "))
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Println(email.Body)
	fmt.Println()
}

// PrintCalendarEventList prints a list of calendar events
func PrintCalendarEventList(events []CalendarEventSummary) {
	if JSONOutput {
		PrintJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No events found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSUMMARY\tSTART\tEND\tLOCATION\tACCOUNT")
	fmt.Fprintln(w, "──\t───────\t─────\t───\t────────\t───────")

	for _, event := range events {
		summary := truncate(event.Summary, 35)
		location := truncate(event.Location, 20)
		if location == "" {
			location = "-"
		}
		account := event.Account
		if account == "" {
			account = "-"
		}

		var startStr, endStr string
		if event.AllDay {
			startStr = event.Start.Format("2006-01-02")
			endStr = "All day"
		} else {
			startStr = event.Start.Format("2006-01-02 15:04")
			endStr = event.End.Format("15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(event.ID, 16), summary, startStr, endStr, location, account)
	}
	w.Flush()
}

// PrintCalendarEventDetail prints detailed calendar event information
func PrintCalendarEventDetail(event CalendarEventDetail) {
	if JSONOutput {
		PrintJSON(event)
		return
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("ID:       %s\n", event.ID)
	if event.Account != "" {
		fmt.Printf("Account:  %s\n", event.Account)
	}
	fmt.Printf("Summary:  %s\n", event.Summary)
	
	if event.AllDay {
		fmt.Printf("Date:     %s (All day)\n", event.Start.Format("Mon, 02 Jan 2006"))
	} else {
		fmt.Printf("Start:    %s\n", event.Start.Format("Mon, 02 Jan 2006 15:04 MST"))
		fmt.Printf("End:      %s\n", event.End.Format("Mon, 02 Jan 2006 15:04 MST"))
	}
	
	if event.Location != "" {
		fmt.Printf("Location: %s\n", event.Location)
	}
	fmt.Printf("Status:   %s\n", event.Status)
	if event.Organizer != "" {
		fmt.Printf("Organizer: %s\n", event.Organizer)
	}
	if len(event.Attendees) > 0 {
		fmt.Printf("Attendees: %s\n", strings.Join(event.Attendees, ", "))
	}
	if event.HtmlLink != "" {
		fmt.Printf("Link:     %s\n", event.HtmlLink)
	}
	fmt.Println(strings.Repeat("─", 80))
	if event.Description != "" {
		fmt.Println()
		fmt.Println(event.Description)
		fmt.Println()
	}
}

// PrintScheduledEmails prints a list of scheduled emails
func PrintScheduledEmails(emails []ScheduledEmail) {
	if JSONOutput {
		PrintJSON(emails)
		return
	}

	if len(emails) == 0 {
		fmt.Println("No scheduled emails found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTO\tSUBJECT\tSCHEDULED FOR\tSTATUS\tACCOUNT")
	fmt.Fprintln(w, "──\t──\t───────\t─────────────\t──────\t───────")

	for _, email := range emails {
		to := truncate(strings.Join(email.To, ", "), 25)
		subject := truncate(email.Subject, 30)
		scheduledAt := email.ScheduledAt.Format("2006-01-02 15:04")
		
		var status string
		if email.Sent {
			status = "✅ Sent"
		} else if email.Error != "" {
			status = "❌ Error"
		} else {
			status = "⏳ Pending"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(email.ID, 8), to, subject, scheduledAt, status, email.Account)
	}
	w.Flush()
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("✅ "+format+"\n", args...)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("⚠️  "+format+"\n", args...)
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ️  "+format+"\n", args...)
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// AccountInfo represents account information for display
type AccountInfo struct {
	Name       string `json:"name"`
	IsDefault  bool   `json:"is_default"`
	HasToken   bool   `json:"has_token"`
	CalendarID string `json:"calendar_id,omitempty"`
}

// PrintAccountList prints a list of accounts
func PrintAccountList(accounts []AccountInfo) {
	if JSONOutput {
		PrintJSON(accounts)
		return
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts configured.")
		fmt.Println("\nRun 'gcli auth add <name>' to add an account.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDEFAULT\tAUTH STATUS\tCALENDAR")
	fmt.Fprintln(w, "────\t───────\t───────────\t────────")

	for _, acc := range accounts {
		def := ""
		if acc.IsDefault {
			def = "✓"
		}
		status := "❌ Not authenticated"
		if acc.HasToken {
			status = "✅ Authenticated"
		}
		calID := acc.CalendarID
		if calID == "" {
			calID = "primary"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", acc.Name, def, status, calID)
	}
	w.Flush()
}
