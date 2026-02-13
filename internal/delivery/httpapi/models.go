package httpapi

import "time"

// ============= EMAIL MODELS =============

// SendEmailRequest represents send email request
type SendEmailRequest struct {
	To           []string               `json:"to" binding:"required"`
	CC           []string               `json:"cc,omitempty"`
	BCC          []string               `json:"bcc,omitempty"`
	From         string                 `json:"from,omitempty"`
	ReplyTo      string                 `json:"reply_to,omitempty"`
	Subject      string                 `json:"subject" binding:"required"`
	BodyHTML     string                 `json:"body_html,omitempty"`
	BodyText     string                 `json:"body_text,omitempty"`
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	ProviderName string                 `json:"provider_name,omitempty"`
	TrackOpens   bool                   `json:"track_opens,omitempty"`
	TrackClicks  bool                   `json:"track_clicks,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SendEmailResponse represents send email response
type SendEmailResponse struct {
	EmailID           string     `json:"email_id"`
	Status            string     `json:"status"`
	ProviderName      string     `json:"provider_name"`
	ProviderMessageID string     `json:"provider_message_id"`
	CreatedAt         time.Time  `json:"created_at"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	Message           string     `json:"message"`
}

// SendBulkEmailRequest represents bulk send request
type SendBulkEmailRequest struct {
	Emails      []SendEmailRequest `json:"emails" binding:"required"`
	BatchSize   int                `json:"batch_size,omitempty"`
	StopOnError bool               `json:"stop_on_error,omitempty"`
	RateLimit   int                `json:"rate_limit,omitempty"`
	DryRun      bool               `json:"dry_run,omitempty"`
}

// SendBulkEmailResponse represents bulk send response
type SendBulkEmailResponse struct {
	TotalEmails  int                       `json:"total_emails"`
	SuccessCount int                       `json:"success_count"`
	FailureCount int                       `json:"failure_count"`
	Results      []BulkEmailResultResponse `json:"results"`
	Duration     string                    `json:"duration"`
}

