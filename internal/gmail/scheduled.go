package gmail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
)

const scheduledFileName = "scheduled.json"

// ScheduledEmailData represents the stored scheduled email data
type ScheduledEmailData struct {
	ID          string    `json:"id"`
	Account     string    `json:"account"`
	DraftID     string    `json:"draft_id"`
	To          []string  `json:"to"`
	CC          []string  `json:"cc,omitempty"`
	BCC         []string  `json:"bcc,omitempty"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	IsHTML      bool      `json:"is_html"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
	Sent        bool      `json:"sent"`
	SentAt      time.Time `json:"sent_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// getScheduledPath returns the path to the scheduled emails file
func getScheduledPath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, scheduledFileName), nil
}

// LoadScheduledEmails loads all scheduled emails
func LoadScheduledEmails() ([]ScheduledEmailData, error) {
	path, err := getScheduledPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduledEmailData{}, nil
		}
		return nil, fmt.Errorf("failed to read scheduled emails: %w", err)
	}

	var emails []ScheduledEmailData
	if err := json.Unmarshal(data, &emails); err != nil {
		return nil, fmt.Errorf("failed to parse scheduled emails: %w", err)
	}

	return emails, nil
}

// SaveScheduledEmails saves all scheduled emails
func SaveScheduledEmails(emails []ScheduledEmailData) error {
	if err := config.EnsureConfigDir(); err != nil {
		return err
	}

	path, err := getScheduledPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(emails, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled emails: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write scheduled emails: %w", err)
	}

	return nil
}

// AddScheduledEmail adds a new scheduled email
func AddScheduledEmail(email ScheduledEmailData) error {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return err
	}

	email.ID = generateID()
	email.CreatedAt = time.Now()
	email.Sent = false

	emails = append(emails, email)
	return SaveScheduledEmails(emails)
}

// GetScheduledEmailsByAccount returns scheduled emails for a specific account
func GetScheduledEmailsByAccount(accountName string) ([]output.ScheduledEmail, error) {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return nil, err
	}

	var result []output.ScheduledEmail
	for _, e := range emails {
		if accountName == "" || e.Account == accountName {
			result = append(result, output.ScheduledEmail{
				ID:          e.ID,
				Account:     e.Account,
				DraftID:     e.DraftID,
				To:          e.To,
				Subject:     e.Subject,
				ScheduledAt: e.ScheduledAt,
				CreatedAt:   e.CreatedAt,
				Sent:        e.Sent,
				SentAt:      e.SentAt,
				Error:       e.Error,
			})
		}
	}

	return result, nil
}

// GetPendingScheduledEmails returns scheduled emails that are ready to be sent
func GetPendingScheduledEmails(accountName string) ([]ScheduledEmailData, error) {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var pending []ScheduledEmailData
	for _, e := range emails {
		if !e.Sent && e.Error == "" && e.ScheduledAt.Before(now) {
			if accountName == "" || e.Account == accountName {
				pending = append(pending, e)
			}
		}
	}

	return pending, nil
}

// UpdateScheduledEmail updates a scheduled email
func UpdateScheduledEmail(id string, updateFn func(*ScheduledEmailData)) error {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return err
	}

	found := false
	for i := range emails {
		if emails[i].ID == id {
			updateFn(&emails[i])
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("scheduled email '%s' not found", id)
	}

	return SaveScheduledEmails(emails)
}

// MarkScheduledEmailSent marks a scheduled email as sent
func MarkScheduledEmailSent(id string, messageID string) error {
	return UpdateScheduledEmail(id, func(e *ScheduledEmailData) {
		e.Sent = true
		e.SentAt = time.Now()
	})
}

// MarkScheduledEmailError marks a scheduled email with an error
func MarkScheduledEmailError(id string, errMsg string) error {
	return UpdateScheduledEmail(id, func(e *ScheduledEmailData) {
		e.Error = errMsg
	})
}

// ClearSentScheduledEmails removes all sent scheduled emails
func ClearSentScheduledEmails(accountName string) error {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return err
	}

	var remaining []ScheduledEmailData
	for _, e := range emails {
		// Keep if not sent, or if filtering by account and this is a different account
		if !e.Sent || (accountName != "" && e.Account != accountName) {
			remaining = append(remaining, e)
		}
	}

	return SaveScheduledEmails(remaining)
}

// ClearAllScheduledEmails removes all scheduled emails for an account
func ClearAllScheduledEmails(accountName string) error {
	emails, err := LoadScheduledEmails()
	if err != nil {
		return err
	}

	if accountName == "" {
		// Clear all
		return SaveScheduledEmails([]ScheduledEmailData{})
	}

	var remaining []ScheduledEmailData
	for _, e := range emails {
		if e.Account != accountName {
			remaining = append(remaining, e)
		}
	}

	return SaveScheduledEmails(remaining)
}

// generateID generates a simple unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
