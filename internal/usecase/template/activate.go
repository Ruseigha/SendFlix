package template

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// ActivateTemplateUseCase handles template activation
type ActivateTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewActivateTemplateUseCase creates new use case
func NewActivateTemplateUseCase(
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *ActivateTemplateUseCase {
	return &ActivateTemplateUseCase{
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// ActivateTemplateRequest represents activate request
type ActivateTemplateRequest struct {
	ID          string
	ActivatedBy string
}

// ActivateTemplateResponse represents activate response
type ActivateTemplateResponse struct {
	ID          string
	Name        string
	Status      domain.TemplateStatus
	Version     int
	ActivatedAt time.Time
	Message     string
}

// Execute activates template
//
// ACTIVATION PROCESS:
// 1. Validate template exists
// 2. Check current status (must be draft or archived)
// 3. Validate template can be rendered
// 4. Test with sample data if available
// 5. Change status to active
// 6. Update in repository
//
// SAFETY CHECKS:
// - Template must validate successfully
// - Template must render without errors
// - Sample data test must pass
//
// PARAMETERS:
// - ctx: Context
// - req: Activation request
//
// RETURNS:
// - *ActivateTemplateResponse: Activation result
// - error: If activation fails
func (uc *ActivateTemplateUseCase) Execute(ctx context.Context, req ActivateTemplateRequest) (*ActivateTemplateResponse, error) {
	uc.logger.Info("activating template", "id", req.ID)

	// Get template
	template, err := uc.templateRepo.FindByID(ctx, req.ID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.activate.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	// Check current status
	if template.Status == domain.TemplateStatusActive {
		uc.logger.Info("template already active", "id", req.ID)
		uc.metrics.IncrementCounter("template.activate.already_active", 1)
		return &ActivateTemplateResponse{
			ID:          template.ID,
			Name:        template.Name,
			Status:      template.Status,
			Version:     template.Version,
			ActivatedAt: template.UpdatedAt,
			Message:     "Template is already active",
		}, nil
	}

	// Only draft and archived templates can be activated
	if template.Status != domain.TemplateStatusDraft && template.Status != domain.TemplateStatusArchived {
		uc.metrics.IncrementCounter("template.activate.invalid_status", 1)
		return nil, errors.New(
			"INVALID_STATUS",
			"Only draft or archived templates can be activated",
			400,
		)
	}

	// Validate template
	if err := template.Validate(); err != nil {
		uc.logger.Error("template validation failed", "error", err)
		uc.metrics.IncrementCounter("template.activate.validation_failed", 1)
		return nil, errors.Wrap(err, "VALIDATION_FAILED", "Template validation failed", 400)
	}

	// Test render with sample data
	if len(template.SampleData) > 0 {
		uc.logger.Debug("testing template render with sample data", "id", req.ID)

		rendered, err := template.Render(template.SampleData)
		if err != nil {
			uc.logger.Error("template render test failed", "error", err)
			uc.metrics.IncrementCounter("template.activate.render_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR",
				"Template failed to render with sample data. Fix errors before activating",
				400)
		}

		// Validate rendered content is not empty
		if rendered.Subject == "" {
			uc.metrics.IncrementCounter("template.activate.empty_subject", 1)
			return nil, errors.New(
				"TEMPLATE_ERROR",
				"Rendered subject is empty",
				400,
			)
		}

		if template.Type == domain.TemplateTypeHTML && rendered.BodyHTML == "" {
			uc.metrics.IncrementCounter("template.activate.empty_html", 1)
			return nil, errors.New(
				"TEMPLATE_ERROR",
				"Rendered HTML body is empty",
				400,
			)
		}

		if template.Type == domain.TemplateTypeText && rendered.BodyText == "" {
			uc.metrics.IncrementCounter("template.activate.empty_text", 1)
			return nil, errors.New(
				"TEMPLATE_ERROR",
				"Rendered text body is empty",
				400,
			)
		}

		uc.logger.Debug("template render test passed", "id", req.ID)
	}

	// Test render with default variables if no sample data
	if len(template.SampleData) == 0 && len(template.DefaultVariables) > 0 {
		uc.logger.Debug("testing template render with default variables", "id", req.ID)

		if _, err := template.Render(template.DefaultVariables); err != nil {
			uc.logger.Error("template render test with defaults failed", "error", err)
			uc.metrics.IncrementCounter("template.activate.default_render_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR",
				"Template failed to render with default variables",
				400)
		}
	}

	// Check for required variables without defaults
	missingDefaults := uc.checkMissingDefaults(template)
	if len(missingDefaults) > 0 {
		uc.logger.Warn("template has required variables without defaults",
			"id", req.ID,
			"missing", missingDefaults)
		// This is a warning, not an error - templates can have required variables
	}

	// Activate template
	template.Status = domain.TemplateStatusActive
	template.UpdatedBy = req.ActivatedBy
	template.UpdatedAt = time.Now()

	if err := uc.templateRepo.Update(ctx, template); err != nil {
		uc.logger.Error("failed to activate template", "error", err)
		uc.metrics.IncrementCounter("template.activate.update_failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to activate template", 500)
	}

	uc.logger.Info("template activated successfully",
		"id", template.ID,
		"name", template.Name,
		"version", template.Version)
	uc.metrics.IncrementCounter("template.activate.success", 1)

	return &ActivateTemplateResponse{
		ID:          template.ID,
		Name:        template.Name,
		Status:      template.Status,
		Version:     template.Version,
		ActivatedAt: template.UpdatedAt,
		Message:     "Template activated successfully",
	}, nil
}

// DeactivateTemplate deactivates an active template
//
// DEACTIVATION:
// Changes status from active to draft.
// Template can be edited and re-activated.
//
// PARAMETERS:
// - ctx: Context
// - templateID: Template to deactivate
// - deactivatedBy: Who is deactivating
//
// RETURNS:
// - error: If deactivation fails
func (uc *ActivateTemplateUseCase) DeactivateTemplate(ctx context.Context, templateID, deactivatedBy string) error {
	uc.logger.Info("deactivating template", "id", templateID)

	// Get template
	template, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		return errors.ErrTemplateNotFound
	}

	// Check if active
	if template.Status != domain.TemplateStatusActive {
		uc.metrics.IncrementCounter("template.deactivate.not_active", 1)
		return errors.New(
			"INVALID_STATE",
			"Template is not active",
			400,
		)
	}

	// Deactivate
	template.Status = domain.TemplateStatusDraft
	template.UpdatedBy = deactivatedBy
	template.UpdatedAt = time.Now()

	if err := uc.templateRepo.Update(ctx, template); err != nil {
		uc.logger.Error("failed to deactivate template", "error", err)
		uc.metrics.IncrementCounter("template.deactivate.failed", 1)
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to deactivate template", 500)
	}

	uc.logger.Info("template deactivated", "id", templateID)
	uc.metrics.IncrementCounter("template.deactivate.success", 1)

	return nil
}

// checkMissingDefaults checks for required variables without defaults
func (uc *ActivateTemplateUseCase) checkMissingDefaults(template *domain.Template) []string {
	missing := []string{}

	for _, required := range template.RequiredVariables {
		if _, exists := template.DefaultVariables[required]; !exists {
			missing = append(missing, required)
		}
	}

	return missing
}