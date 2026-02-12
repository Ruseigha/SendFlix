package domain

import (
	"fmt"
	"strings"
	"time"
)

// EmailStatus represents email delivery status
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusQueued    EmailStatus = "queued"
	EmailStatusSending   EmailStatus = "sending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusFailed    EmailStatus = "failed"
	EmailStatusScheduled EmailStatus = "scheduled"
	EmailStatusCancelled EmailStatus = "cancelled"
	EmailStatusDeadLetter EmailStatus = "dead_letter"
)

// EmailPriority represents email priority
type EmailPriority string

const (
	EmailPriorityLow    EmailPriority = "low"
	EmailPriorityNormal EmailPriority = "normal"
	EmailPriorityHigh   EmailPriority = "high"
)

// Email represents an email entity
type Email struct {
	ID                string
	To                []string
	CC                []string
	BCC               []string
	From              string
	ReplyTo           string
	Subject           string
	BodyHTML          string
	BodyText          string
	TemplateID        string
	TemplateData      map[string]interface{}
	Attachments       []Attachment
	Priority          EmailPriority
	Status            EmailStatus
	ProviderName      string
	ProviderMessageID string
	ScheduledAt       *time.Time
	RetryCount        int
	MaxRetries        int
	LastError         string
	TrackOpens        bool
	TrackClicks       bool
	Metadata          map[string]interface{}
	CreatedAt         time.Time
	UpdatedAt         time.Time
	SentAt            *time.Time
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	Inline      bool
	ContentID   string
}

// Validate validates email entity
func (e *Email) Validate() error {
	if len(e.To) == 0 && len(e.CC) == 0 && len(e.BCC) == 0 {
		return fmt.Errorf("at least one recipient required")
	}

	for _, addr := range append(append(e.To, e.CC...), e.BCC...) {
		if !isValidEmail(addr) {
			return fmt.Errorf("invalid email address: %s", addr)
		}
	}

	if e.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if e.BodyHTML == "" && e.BodyText == "" && e.TemplateID == "" {
		return fmt.Errorf("email body or template is required")
	}

	return nil
}

// CanRetry checks if email can be retried
func (e *Email) CanRetry() bool {
	return e.Status == EmailStatusFailed && e.RetryCount < e.MaxRetries
}

// MarkAsSent marks email as sent
func (e *Email) MarkAsSent(messageID string) {
	now := time.Now()
	e.Status = EmailStatusSent
	e.ProviderMessageID = messageID
	e.SentAt = &now
	e.UpdatedAt = now
}

// MarkAsFailed marks email as failed
func (e *Email) MarkAsFailed(err error) {
	e.Status = EmailStatusFailed
	e.RetryCount++
	e.LastError = err.Error()
	e.UpdatedAt = time.Now()

	if !e.CanRetry() {
		e.Status = EmailStatusDeadLetter
	}
}

// TotalRecipients returns total number of recipients
func (e *Email) TotalRecipients() int {
	return len(e.To) + len(e.CC) + len(e.BCC)
}

// IsScheduled checks if email is scheduled
func (e *Email) IsScheduled() bool {
	return e.ScheduledAt != nil && e.ScheduledAt.After(time.Now())
}

// Helper function to validate email
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && len(email) > 3
}