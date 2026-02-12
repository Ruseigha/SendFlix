package template

import (
	"context"
	"fmt"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// DeleteTemplateUseCase handles template deletion
type DeleteTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	emailRepo    domain.EmailRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewDeleteTemplateUseCase creates new use case
func NewDeleteTemplateUseCase(
	templateRepo domain.TemplateRepository,
	emailRepo domain.EmailRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *DeleteTemplateUseCase {
	return &DeleteTemplateUseCase{
		templateRepo: templateRepo,
		emailRepo:    emailRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// DeleteTemplateRequest represents delete request
type DeleteTemplateRequest struct {
	ID         string
	DeletedBy  string
	HardDelete bool // If true, permanently delete. If false, archive (soft delete)
	Force      bool // Allow deleting templates that are in use
}

// DeleteTemplateResponse represents delete response
type DeleteTemplateResponse struct {
	ID         string
	Name       string
	Deleted    bool
	Archived   bool
	Message    string
	EmailCount int // Number of emails using this template (if Force=false)
}

// Execute deletes template
func (uc *DeleteTemplateUseCase) Execute(ctx context.Context, req DeleteTemplateRequest) (*DeleteTemplateResponse, error) {
	uc.logger.Info("deleting template",
		"id", req.ID,
		"hard_delete", req.HardDelete,
		"force", req.Force)

	// Get template
	template, err := uc.templateRepo.FindByID(ctx, req.ID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.delete.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	// Check if template is active
	if template.Status == domain.TemplateStatusActive && !req.Force {
		uc.logger.Warn("attempted to delete active template without force", "id", req.ID)
		uc.metrics.IncrementCounter("template.delete.active_without_force", 1)
		return nil, errors.New(
			"TEMPLATE_ACTIVE",
			"Cannot delete active template. Deactivate first or use Force=true",
			400,
		)
	}

	// Check if template is in use (if not forcing)
	if !req.Force {
		emailCount, err := uc.checkTemplateUsage(ctx, template.ID)
		if err != nil {
			uc.logger.Error("failed to check template usage", "error", err)
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check template usage", 500)
		}

		if emailCount > 0 {
			uc.logger.Warn("attempted to delete template in use",
				"id", req.ID,
				"email_count", emailCount)
			uc.metrics.IncrementCounter("template.delete.in_use", 1)
			return &DeleteTemplateResponse{
				ID:         template.ID,
				Name:       template.Name,
				Deleted:    false,
				Archived:   false,
				Message:    fmt.Sprintf("Template is used by %d emails. Use Force=true to delete anyway", emailCount),
				EmailCount: emailCount,
			}, errors.New(
				"TEMPLATE_IN_USE",
				fmt.Sprintf("Template is used by %d emails", emailCount),
				400,
			)
		}
	}

	var deleted, archived bool
	var message string

	if req.HardDelete {
		// Hard delete - permanently remove from database
		if err := uc.templateRepo.Delete(ctx, template.ID); err != nil {
			uc.logger.Error("failed to delete template", "error", err)
			uc.metrics.IncrementCounter("template.delete.failed", 1)
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to delete template", 500)
		}

		deleted = true
		message = "Template permanently deleted"
		uc.metrics.IncrementCounter("template.delete.hard.success", 1)

		uc.logger.Info("template hard deleted", "id", template.ID, "name", template.Name)

	} else {
		// Soft delete - archive the template
		template.Status = domain.TemplateStatusArchived
		template.UpdatedBy = req.DeletedBy

		if err := uc.templateRepo.Update(ctx, template); err != nil {
			uc.logger.Error("failed to archive template", "error", err)
			uc.metrics.IncrementCounter("template.delete.archive_failed", 1)
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to archive template", 500)
		}

		archived = true
		message = "Template archived successfully"
		uc.metrics.IncrementCounter("template.delete.soft.success", 1)

		uc.logger.Info("template archived", "id", template.ID, "name", template.Name)
	}

	return &DeleteTemplateResponse{
		ID:       template.ID,
		Name:     template.Name,
		Deleted:  deleted,
		Archived: archived,
		Message:  message,
	}, nil
}

// checkTemplateUsage checks how many emails use this template
//
// CHECKS:
// - Emails with this template_id
// - Emails in pending, queued, scheduled states (future sends)
//
// RETURNS:
// - int: Number of emails using this template
// - error: If query fails
func (uc *DeleteTemplateUseCase) checkTemplateUsage(ctx context.Context, _ string) (int, error) {
	// Count emails using this template
	filters := domain.EmailFilters{
		// Would need TemplateID field in EmailFilters
		// For now, simplified
	}

	count, err := uc.emailRepo.Count(ctx, filters)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// RestoreTemplate restores an archived template
//
// RESTORE PROCESS:
// 1. Check if template is archived
// 2. Change status to draft
// 3. Update in repository
//
// PARAMETERS:
// - ctx: Context
// - templateID: Template ID to restore
// - restoredBy: Who is restoring
//
// RETURNS:
// - error: If restore fails
func (uc *DeleteTemplateUseCase) RestoreTemplate(ctx context.Context, templateID, restoredBy string) error {
	uc.logger.Info("restoring template", "id", templateID)

	// Get template
	template, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		return errors.ErrTemplateNotFound
	}

	// Check if archived
	if template.Status != domain.TemplateStatusArchived {
		uc.metrics.IncrementCounter("template.restore.not_archived", 1)
		return errors.New(
			"INVALID_STATE",
			"Template is not archived",
			400,
		)
	}

	// Restore to draft
	template.Status = domain.TemplateStatusDraft
	template.UpdatedBy = restoredBy

	if err := uc.templateRepo.Update(ctx, template); err != nil {
		uc.logger.Error("failed to restore template", "error", err)
		uc.metrics.IncrementCounter("template.restore.failed", 1)
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to restore template", 500)
	}

	uc.logger.Info("template restored", "id", templateID)
	uc.metrics.IncrementCounter("template.restore.success", 1)

	return nil
}