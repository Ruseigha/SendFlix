package domain

import (
	"context"
	"time"
)

// Provider defines email provider interface
type Provider interface {
	// Name returns provider identifier
	Name() string

	// Send sends a single email
	Send(ctx context.Context, email *Email) (messageID string, err error)

	// SendBulk sends multiple emails
	SendBulk(ctx context.Context, emails []*Email) ([]BulkResult, error)

	// ValidateConfig validates provider configuration
	ValidateConfig(ctx context.Context) error

	// GetQuota returns provider quota information
	GetQuota(ctx context.Context) (*Quota, error)

	// SupportsFeature checks if provider supports a feature
	SupportsFeature(feature ProviderFeature) bool
}

// BulkResult represents result of bulk email send
type BulkResult struct {
	Index     int
	EmailID   string
	Success   bool
	MessageID string
	Error     error
	Timestamp time.Time
}

// Quota represents provider quota
type Quota struct {
	DailyLimit     int64
	DailySent      int64
	DailyRemaining int64
	RateLimit      int
	RatePeriod     string
}

// ProviderFeature represents provider features
type ProviderFeature string

const (
	FeatureAttachments   ProviderFeature = "attachments"
	FeatureInlineImages  ProviderFeature = "inline_images"
	FeatureScheduling    ProviderFeature = "scheduling"
	FeatureTemplates     ProviderFeature = "templates"
	FeatureTracking      ProviderFeature = "tracking"
	FeatureBulkSending   ProviderFeature = "bulk_sending"
	FeatureWebhooks      ProviderFeature = "webhooks"
)

// ProviderSelector selects appropriate provider
type ProviderSelector interface {
	SelectProvider(ctx context.Context, email *Email) (Provider, error)
}