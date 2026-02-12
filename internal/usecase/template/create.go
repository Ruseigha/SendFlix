package template

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/Ruseigha/SendFlix/pkg/utils"
)

// CreateTemplateUseCase handles template creation
type CreateTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewCreateTemplateUseCase creates new use case
func NewCreateTemplateUseCase(
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *CreateTemplateUseCase {
	return &CreateTemplateUseCase{
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// CreateTemplateRequest represents create request
type CreateTemplateRequest struct {
	Name              string
	Description       string
	Type              domain.TemplateType
	Subject           string
	BodyHTML          string
	BodyText          string
	Language          string
	Category          string
	Tags              []string
	RequiredVariables []string
	OptionalVariables []string
	DefaultVariables  map[string]interface{}
	SampleData        map[string]interface{}
	CreatedBy         string
}

// CreateTemplateResponse represents create response
type CreateTemplateResponse struct {
	ID          string
	Name        string
	Status      domain.TemplateStatus
	Version     int
	CreatedAt   time.Time
	Message     string
}

// Execute creates template
func (uc *CreateTemplateUseCase) Execute(ctx context.Context, req CreateTemplateRequest) (*CreateTemplateResponse, error) {
	uc.logger.Info("creating template", "name", req.Name)

	// Check if template with same name exists
	existing, err := uc.templateRepo.FindByName(ctx, req.Name)
	if err == nil && existing != nil {
		uc.metrics.IncrementCounter("template.create.already_exists", 1)
		return nil, errors.New("TEMPLATE_EXISTS", "Template with this name already exists", 409)
	}

	// Create template entity
	template := &domain.Template{
		ID:                utils.GenerateID("tpl"),
		Name:              req.Name,
		Description:       req.Description,
		Type:              req.Type,
		Status:            domain.TemplateStatusDraft,
		Version:           1,
		Subject:           req.Subject,
		BodyHTML:          req.BodyHTML,
		BodyText:          req.BodyText,
		Language:          req.Language,
		Category:          req.Category,
		Tags:              req.Tags,
		RequiredVariables: req.RequiredVariables,
		OptionalVariables: req.OptionalVariables,
		DefaultVariables:  req.DefaultVariables,
		SampleData:        req.SampleData,
		CreatedBy:         req.CreatedBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		UsageCount:        0,
	}

	// Set defaults
	if template.Type == "" {
		template.Type = domain.TemplateTypeBoth
	}
	if template.Language == "" {
		template.Language = "en"
	}

	// Validate template
	if err := template.Validate(); err != nil {
		uc.logger.Error("validation failed", "error", err)
		uc.metrics.IncrementCounter("template.create.validation_failed", 1)
		return nil, errors.Wrap(err, "VALIDATION_FAILED", "Template validation failed", 400)
	}

	// Test render with sample data if provided
	if len(template.SampleData) > 0 {
		if _, err := template.Render(template.SampleData); err != nil {
			uc.logger.Error("template render test failed", "error", err)
			uc.metrics.IncrementCounter("template.create.render_failed", 1)
			return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Template cannot be rendered", 400)
		}
	}

	// Save to repository
	if err := uc.templateRepo.Save(ctx, template); err != nil {
		uc.logger.Error("failed to save template", "error", err)
		uc.metrics.IncrementCounter("template.create.save_failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to save template", 500)
	}

	uc.logger.Info("template created successfully", "id", template.ID, "name", template.Name)
	uc.metrics.IncrementCounter("template.create.success", 1)

	return &CreateTemplateResponse{
		ID:        template.ID,
		Name:      template.Name,
		Status:    template.Status,
		Version:   template.Version,
		CreatedAt: template.CreatedAt,
		Message:   "Template created successfully",
	}, nil
}