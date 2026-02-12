package template

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// GetTemplateUseCase handles template queries
type GetTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewGetTemplateUseCase creates new use case
func NewGetTemplateUseCase(
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *GetTemplateUseCase {
	return &GetTemplateUseCase{
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// TemplateDetailResponse represents template detail
type TemplateDetailResponse struct {
	ID                string
	Name              string
	Description       string
	Type              domain.TemplateType
	Status            domain.TemplateStatus
	Version           int
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
	UpdatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	UsageCount        int
	LastUsedAt        *time.Time
}

// GetByID gets template by ID
func (uc *GetTemplateUseCase) GetByID(ctx context.Context, id string) (*TemplateDetailResponse, error) {
	uc.logger.Debug("getting template by ID", "id", id)

	template, err := uc.templateRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.get.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	uc.metrics.IncrementCounter("template.get.success", 1)

	return toDetailResponse(template), nil
}

// GetByName gets template by name
func (uc *GetTemplateUseCase) GetByName(ctx context.Context, name string) (*TemplateDetailResponse, error) {
	uc.logger.Debug("getting template by name", "name", name)

	template, err := uc.templateRepo.FindByName(ctx, name)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.get.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	uc.metrics.IncrementCounter("template.get.success", 1)

	return toDetailResponse(template), nil
}

// ListTemplates lists templates
func (uc *GetTemplateUseCase) ListTemplates(
	ctx context.Context,
	filters domain.TemplateFilters,
	opts domain.QueryOptions,
) (*ListTemplatesResponse, error) {
	uc.logger.Debug("listing templates")

	templates, err := uc.templateRepo.FindAll(ctx, filters, opts)
	if err != nil {
		uc.logger.Error("failed to list templates", "error", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to list templates", 500)
	}

	summaries := make([]TemplateSummaryResponse, len(templates))
	for i, tmpl := range templates {
		summaries[i] = TemplateSummaryResponse{
			ID:          tmpl.ID,
			Name:        tmpl.Name,
			Description: tmpl.Description,
			Type:        tmpl.Type,
			Status:      tmpl.Status,
			Version:     tmpl.Version,
			Category:    tmpl.Category,
			Language:    tmpl.Language,
			UsageCount:  tmpl.UsageCount,
			CreatedAt:   tmpl.CreatedAt,
		}
	}

	return &ListTemplatesResponse{
		Templates: summaries,
	}, nil
}

// ListTemplatesResponse represents list response
type ListTemplatesResponse struct {
	Templates []TemplateSummaryResponse
}

// TemplateSummaryResponse represents template summary
type TemplateSummaryResponse struct {
	ID          string
	Name        string
	Description string
	Type        domain.TemplateType
	Status      domain.TemplateStatus
	Version     int
	Category    string
	Language    string
	UsageCount  int
	CreatedAt   time.Time
}

func toDetailResponse(template *domain.Template) *TemplateDetailResponse {
	return &TemplateDetailResponse{
		ID:                template.ID,
		Name:              template.Name,
		Description:       template.Description,
		Type:              template.Type,
		Status:            template.Status,
		Version:           template.Version,
		Subject:           template.Subject,
		BodyHTML:          template.BodyHTML,
		BodyText:          template.BodyText,
		Language:          template.Language,
		Category:          template.Category,
		Tags:              template.Tags,
		RequiredVariables: template.RequiredVariables,
		OptionalVariables: template.OptionalVariables,
		DefaultVariables:  template.DefaultVariables,
		SampleData:        template.SampleData,
		CreatedBy:         template.CreatedBy,
		UpdatedBy:         template.UpdatedBy,
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
		UsageCount:        template.UsageCount,
		LastUsedAt:        template.LastUsedAt,
	}
}