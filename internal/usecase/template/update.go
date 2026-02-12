package template

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// UpdateTemplateUseCase handles template updates
type UpdateTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewUpdateTemplateUseCase creates new use case
func NewUpdateTemplateUseCase(
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *UpdateTemplateUseCase {
	return &UpdateTemplateUseCase{
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// UpdateTemplateRequest represents update request
//
// PARTIAL UPDATE SUPPORT:
// Only non-nil pointer fields will be updated.
// This allows updating specific fields without affecting others.
//
// EXAMPLE:
//   req := UpdateTemplateRequest{
//       ID: "tpl_123",
//       Subject: strPtr("New Subject"), // Will update
//       BodyHTML: nil,                  // Will not update
//   }
type UpdateTemplateRequest struct {
	ID                string
	Name              *string
	Description       *string
	Type              *domain.TemplateType
	Subject           *string
	BodyHTML          *string
	BodyText          *string
	Language          *string
	Category          *string
	Tags              *[]string
	RequiredVariables *[]string
	OptionalVariables *[]string
	DefaultVariables  *map[string]interface{}
	SampleData        *map[string]interface{}
	UpdatedBy         string
	ForceUpdate       bool // Allow updating active templates
}

// UpdateTemplateResponse represents update response
type UpdateTemplateResponse struct {
	ID        string
	Name      string
	Status    domain.TemplateStatus
	Version   int
	UpdatedAt time.Time
	Message   string
	WasActive bool // Indicates if template was active before update
}

// Execute updates template
func (uc *UpdateTemplateUseCase) Execute(ctx context.Context, req UpdateTemplateRequest) (*UpdateTemplateResponse, error) {
	uc.logger.Info("updating template", "id", req.ID)

	// Get existing template
	template, err := uc.templateRepo.FindByID(ctx, req.ID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.update.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	wasActive := template.Status == domain.TemplateStatusActive

	// Check if template is active
	if template.Status == domain.TemplateStatusActive && !req.ForceUpdate {
		uc.logger.Warn("attempted to update active template without force flag", "id", req.ID)
		uc.metrics.IncrementCounter("template.update.active_without_force", 1)
		return nil, errors.New(
			"TEMPLATE_ACTIVE",
			"Cannot update active template. Set ForceUpdate=true to update and move to draft, or deactivate first",
			400,
		)
	}

	// If force updating active template, move to draft
	if wasActive && req.ForceUpdate {
		uc.logger.Info("force updating active template, moving to draft", "id", req.ID)
		template.Status = domain.TemplateStatusDraft
	}

	// Apply partial updates
	uc.applyUpdates(template, req)

	// Update metadata
	template.UpdatedBy = req.UpdatedBy
	template.UpdatedAt = time.Now()

	// Validate updated template
	if err := template.Validate(); err != nil {
		uc.logger.Error("validation failed", "error", err)
		uc.metrics.IncrementCounter("template.update.validation_failed", 1)
		return nil, errors.Wrap(err, "VALIDATION_FAILED", "Template validation failed", 400)
	}

	// Test render with sample data if provided
	if len(template.SampleData) > 0 {
		if _, err := template.Render(template.SampleData); err != nil {
			uc.logger.Error("template render test failed", "error", err)
			uc.metrics.IncrementCounter("template.update.render_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Template cannot be rendered with sample data", 400)
		}
	}

	// Check for name uniqueness if name was changed
	if req.Name != nil && *req.Name != template.Name {
		existing, err := uc.templateRepo.FindByName(ctx, *req.Name)
		if err == nil && existing != nil && existing.ID != template.ID {
			uc.metrics.IncrementCounter("template.update.name_conflict", 1)
			return nil, errors.New("NAME_CONFLICT", "Template with this name already exists", 409)
		}
	}

	// Update in repository
	if err := uc.templateRepo.Update(ctx, template); err != nil {
		uc.logger.Error("failed to update template", "error", err)
		uc.metrics.IncrementCounter("template.update.failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to update template", 500)
	}

	uc.logger.Info("template updated successfully",
		"id", template.ID,
		"name", template.Name,
		"was_active", wasActive)
	uc.metrics.IncrementCounter("template.update.success", 1)

	message := "Template updated successfully"
	if wasActive && req.ForceUpdate {
		message = "Template updated and moved to draft. Activate to use in production"
	}

	return &UpdateTemplateResponse{
		ID:        template.ID,
		Name:      template.Name,
		Status:    template.Status,
		Version:   template.Version,
		UpdatedAt: template.UpdatedAt,
		Message:   message,
		WasActive: wasActive,
	}, nil
}

// applyUpdates applies partial updates to template
//
// WHY POINTER FIELDS:
// Using pointers allows us to distinguish between:
// - Field not provided (nil) - don't update
// - Field provided with zero value (e.g., empty string) - update to empty
// - Field provided with value - update to value
func (uc *UpdateTemplateUseCase) applyUpdates(template *domain.Template, req UpdateTemplateRequest) {
	if req.Name != nil {
		template.Name = *req.Name
	}

	if req.Description != nil {
		template.Description = *req.Description
	}

	if req.Type != nil {
		template.Type = *req.Type
	}

	if req.Subject != nil {
		template.Subject = *req.Subject
	}

	if req.BodyHTML != nil {
		template.BodyHTML = *req.BodyHTML
	}

	if req.BodyText != nil {
		template.BodyText = *req.BodyText
	}

	if req.Language != nil {
		template.Language = *req.Language
	}

	if req.Category != nil {
		template.Category = *req.Category
	}

	if req.Tags != nil {
		template.Tags = *req.Tags
	}

	if req.RequiredVariables != nil {
		template.RequiredVariables = *req.RequiredVariables
	}

	if req.OptionalVariables != nil {
		template.OptionalVariables = *req.OptionalVariables
	}

	if req.DefaultVariables != nil {
		template.DefaultVariables = *req.DefaultVariables
	}

	if req.SampleData != nil {
		template.SampleData = *req.SampleData
	}
}

// Helper functions for pointer conversions
func strPtr(s string) *string {
	return &s
}

func templateTypePtr(t domain.TemplateType) *domain.TemplateType {
	return &t
}

func strSlicePtr(s []string) *[]string {
	return &s
}

func mapPtr(m map[string]interface{}) *map[string]interface{} {
	return &m
}