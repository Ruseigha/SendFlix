package email

import (
	"context"
	"fmt"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/Ruseigha/SendFlix/pkg/utils"
)

// SendEmailUseCase handles single email sending
type SendEmailUseCase struct {
	emailRepo        domain.EmailRepository
	templateRepo     domain.TemplateRepository
	providerSelector domain.ProviderSelector
	logger           logger.Logger
	metrics          metrics.MetricsCollector
}

// NewSendEmailUseCase creates new use case
func NewSendEmailUseCase(
	emailRepo domain.EmailRepository,
	templateRepo domain.TemplateRepository,
	providerSelector domain.ProviderSelector,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *SendEmailUseCase {
	return &SendEmailUseCase{
		emailRepo:        emailRepo,
		templateRepo:     templateRepo,
		providerSelector: providerSelector,
		logger:           logger,
		metrics:          metrics,
	}
}

// SendEmailRequest represents send request
type SendEmailRequest struct {
	To           []string
	CC           []string
	BCC          []string
	From         string
	ReplyTo      string
	Subject      string
	BodyHTML     string
	BodyText     string
	TemplateID   string
	TemplateData map[string]interface{}
	Attachments  []domain.Attachment
	Priority     domain.EmailPriority
	ProviderName string
	TrackOpens   bool
	TrackClicks  bool
	Metadata     map[string]interface{}
}

// SendEmailResponse represents send response
type SendEmailResponse struct {
	EmailID           string
	Status            domain.EmailStatus
	ProviderName      string
	ProviderMessageID string
	CreatedAt         time.Time
	SentAt            *time.Time
	Message           string
}

// Execute sends email
func (uc *SendEmailUseCase) Execute(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	startTime := time.Now()
	uc.logger.Info("sending email", "to", req.To, "subject", req.Subject)

	// Create email entity
	email := &domain.Email{
		ID:          utils.GenerateID("email"),
		To:          req.To,
		CC:          req.CC,
		BCC:         req.BCC,
		From:        req.From,
		ReplyTo:     req.ReplyTo,
		Subject:     req.Subject,
		BodyHTML:    req.BodyHTML,
		BodyText:    req.BodyText,
		TemplateID:  req.TemplateID,
		TemplateData: req.TemplateData,
		Attachments: req.Attachments,
		Priority:    req.Priority,
		Status:      domain.EmailStatusPending,
		MaxRetries:  3,
		TrackOpens:  req.TrackOpens,
		TrackClicks: req.TrackClicks,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set default priority
	if email.Priority == "" {
		email.Priority = domain.EmailPriorityNormal
	}

	// Process template if provided
	if req.TemplateID != "" {
		if err := uc.processTemplate(ctx, email); err != nil {
			uc.logger.Error("template processing failed", "error", err)
			uc.metrics.IncrementCounter("email.send.template_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Failed to process template", 400)
		}
	}

	// Validate email
	if err := email.Validate(); err != nil {
		uc.logger.Error("validation failed", "error", err)
		uc.metrics.IncrementCounter("email.send.validation_failed", 1)
		return nil, errors.Wrap(err, "VALIDATION_FAILED", "Email validation failed", 400)
	}

	// Save to repository (pending state)
	if err := uc.emailRepo.Save(ctx, email); err != nil {
		uc.logger.Error("failed to save email", "error", err)
		uc.metrics.IncrementCounter("email.send.save_failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to save email", 500)
	}

	// Select provider
	provider, err := uc.providerSelector.SelectProvider(ctx, email)
	if err != nil {
		uc.logger.Error("provider selection failed", "error", err)
		email.Status = domain.EmailStatusFailed
		email.LastError = err.Error()
		uc.emailRepo.Update(ctx, email)
		uc.metrics.IncrementCounter("email.send.provider_selection_failed", 1)
		return nil, errors.Wrap(err, "PROVIDER_ERROR", "No provider available", 503)
	}

	email.ProviderName = provider.Name()

	// Update status to sending
	email.Status = domain.EmailStatusSending
	if err := uc.emailRepo.Update(ctx, email); err != nil {
		uc.logger.Error("failed to update status", "error", err)
	}

	// Send via provider
	messageID, err := provider.Send(ctx, email)
	if err != nil {
		uc.logger.Error("send failed", "error", err, "provider", provider.Name())
		email.MarkAsFailed(err)
		uc.emailRepo.Update(ctx, email)
		uc.metrics.IncrementCounter("email.send.failed", 1, "provider", provider.Name())
		return nil, errors.Wrap(err, "SEND_FAILED", "Failed to send email", 500)
	}

	// Mark as sent
	email.MarkAsSent(messageID)
	if err := uc.emailRepo.Update(ctx, email); err != nil {
		uc.logger.Error("failed to update sent status", "error", err)
	}

	// Increment template usage
	if req.TemplateID != "" {
		if err := uc.templateRepo.IncrementUsage(ctx, req.TemplateID); err != nil {
			uc.logger.Error("failed to increment template usage", "error", err)
		}
	}

	duration := time.Since(startTime)
	uc.logger.Info("email sent successfully",
		"email_id", email.ID,
		"provider", provider.Name(),
		"duration", duration)

	uc.metrics.IncrementCounter("email.send.success", 1, "provider", provider.Name())
	uc.metrics.RecordDuration("email.send.duration", duration, "provider", provider.Name())

	return &SendEmailResponse{
		EmailID:           email.ID,
		Status:            email.Status,
		ProviderName:      email.ProviderName,
		ProviderMessageID: email.ProviderMessageID,
		CreatedAt:         email.CreatedAt,
		SentAt:            email.SentAt,
		Message:           "Email sent successfully",
	}, nil
}

// processTemplate processes template
func (uc *SendEmailUseCase) processTemplate(ctx context.Context, email *domain.Email) error {
	// Get template
	template, err := uc.templateRepo.FindByID(ctx, email.TemplateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Check if active
	if !template.IsActive() {
		return fmt.Errorf("template is not active")
	}

	// Render template
	rendered, err := template.Render(email.TemplateData)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Update email with rendered content
	email.Subject = rendered.Subject
	email.BodyHTML = rendered.BodyHTML
	email.BodyText = rendered.BodyText

	return nil
}