// BulkEmailResultResponse represents individual result
type BulkEmailResultResponse struct {
	Index     int       `json:"index"`
	EmailID   string    `json:"email_id"`
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ScheduleEmailRequest represents schedule request
type ScheduleEmailRequest struct {
	Email       SendEmailRequest `json:"email" binding:"required"`
	ScheduledAt time.Time        `json:"scheduled_at" binding:"required"`
}

// ScheduleEmailResponse represents schedule response
type ScheduleEmailResponse struct {
	EmailID     string    `json:"email_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Message     string    `json:"message"`
}

// EmailDetailResponse represents email detail
type EmailDetailResponse struct {
	ID                string     `json:"id"`
	To                []string   `json:"to"`
	CC                []string   `json:"cc,omitempty"`
	BCC               []string   `json:"bcc,omitempty"`
	From              string     `json:"from"`
	Subject           string     `json:"subject"`
	Status            string     `json:"status"`
	Priority          string     `json:"priority"`
	ProviderName      string     `json:"provider_name,omitempty"`
	ProviderMessageID string     `json:"provider_message_id,omitempty"`
	RetryCount        int        `json:"retry_count"`
	MaxRetries        int        `json:"max_retries"`
	LastError         string     `json:"last_error,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	ScheduledAt       *time.Time `json:"scheduled_at,omitempty"`
}

// EmailSummaryResponse represents email summary
type EmailSummaryResponse struct {
	ID           string     `json:"id"`
	To           []string   `json:"to"`
	Subject      string     `json:"subject"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	ProviderName string     `json:"provider_name,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
}

// ListEmailsQuery represents list query parameters
type ListEmailsQuery struct {
	Status   string `form:"status"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	SortBy   string `form:"sort_by"`
	SortDesc bool   `form:"sort_desc"`
}

// ListEmailsResponse represents list response
type ListEmailsResponse struct {
	Emails     []EmailSummaryResponse `json:"emails"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// ============= TEMPLATE MODELS =============

// CreateTemplateRequest represents create template request
type CreateTemplateRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Description       string                 `json:"description,omitempty"`
	Type              string                 `json:"type" binding:"required"`
	Subject           string                 `json:"subject" binding:"required"`
	BodyHTML          string                 `json:"body_html,omitempty"`
	BodyText          string                 `json:"body_text,omitempty"`
	Language          string                 `json:"language,omitempty"`
	Category          string                 `json:"category,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	RequiredVariables []string               `json:"required_variables,omitempty"`
	OptionalVariables []string               `json:"optional_variables,omitempty"`
	DefaultVariables  map[string]interface{} `json:"default_variables,omitempty"`
	SampleData        map[string]interface{} `json:"sample_data,omitempty"`
}

// CreateTemplateResponse represents create response
type CreateTemplateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// UpdateTemplateRequest represents update request
type UpdateTemplateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	BodyHTML    *string `json:"body_html,omitempty"`
	BodyText    *string `json:"body_text,omitempty"`
	ForceUpdate bool    `json:"force_update,omitempty"`
}

// UpdateTemplateResponse represents update response
type UpdateTemplateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Message   string    `json:"message"`
	WasActive bool      `json:"was_active"`
}

// DeleteTemplateQuery represents delete query parameters
type DeleteTemplateQuery struct {
	HardDelete bool `form:"hard_delete"`
	Force      bool `form:"force"`
}

// DeleteTemplateResponse represents delete response
type DeleteTemplateResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Deleted    bool   `json:"deleted"`
	Archived   bool   `json:"archived"`
	Message    string `json:"message"`
	EmailCount int    `json:"email_count,omitempty"`
}

// TemplateDetailResponse represents template detail
type TemplateDetailResponse struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Type              string                 `json:"type"`
	Status            string                 `json:"status"`
	Version           int                    `json:"version"`
	Subject           string                 `json:"subject"`
	BodyHTML          string                 `json:"body_html,omitempty"`
	BodyText          string                 `json:"body_text,omitempty"`
	Language          string                 `json:"language"`
	Category          string                 `json:"category,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	RequiredVariables []string               `json:"required_variables,omitempty"`
	OptionalVariables []string               `json:"optional_variables,omitempty"`
	DefaultVariables  map[string]interface{} `json:"default_variables,omitempty"`
	SampleData        map[string]interface{} `json:"sample_data,omitempty"`
	CreatedBy         string                 `json:"created_by"`
	UpdatedBy         string                 `json:"updated_by,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	UsageCount        int                    `json:"usage_count"`
	LastUsedAt        *time.Time             `json:"last_used_at,omitempty"`
}

// TemplateSummaryResponse represents template summary
type TemplateSummaryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	Category    string    `json:"category,omitempty"`
	Language    string    `json:"language"`
	UsageCount  int       `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListTemplatesQuery represents list query parameters
type ListTemplatesQuery struct {
	Status   string `form:"status"`
	Category string `form:"category"`
	Language string `form:"language"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	SortBy   string `form:"sort_by"`
	SortDesc bool   `form:"sort_desc"`
}

// ListTemplatesResponse represents list response
type ListTemplatesResponse struct {
	Templates []TemplateSummaryResponse `json:"templates"`
}

// ActivateTemplateResponse represents activate response
type ActivateTemplateResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	ActivatedAt time.Time `json:"activated_at"`
	Message     string    `json:"message"`
}

// PreviewTemplateRequest represents preview request
type PreviewTemplateRequest struct {
	Data      map[string]interface{} `json:"data,omitempty"`
	UseSample bool                   `json:"use_sample,omitempty"`
}

// PreviewTemplateResponse represents preview response
type PreviewTemplateResponse struct {
	TemplateID   string                 `json:"template_id"`
	TemplateName string                 `json:"template_name"`
	Subject      string                 `json:"subject"`
	BodyHTML     string                 `json:"body_html"`
	BodyText     string                 `json:"body_text"`
	DataUsed     map[string]interface{} `json:"data_used"`
	Warnings     []string               `json:"warnings,omitempty"`
}