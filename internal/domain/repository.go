package domain

import "context"

// EmailRepository defines email data access
type EmailRepository interface {
	Save(ctx context.Context, email *Email) error
	FindByID(ctx context.Context, id string) (*Email, error)
	FindByStatus(ctx context.Context, status EmailStatus, opts QueryOptions) ([]*Email, error)
	FindScheduledReady(ctx context.Context, limit int) ([]*Email, error)
	Update(ctx context.Context, email *Email) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context, filters EmailFilters) (int64, error)
}

// TemplateRepository defines template data access
type TemplateRepository interface {
	Save(ctx context.Context, template *Template) error
	FindByID(ctx context.Context, id string) (*Template, error)
	FindByName(ctx context.Context, name string) (*Template, error)
	FindByStatus(ctx context.Context, status TemplateStatus, opts QueryOptions) ([]*Template, error)
	FindAll(ctx context.Context, filters TemplateFilters, opts QueryOptions) ([]*Template, error)
	Update(ctx context.Context, template *Template) error
	Delete(ctx context.Context, id string) error
	IncrementUsage(ctx context.Context, id string) error
	FindMostUsed(ctx context.Context, limit int) ([]*Template, error)
}

// QueryOptions represents query options
type QueryOptions struct {
	Page     int
	Limit    int
	SortBy   string
	SortDesc bool
}

// EmailFilters represents email filters
type EmailFilters struct {
	Status   EmailStatus
	Priority EmailPriority
	Provider string
	From     string
	To       string
}

// TemplateFilters represents template filters
type TemplateFilters struct {
	Status   TemplateStatus
	Type     TemplateType
	Category string
	Language string
	Tags     []string
	Search   string
}