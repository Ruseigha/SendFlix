package email

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// GetEmailUseCase handles email queries
type GetEmailUseCase struct {
	emailRepo domain.EmailRepository
	logger    logger.Logger
	metrics   metrics.MetricsCollector
}

// NewGetEmailUseCase creates new use case
func NewGetEmailUseCase(
	emailRepo domain.EmailRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *GetEmailUseCase {
	return &GetEmailUseCase{
		emailRepo: emailRepo,
		logger:    logger,
		metrics:   metrics,
	}
}

// EmailDetailResponse represents email detail
type EmailDetailResponse struct {
	ID                string
	To                []string
	CC                []string
	BCC               []string
	From              string
	ReplyTo           string
	Subject           string
	Status            domain.EmailStatus
	Priority          domain.EmailPriority
	ProviderName      string
	ProviderMessageID string
	RetryCount        int
	MaxRetries        int
	LastError         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	SentAt            *time.Time
	ScheduledAt       *time.Time
}

// GetByID gets email by ID
func (uc *GetEmailUseCase) GetByID(ctx context.Context, id string) (*EmailDetailResponse, error) {
	uc.logger.Debug("getting email by ID", "id", id)

	email, err := uc.emailRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error("email not found", "error", err)
		uc.metrics.IncrementCounter("email.get.not_found", 1)
		return nil, errors.ErrEmailNotFound
	}

	uc.metrics.IncrementCounter("email.get.success", 1)

	return &EmailDetailResponse{
		ID:                email.ID,
		To:                email.To,
		CC:                email.CC,
		BCC:               email.BCC,
		From:              email.From,
		ReplyTo:           email.ReplyTo,
		Subject:           email.Subject,
		Status:            email.Status,
		Priority:          email.Priority,
		ProviderName:      email.ProviderName,
		ProviderMessageID: email.ProviderMessageID,
		RetryCount:        email.RetryCount,
		MaxRetries:        email.MaxRetries,
		LastError:         email.LastError,
		CreatedAt:         email.CreatedAt,
		UpdatedAt:         email.UpdatedAt,
		SentAt:            email.SentAt,
		ScheduledAt:       email.ScheduledAt,
	}, nil
}

// ListEmailsRequest represents list request
type ListEmailsRequest struct {
	Filters      domain.EmailFilters
	QueryOptions domain.QueryOptions
}

// ListEmailsResponse represents list response
type ListEmailsResponse struct {
	Emails     []EmailSummaryResponse
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// EmailSummaryResponse represents email summary
type EmailSummaryResponse struct {
	ID           string
	To           []string
	Subject      string
	Status       domain.EmailStatus
	Priority     domain.EmailPriority
	ProviderName string
	CreatedAt    time.Time
	SentAt       *time.Time
}

// ListEmails lists emails with pagination
func (uc *GetEmailUseCase) ListEmails(ctx context.Context, req ListEmailsRequest) (*ListEmailsResponse, error) {
	uc.logger.Debug("listing emails")

	// Get total count
	total, err := uc.emailRepo.Count(ctx, req.Filters)
	if err != nil {
		uc.logger.Error("failed to count emails", "error", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to count emails", 500)
	}

	// Get emails
	var emails []*domain.Email
	if req.Filters.Status != "" {
		emails, err = uc.emailRepo.FindByStatus(ctx, req.Filters.Status, req.QueryOptions)
	} else {
		// Would need FindAll method - simplified here
		emails, err = uc.emailRepo.FindByStatus(ctx, "", req.QueryOptions)
	}

	if err != nil {
		uc.logger.Error("failed to list emails", "error", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to list emails", 500)
	}

	// Convert to summaries
	summaries := make([]EmailSummaryResponse, len(emails))
	for i, email := range emails {
		summaries[i] = EmailSummaryResponse{
			ID:           email.ID,
			To:           email.To,
			Subject:      email.Subject,
			Status:       email.Status,
			Priority:     email.Priority,
			ProviderName: email.ProviderName,
			CreatedAt:    email.CreatedAt,
			SentAt:       email.SentAt,
		}
	}

	// Calculate total pages
	totalPages := int(total) / req.QueryOptions.Limit
	if int(total)%req.QueryOptions.Limit > 0 {
		totalPages++
	}

	return &ListEmailsResponse{
		Emails:     summaries,
		Total:      total,
		Page:       req.QueryOptions.Page,
		Limit:      req.QueryOptions.Limit,
		TotalPages: totalPages,
	}, nil
}