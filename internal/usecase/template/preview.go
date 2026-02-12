package template

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// PreviewTemplateUseCase handles template preview
type PreviewTemplateUseCase struct {
	templateRepo domain.TemplateRepository
	logger       logger.Logger
	metrics      metrics.MetricsCollector
}

// NewPreviewTemplateUseCase creates new use case
func NewPreviewTemplateUseCase(
	templateRepo domain.TemplateRepository,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *PreviewTemplateUseCase {
	return &PreviewTemplateUseCase{
		templateRepo: templateRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// PreviewTemplateRequest represents preview request
type PreviewTemplateRequest struct {
	TemplateID string
	Data       map[string]interface{}
	UseSample  bool // Use template's sample data if true
}

// PreviewTemplateResponse represents preview response
type PreviewTemplateResponse struct {
	TemplateID   string
	TemplateName string
	Subject      string
	BodyHTML     string
	BodyText     string
	DataUsed     map[string]interface{}
	Warnings     []string
}

// Execute previews template
//
// PREVIEW PROCESS:
// 1. Load template
// 2. Determine data to use (provided, sample, or default)
// 3. Render template with data
// 4. Return rendered content
//
// DATA PRIORITY:
// 1. Provided data (highest priority)
// 2. Sample data (if UseSample=true)
// 3. Default variables (lowest priority)
//
// WARNINGS:
// - Missing required variables
// - Variables in data but not in template
// - Empty rendered content
//
// PARAMETERS:
// - ctx: Context
// - req: Preview request
//
// RETURNS:
// - *PreviewTemplateResponse: Rendered preview
// - error: If preview fails
func (uc *PreviewTemplateUseCase) Execute(ctx context.Context, req PreviewTemplateRequest) (*PreviewTemplateResponse, error) {
	uc.logger.Info("previewing template", "template_id", req.TemplateID)

	// Get template
	template, err := uc.templateRepo.FindByID(ctx, req.TemplateID)
	if err != nil {
		uc.logger.Error("template not found", "error", err)
		uc.metrics.IncrementCounter("template.preview.not_found", 1)
		return nil, errors.ErrTemplateNotFound
	}

	// Determine data to use
	data := uc.determinePreviewData(template, req)
	warnings := []string{}

	// Check for missing required variables
	missingVars := uc.checkMissingVariables(template, data)
	if len(missingVars) > 0 {
		warnings = append(warnings,
			"Missing required variables: "+joinStrings(missingVars, ", "))
		uc.logger.Warn("preview with missing variables",
			"template_id", req.TemplateID,
			"missing", missingVars)
	}

	// Check for unused variables in data
	unusedVars := uc.checkUnusedVariables(template, data)
	if len(unusedVars) > 0 {
		warnings = append(warnings,
			"Unused variables in data: "+joinStrings(unusedVars, ", "))
		uc.logger.Debug("preview with unused variables",
			"template_id", req.TemplateID,
			"unused", unusedVars)
	}

	// Render template
	rendered, err := template.Render(data)
	if err != nil {
		uc.logger.Error("template render failed", "error", err)
		uc.metrics.IncrementCounter("template.preview.render_failed", 1)
		return nil, errors.Wrap(err, "TEMPLATE_ERROR", "Failed to render template", 400)
	}

	// Check for empty content
	if rendered.Subject == "" {
		warnings = append(warnings, "Rendered subject is empty")
	}
	if template.Type == domain.TemplateTypeHTML && rendered.BodyHTML == "" {
		warnings = append(warnings, "Rendered HTML body is empty")
	}
	if template.Type == domain.TemplateTypeText && rendered.BodyText == "" {
		warnings = append(warnings, "Rendered text body is empty")
	}

	uc.logger.Info("template preview generated",
		"template_id", req.TemplateID,
		"warnings", len(warnings))
	uc.metrics.IncrementCounter("template.preview.success", 1)

	return &PreviewTemplateResponse{
		TemplateID:   template.ID,
		TemplateName: template.Name,
		Subject:      rendered.Subject,
		BodyHTML:     rendered.BodyHTML,
		BodyText:     rendered.BodyText,
		DataUsed:     data,
		Warnings:     warnings,
	}, nil
}

// determinePreviewData determines which data to use for preview
//
// PRIORITY:
// 1. Provided data (merged with defaults)
// 2. Sample data (if UseSample=true)
// 3. Default variables only
func (uc *PreviewTemplateUseCase) determinePreviewData(
	template *domain.Template,
	req PreviewTemplateRequest,
) map[string]interface{} {
	data := make(map[string]interface{})

	// Start with default variables
	for k, v := range template.DefaultVariables {
		data[k] = v
	}

	// Use sample data if requested
	if req.UseSample && len(template.SampleData) > 0 {
		for k, v := range template.SampleData {
			data[k] = v
		}
	}

	// Override with provided data
	for k, v := range req.Data {
		data[k] = v
	}

	return data
}

// checkMissingVariables checks for required variables not in data
func (uc *PreviewTemplateUseCase) checkMissingVariables(
	template *domain.Template,
	data map[string]interface{},
) []string {
	missing := []string{}

	for _, required := range template.RequiredVariables {
		if _, exists := data[required]; !exists {
			missing = append(missing, required)
		}
	}

	return missing
}

// checkUnusedVariables checks for variables in data not used by template
func (uc *PreviewTemplateUseCase) checkUnusedVariables(
	template *domain.Template,
	data map[string]interface{},
) []string {
	// Combine required and optional variables
	knownVars := make(map[string]bool)
	for _, v := range template.RequiredVariables {
		knownVars[v] = true
	}
	for _, v := range template.OptionalVariables {
		knownVars[v] = true
	}

	// Find unused
	unused := []string{}
	for key := range data {
		if !knownVars[key] {
			unused = append(unused, key)
		}
	}

	return unused
}

// CloneTemplate creates a copy of an existing template
//
// CLONE PROCESS:
// 1. Load source template
// 2. Create new template with copied content
// 3. Generate new ID and name
// 4. Set status to draft
// 5. Increment version
// 6. Save new template
//
// USE CASES:
// - Create new version of active template
// - Duplicate for different language
// - A/B testing variations
//
// PARAMETERS:
// - ctx: Context
// - sourceID: Template to clone
// - newName: Name for cloned template
// - clonedBy: Who is cloning
//
// RETURNS:
// - *domain.Template: Cloned template
// - error: If clone fails
func (uc *PreviewTemplateUseCase) CloneTemplate(
	ctx context.Context,
	sourceID string,
	newName string,
	clonedBy string,
) (*domain.Template, error) {
	uc.logger.Info("cloning template", "source_id", sourceID, "new_name", newName)

	// Get source template
	source, err := uc.templateRepo.FindByID(ctx, sourceID)
	if err != nil {
		uc.logger.Error("source template not found", "error", err)
		return nil, errors.ErrTemplateNotFound
	}

	// Check if name is available
	existing, _ := uc.templateRepo.FindByName(ctx, newName)
	if existing != nil {
		uc.metrics.IncrementCounter("template.clone.name_conflict", 1)
		return nil, errors.New("NAME_CONFLICT", "Template with this name already exists", 409)
	}

	// Create clone
	clone := &domain.Template{
		ID:                generateID("tpl"),
		Name:              newName,
		Description:       source.Description,
		Type:              source.Type,
		Status:            domain.TemplateStatusDraft,
		Version:           1, // New template starts at version 1
		Subject:           source.Subject,
		BodyHTML:          source.BodyHTML,
		BodyText:          source.BodyText,
		Language:          source.Language,
		Category:          source.Category,
		Tags:              copyStringSlice(source.Tags),
		RequiredVariables: copyStringSlice(source.RequiredVariables),
		OptionalVariables: copyStringSlice(source.OptionalVariables),
		DefaultVariables:  copyMap(source.DefaultVariables),
		SampleData:        copyMap(source.SampleData),
		CreatedBy:         clonedBy,
		UpdatedBy:         "",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		UsageCount:        0,
		LastUsedAt:        nil,
	}

	// Save clone
	if err := uc.templateRepo.Save(ctx, clone); err != nil {
		uc.logger.Error("failed to save cloned template", "error", err)
		uc.metrics.IncrementCounter("template.clone.save_failed", 1)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to save cloned template", 500)
	}

	uc.logger.Info("template cloned successfully",
		"source_id", sourceID,
		"clone_id", clone.ID,
		"new_name", newName)
	uc.metrics.IncrementCounter("template.clone.success", 1)

	return clone, nil
}

// Helper functions

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}

func copyStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func generateID(prefix string) string {
	// Would use utils.GenerateID from pkg/utils
	return prefix + "_" + "generated_id"
}