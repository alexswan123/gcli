package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/alexandraswan/gcli/internal/auth"
	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Calendar API client
type Client struct {
	service     *calendar.Service
	accountName string
	calendarID  string
}

// NewClient creates a new Calendar client for the specified account
func NewClient(ctx context.Context, accountName string, account config.AccountConfig) (*Client, error) {
	httpClient, err := auth.GetClient(ctx, accountName, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	service, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %w", err)
	}

	calendarID := account.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	return &Client{
		service:     service,
		accountName: accountName,
		calendarID:  calendarID,
	}, nil
}

// ListEvents lists calendar events within the specified time range
func (c *Client) ListEvents(ctx context.Context, from, to time.Time, maxResults int64) ([]output.CalendarEventSummary, error) {
	req := c.service.Events.List(c.calendarID).
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime")

	if maxResults > 0 {
		req = req.MaxResults(maxResults)
	}

	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var summaries []output.CalendarEventSummary
	for _, event := range resp.Items {
		summary := eventToSummary(event)
		summary.Account = c.accountName
		summary.CalendarID = c.calendarID
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetEvent gets detailed information about a specific event
func (c *Client) GetEvent(ctx context.Context, eventID string) (output.CalendarEventDetail, error) {
	event, err := c.service.Events.Get(c.calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return output.CalendarEventDetail{}, fmt.Errorf("failed to get event: %w", err)
	}

	detail := eventToDetail(event)
	detail.Account = c.accountName
	detail.CalendarID = c.calendarID
	return detail, nil
}

// EventInput represents input for creating or updating an event
type EventInput struct {
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Attendees   []string
}

// CreateEvent creates a new calendar event
func (c *Client) CreateEvent(ctx context.Context, input EventInput) (string, error) {
	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Location:    input.Location,
	}

	if input.AllDay {
		event.Start = &calendar.EventDateTime{
			Date: input.Start.Format("2006-01-02"),
		}
		event.End = &calendar.EventDateTime{
			Date: input.End.Format("2006-01-02"),
		}
	} else {
		// Use RFC3339 format which includes timezone offset
		// Don't set TimeZone field - the offset in DateTime is sufficient
		event.Start = &calendar.EventDateTime{
			DateTime: input.Start.Format(time.RFC3339),
		}
		event.End = &calendar.EventDateTime{
			DateTime: input.End.Format(time.RFC3339),
		}
	}

	if len(input.Attendees) > 0 {
		for _, email := range input.Attendees {
			event.Attendees = append(event.Attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
	}

	resp, err := c.service.Events.Insert(c.calendarID, event).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return resp.Id, nil
}

// UpdateEvent updates an existing calendar event
func (c *Client) UpdateEvent(ctx context.Context, eventID string, input EventInput) error {
	// First get the existing event
	event, err := c.service.Events.Get(c.calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Update fields if provided
	if input.Summary != "" {
		event.Summary = input.Summary
	}
	if input.Description != "" {
		event.Description = input.Description
	}
	if input.Location != "" {
		event.Location = input.Location
	}

	if !input.Start.IsZero() {
		if input.AllDay {
			event.Start = &calendar.EventDateTime{
				Date: input.Start.Format("2006-01-02"),
			}
		} else {
			event.Start = &calendar.EventDateTime{
				DateTime: input.Start.Format(time.RFC3339),
			}
		}
	}

	if !input.End.IsZero() {
		if input.AllDay {
			event.End = &calendar.EventDateTime{
				Date: input.End.Format("2006-01-02"),
			}
		} else {
			event.End = &calendar.EventDateTime{
				DateTime: input.End.Format(time.RFC3339),
			}
		}
	}

	if len(input.Attendees) > 0 {
		event.Attendees = nil
		for _, email := range input.Attendees {
			event.Attendees = append(event.Attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
	}

	_, err = c.service.Events.Update(c.calendarID, eventID, event).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteEvent deletes a calendar event
func (c *Client) DeleteEvent(ctx context.Context, eventID string) error {
	err := c.service.Events.Delete(c.calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// GetAccountName returns the account name for this client
func (c *Client) GetAccountName() string {
	return c.accountName
}

// GetCalendarID returns the calendar ID for this client
func (c *Client) GetCalendarID() string {
	return c.calendarID
}

// eventToSummary converts a calendar event to an output summary
func eventToSummary(event *calendar.Event) output.CalendarEventSummary {
	summary := output.CalendarEventSummary{
		ID:       event.Id,
		Summary:  event.Summary,
		Location: event.Location,
		Status:   event.Status,
	}

	// Parse start time
	if event.Start != nil {
		if event.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.Start.DateTime); err == nil {
				summary.Start = t
			}
		} else if event.Start.Date != "" {
			if t, err := time.Parse("2006-01-02", event.Start.Date); err == nil {
				summary.Start = t
				summary.AllDay = true
			}
		}
	}

	// Parse end time
	if event.End != nil {
		if event.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.End.DateTime); err == nil {
				summary.End = t
			}
		} else if event.End.Date != "" {
			if t, err := time.Parse("2006-01-02", event.End.Date); err == nil {
				summary.End = t
			}
		}
	}

	return summary
}

// eventToDetail converts a calendar event to an output detail
func eventToDetail(event *calendar.Event) output.CalendarEventDetail {
	detail := output.CalendarEventDetail{
		ID:          event.Id,
		Summary:     event.Summary,
		Description: event.Description,
		Location:    event.Location,
		Status:      event.Status,
		HtmlLink:    event.HtmlLink,
	}

	// Parse start time
	if event.Start != nil {
		if event.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.Start.DateTime); err == nil {
				detail.Start = t
			}
		} else if event.Start.Date != "" {
			if t, err := time.Parse("2006-01-02", event.Start.Date); err == nil {
				detail.Start = t
				detail.AllDay = true
			}
		}
	}

	// Parse end time
	if event.End != nil {
		if event.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.End.DateTime); err == nil {
				detail.End = t
			}
		} else if event.End.Date != "" {
			if t, err := time.Parse("2006-01-02", event.End.Date); err == nil {
				detail.End = t
			}
		}
	}

	// Parse organizer
	if event.Organizer != nil {
		detail.Organizer = event.Organizer.Email
	}

	// Parse attendees
	for _, attendee := range event.Attendees {
		detail.Attendees = append(detail.Attendees, attendee.Email)
	}

	// Parse created/updated times
	if event.Created != "" {
		if t, err := time.Parse(time.RFC3339, event.Created); err == nil {
			detail.Created = t
		}
	}
	if event.Updated != "" {
		if t, err := time.Parse(time.RFC3339, event.Updated); err == nil {
			detail.Updated = t
		}
	}

	return detail
}

// ListCalendars lists all calendars for the account
func (c *Client) ListCalendars(ctx context.Context) ([]CalendarInfo, error) {
	resp, err := c.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var calendars []CalendarInfo
	for _, cal := range resp.Items {
		calendars = append(calendars, CalendarInfo{
			ID:          cal.Id,
			Summary:     cal.Summary,
			Description: cal.Description,
			Primary:     cal.Primary,
		})
	}

	return calendars, nil
}

// CalendarInfo represents basic calendar information
type CalendarInfo struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Primary     bool   `json:"primary"`
}
