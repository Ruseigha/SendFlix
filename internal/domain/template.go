package domain

import (
	"bytes"
	"fmt"
	"html/template"
	textTemplate "text/template"
	"time"
)

// TemplateType represents template type
type TemplateType string

const (
	TemplateTypeHTML TemplateType = "html"
	TemplateTypeText TemplateType = "text"
	TemplateTypeBoth TemplateType = "both"
)

// TemplateStatus represents template status
type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "draft"
	TemplateStatusActive   TemplateStatus = "active"
	TemplateStatusArchived TemplateStatus = "archived"
)

// Template represents an email template
type Template struct {
	ID                string
	Name              string
	Description       string
	Type              TemplateType
	Status            TemplateStatus
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

// RenderedTemplate represents a rendered template
type RenderedTemplate struct {
	Subject  string
	BodyHTML string
	BodyText string
}

// Validate validates template
func (t *Template) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if t.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if t.Type == TemplateTypeHTML && t.BodyHTML == "" {
		return fmt.Errorf("HTML body is required for HTML templates")
	}

	if t.Type == TemplateTypeText && t.BodyText == "" {
		return fmt.Errorf("text body is required for text templates")
	}

	if t.Type == TemplateTypeBoth && (t.BodyHTML == "" || t.BodyText == "") {
		return fmt.Errorf("both HTML and text bodies are required")
	}

	return nil
}

// Render renders template with data
func (t *Template) Render(data map[string]interface{}) (*RenderedTemplate, error) {
	// Validate required variables
	for _, varName := range t.RequiredVariables {
		if _, exists := data[varName]; !exists {
			return nil, fmt.Errorf("required variable missing: %s", varName)
		}
	}

	// Merge with default variables
	mergedData := make(map[string]interface{})
	for k, v := range t.DefaultVariables {
		mergedData[k] = v
	}
	for k, v := range data {
		mergedData[k] = v
	}

	rendered := &RenderedTemplate{}

	// Render subject
	subjectTmpl, err := template.New("subject").Parse(t.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, mergedData); err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}
	rendered.Subject = subjectBuf.String()

	// Render HTML body
	if t.BodyHTML != "" {
		htmlTmpl, err := template.New("html").Parse(t.BodyHTML)
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML template: %w", err)
		}

		var htmlBuf bytes.Buffer
		if err := htmlTmpl.Execute(&htmlBuf, mergedData); err != nil {
			return nil, fmt.Errorf("failed to render HTML: %w", err)
		}
		rendered.BodyHTML = htmlBuf.String()
	}

	// Render text body
	if t.BodyText != "" {
		textTmpl, err := textTemplate.New("text").Parse(t.BodyText)
		if err != nil {
			return nil, fmt.Errorf("failed to parse text template: %w", err)
		}

		var textBuf bytes.Buffer
		if err := textTmpl.Execute(&textBuf, mergedData); err != nil {
			return nil, fmt.Errorf("failed to render text: %w", err)
		}
		rendered.BodyText = textBuf.String()
	}

	return rendered, nil
}

// IsActive checks if template is active
func (t *Template) IsActive() bool {
	return t.Status == TemplateStatusActive
}

// IncrementUsage increments usage count
func (t *Template) IncrementUsage() {
	t.UsageCount++
	now := time.Now()
	t.LastUsedAt = &now
}