package email

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/Ruseigha/SendFlix/pkg/utils"
)

// ScheduleEmailUseCase handles email scheduling
type ScheduleEmailUseCase struct {
	emailRepo    domain.EmailRepository
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewScheduleEmailUseCase creates new use case
func NewScheduleEmailUseCase(
	emailRepo domain.EmailRepository,
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *ScheduleEmailUseCase {
	return &ScheduleEmailUseCase{
		emailRepo:    emailRepo,
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// ScheduleEmailRequest represents schedule request
type ScheduleEmailRequest struct {
	Email       SendEmailRequest
	ScheduledAt time.Time
}

// ScheduleEmailResponse represents schedule response
type ScheduleEmailResponse struct {
	EmailID     string
	Status      domain.EmailStatus
	ScheduledAt time.Time
	Message     string
}

// Execute schedules email
func (uc *ScheduleEmailUseCase) Execute(ctx context.Context, req ScheduleEmailRequest) (*ScheduleEmailResponse, error) {
	uc.logger.Info("scheduling email", "scheduled_at", req.ScheduledAt)

	// Validate scheduled time
	if req.ScheduledAt.Before(time.Now()) {
		uc.metrics.IncrementCounter("email.schedule.invalid_time", 1)
		return nil, errors.New("INVALID_TIME", "Scheduled time must be in the future", 400)
	}

	// Max 1 year in future
	maxFuture := time.Now().AddDate(1, 0, 0)
	if req.ScheduledAt.After(maxFuture) {
		uc.metrics.IncrementCounter("email.schedule.too_far", 1)
		return nil, errors.New("INVALID_TIME", "Cannot schedule more than 1 year in advance", 400)
	}

	// Create email entity
	email := &domain.Email{
		ID:           utils.GenerateID("email"),
		To:           req.Email.To,
		CC:           req.Email.CC,
		BCC:          req.Email.BCC,
		From:         req.Email.From,
		ReplyTo:      req.Email.ReplyTo,
		Subject:      req.Email.Subject,
		BodyHTML:     req.Email.BodyHTML,
		BodyText:     req.Email.BodyText,
		TemplateID:   req.Email.TemplateID,
		TemplateData: req.Email.TemplateData,
		Attachments:  req.Email.Attachments,
		Priority:     req.Email.Priority,
		Status:       domain.EmailStatusScheduled,
		ScheduledAt:  &req.ScheduledAt,
		MaxRetries:   3,
		TrackOpens:   req.Email.TrackOpens,
		TrackClicks:  req.Email.TrackClicks,
		Metadata:     req.Email.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set default priority
	if email.Priority == "" {
		email.Priority = domain.EmailPriorityNormal
	}

	// Process template if provided
	if req.Email.TemplateID != "" {
		template, err := uc.templateRepo.FindByID(ctx, req.Email.TemplateID)
		if err != nil {
			uc.logger.Error("template not found", "error", err)
			uc.metrics.IncrementCounter("email.schedule.template_not_found", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Template not found", 404)
		}

		if !template.IsActive() {
			return nil, errors.New("TEMPLATE_ERROR", "Template is not active", 400)
		}

		rendered, err := template.Render(email.TemplateData)
		if err != nil {
			uc.logger.Error("template render failed", "error", err)
			uc.metrics.IncrementCounter("email.schedule.template_render_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Failed to render template", 400)
		}

		email.Subject = rendered.Subject
		email.BodyHTML = rendered.BodyHTML
		email.BodyText = rendered.BodyText
	}

	// Validate email
	if err := email.Validate(); err != nil {
		uc.logger.Error("validation failed", "error", err)
		uc.metrics.IncrementCounter("email.schedule.validation_failed", 1)
		return nil, errors.Wrap(err, "VALIDATION_FAILED", "Email validation failed", 400)
	}

	// Save to repository
	if err := uc.emailRepo.Save(ctx, email); err != nil {
		uc.logger.Error("failed to save scheduled email", "error", err)
		uc.metrics.IncrementCounter("email.schedule.save_failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to save email", 500)
	}

	uc.logger.Info("email scheduled successfully", "email_id", email.ID, "scheduled_at", req.ScheduledAt)
	uc.metrics.IncrementCounter("email.schedule.success", 1)

	return &ScheduleEmailResponse{
		EmailID:     email.ID,
		Status:      email.Status,
		ScheduledAt: req.ScheduledAt,
		Message:     "Email scheduled successfully",
	}, nil
}

// CancelScheduledEmail cancels a scheduled email
func (uc *ScheduleEmailUseCase) CancelScheduledEmail(ctx context.Context, emailID string) error {
	uc.logger.Info("cancelling scheduled email", "email_id", emailID)

	// Get email
	email, err := uc.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		uc.logger.Error("email not found", "error", err)
		return errors.ErrEmailNotFound
	}

	// Check if scheduled
	if email.Status != domain.EmailStatusScheduled {
		uc.metrics.IncrementCounter("email.cancel.not_scheduled", 1)
		return errors.New("INVALID_STATE", "Email is not in scheduled state", 400)
	}

	// Update status
	email.Status = domain.EmailStatusCancelled
	email.UpdatedAt = time.Now()

	if err := uc.emailRepo.Update(ctx, email); err != nil {
		uc.logger.Error("failed to cancel email", "error", err)
		uc.metrics.IncrementCounter("email.cancel.failed", 1)
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to cancel email", 500)
	}

	uc.logger.Info("email cancelled successfully", "email_id", emailID)
	uc.metrics.IncrementCounter("email.cancel.success", 1)

	return nil
}