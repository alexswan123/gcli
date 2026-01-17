package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/alexandraswan/gcli/internal/auth"
	"github.com/alexandraswan/gcli/internal/config"
	"github.com/alexandraswan/gcli/internal/output"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps the Gmail API client
type Client struct {
	service     *gmail.Service
	accountName string
}

// NewClient creates a new Gmail client for the specified account
func NewClient(ctx context.Context, accountName string, account config.AccountConfig) (*Client, error) {
	httpClient, err := auth.GetClient(ctx, accountName, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &Client{
		service:     service,
		accountName: accountName,
	}, nil
}

// ListMessages lists messages matching the query
func (c *Client) ListMessages(ctx context.Context, query string, maxResults int64) ([]output.EmailSummary, error) {
	req := c.service.Users.Messages.List("me")
	if query != "" {
		req = req.Q(query)
	}
	if maxResults > 0 {
		req = req.MaxResults(maxResults)
	}

	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var summaries []output.EmailSummary
	for _, msg := range resp.Messages {
		summary, err := c.getMessageSummary(ctx, msg.Id)
		if err != nil {
			// Log and continue
			continue
		}
		summary.Account = c.accountName
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// getMessageSummary gets a summary of a single message
func (c *Client) getMessageSummary(ctx context.Context, id string) (output.EmailSummary, error) {
	msg, err := c.service.Users.Messages.Get("me", id).
		Format("metadata").
		MetadataHeaders("From", "Subject", "Date").
		Context(ctx).
		Do()
	if err != nil {
		return output.EmailSummary{}, err
	}

	summary := output.EmailSummary{
		ID:      msg.Id,
		Snippet: msg.Snippet,
	}

	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From":
			summary.From = header.Value
		case "Subject":
			summary.Subject = header.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
				summary.Date = t
			} else if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", header.Value); err == nil {
				summary.Date = t
			}
		}
	}

	// Check for attachments
	if msg.Payload.Parts != nil {
		for _, part := range msg.Payload.Parts {
			if part.Filename != "" {
				summary.HasAttach = true
				break
			}
		}
	}

	return summary, nil
}

// GetMessage gets detailed information about a message
func (c *Client) GetMessage(ctx context.Context, id string) (output.EmailDetail, error) {
	msg, err := c.service.Users.Messages.Get("me", id).
		Format("full").
		Context(ctx).
		Do()
	if err != nil {
		return output.EmailDetail{}, fmt.Errorf("failed to get message: %w", err)
	}

	detail := output.EmailDetail{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Account:  c.accountName,
	}

	// Parse headers
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From":
			detail.From = header.Value
		case "To":
			detail.To = parseAddresses(header.Value)
		case "Cc":
			detail.CC = parseAddresses(header.Value)
		case "Subject":
			detail.Subject = header.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
				detail.Date = t
			} else if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", header.Value); err == nil {
				detail.Date = t
			}
		}
	}

	// Extract body
	detail.Body = extractBody(msg.Payload)

	// Extract attachments
	detail.Attachments = extractAttachmentNames(msg.Payload)

	return detail, nil
}

// extractBody extracts the body text from a message payload
func extractBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}

	// Check for direct body
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Recursively check parts
	if payload.Parts != nil {
		// Prefer text/plain over text/html
		for _, part := range payload.Parts {
			if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(data)
				}
			}
		}

		// Fall back to text/html
		for _, part := range payload.Parts {
			if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return stripHTML(string(data))
				}
			}
		}

		// Recursively check nested parts
		for _, part := range payload.Parts {
			if body := extractBody(part); body != "" {
				return body
			}
		}
	}

	return ""
}

// extractAttachmentNames extracts attachment filenames from a message payload
func extractAttachmentNames(payload *gmail.MessagePart) []string {
	var names []string

	if payload.Filename != "" {
		names = append(names, payload.Filename)
	}

	if payload.Parts != nil {
		for _, part := range payload.Parts {
			names = append(names, extractAttachmentNames(part)...)
		}
	}

	return names
}

// parseAddresses parses a comma-separated list of email addresses
func parseAddresses(s string) []string {
	parts := strings.Split(s, ",")
	var addrs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			addrs = append(addrs, p)
		}
	}
	return addrs
}

// stripHTML removes HTML tags from a string (simple implementation)
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// DraftEmail represents an email to be drafted
type DraftEmail struct {
	To      []string
	CC      []string
	BCC     []string
	Subject string
	Body    string
	IsHTML  bool
}

// CreateDraft creates a draft email
func (c *Client) CreateDraft(ctx context.Context, draft DraftEmail) (string, error) {
	rawMessage := buildRawMessage(draft)

	d := &gmail.Draft{
		Message: &gmail.Message{
			Raw: rawMessage,
		},
	}

	resp, err := c.service.Users.Drafts.Create("me", d).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create draft: %w", err)
	}

	return resp.Id, nil
}

// SendDraft sends an existing draft
func (c *Client) SendDraft(ctx context.Context, draftID string) (string, error) {
	d := &gmail.Draft{
		Id: draftID,
	}

	resp, err := c.service.Users.Drafts.Send("me", d).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send draft: %w", err)
	}

	return resp.Id, nil
}

// SendEmail sends an email directly (without creating a draft first)
func (c *Client) SendEmail(ctx context.Context, email DraftEmail) (string, error) {
	rawMessage := buildRawMessage(email)

	msg := &gmail.Message{
		Raw: rawMessage,
	}

	resp, err := c.service.Users.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return resp.Id, nil
}

// buildRawMessage builds a base64url-encoded RFC 2822 message
func buildRawMessage(email DraftEmail) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	if len(email.CC) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}
	if len(email.BCC) > 0 {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", strings.Join(email.BCC, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if email.IsHTML {
		msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	}

	msg.WriteString("\r\n")
	msg.WriteString(email.Body)

	// Base64url encode
	encoded := base64.URLEncoding.EncodeToString([]byte(msg.String()))
	return encoded
}

// GetAccountName returns the account name for this client
func (c *Client) GetAccountName() string {
	return c.accountName
}
