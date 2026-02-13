package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TemplateRepository implements domain.TemplateRepository
type TemplateRepository struct {
	db      *sqlx.DB
	logger  logger.Logger
	metrics metrics.MetricsCollector
}

// NewTemplateRepository creates new repository
func NewTemplateRepository(
	db *sqlx.DB,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) domain.TemplateRepository {
	return &TemplateRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// templateRow represents database row
type templateRow struct {
	ID                string         `db:"id"`
	Name              string         `db:"name"`
	Description       sql.NullString `db:"description"`
	Type              string         `db:"type"`
	Status            string         `db:"status"`
	Version           int            `db:"version"`
	Subject           string         `db:"subject"`
	BodyHTML          sql.NullString `db:"body_html"`
	BodyText          sql.NullString `db:"body_text"`
	Language          string         `db:"language"`
	Category          sql.NullString `db:"category"`
	Tags              pq.StringArray `db:"tags"`
	RequiredVariables pq.StringArray `db:"required_variables"`
	OptionalVariables pq.StringArray `db:"optional_variables"`
	DefaultVariables  []byte         `db:"default_variables"`
	SampleData        []byte         `db:"sample_data"`
	CreatedBy         string         `db:"created_by"`
	UpdatedBy         sql.NullString `db:"updated_by"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
	UsageCount        int            `db:"usage_count"`
	LastUsedAt        sql.NullTime   `db:"last_used_at"`
}

// Save saves template
func (r *TemplateRepository) Save(ctx context.Context, template *domain.Template) error {
	r.logger.Debug("saving template", "id", template.ID)

	row, err := r.toRow(template)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO templates (
			id, name, description, type, status, version,
			subject, body_html, body_text,
			language, category, tags,
			required_variables, optional_variables,
			default_variables, sample_data,
			created_by, updated_by, created_at, updated_at,
			usage_count, last_used_at
		) VALUES (
			:id, :name, :description, :type, :status, :version,
			:subject, :body_html, :body_text,
			:language, :category, :tags,
			:required_variables, :optional_variables,
			:default_variables, :sample_data,
			:created_by, :updated_by, :created_at, :updated_at,
			:usage_count, :last_used_at
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			status = EXCLUDED.status,
			version = templates.version + 1,
			subject = EXCLUDED.subject,
			body_html = EXCLUDED.body_html,
			body_text = EXCLUDED.body_text,
			language = EXCLUDED.language,
			category = EXCLUDED.category,
			tags = EXCLUDED.tags,
			required_variables = EXCLUDED.required_variables,
			optional_variables = EXCLUDED.optional_variables,
			default_variables = EXCLUDED.default_variables,
			sample_data = EXCLUDED.sample_data,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.logger.Error("failed to save template", "error", err)
		r.metrics.IncrementCounter("template.repository.save.failed", 1)
		return fmt.Errorf("failed to save template: %w", err)
	}

	r.metrics.IncrementCounter("template.repository.save.success", 1)
	return nil
}

// FindByID finds template by ID
func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*domain.Template, error) {
	r.logger.Debug("finding template by ID", "id", id)

	query := `SELECT * FROM templates WHERE id = $1`

	var row templateRow
	err := r.db.GetContext(ctx, &row, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrTemplateNotFound
		}
		r.logger.Error("failed to find template", "error", err)
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	template, err := r.toEntity(&row)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("template.repository.find_by_id.success", 1)
	return template, nil
}

// FindByName finds template by name
func (r *TemplateRepository) FindByName(ctx context.Context, name string) (*domain.Template, error) {
	r.logger.Debug("finding template by name", "name", name)

	query := `SELECT * FROM templates WHERE name = $1`

	var row templateRow
	err := r.db.GetContext(ctx, &row, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrTemplateNotFound
		}
		r.logger.Error("failed to find template by name", "error", err)
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	template, err := r.toEntity(&row)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("template.repository.find_by_name.success", 1)
	return template, nil
}

// FindByStatus finds templates by status
func (r *TemplateRepository) FindByStatus(
	ctx context.Context,
	status domain.TemplateStatus,
	opts domain.QueryOptions,
) ([]*domain.Template, error) {
	r.logger.Debug("finding templates by status", "status", status)

	query := `
		SELECT * FROM templates
		WHERE status = $1
		ORDER BY ` + r.buildOrderBy(opts) + `
		LIMIT $2 OFFSET $3
	`

	offset := (opts.Page - 1) * opts.Limit

	var rows []templateRow
	err := r.db.SelectContext(ctx, &rows, query, status, opts.Limit, offset)
	if err != nil {
		r.logger.Error("failed to find templates by status", "error", err)
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	return r.rowsToEntities(rows)
}

// FindAll finds all templates with filters
func (r *TemplateRepository) FindAll(
	ctx context.Context,
	filters domain.TemplateFilters,
	opts domain.QueryOptions,
) ([]*domain.Template, error) {
	r.logger.Debug("finding all templates")

	whereClause, args := r.buildWhereClause(filters)

	query := `
		SELECT * FROM templates
	` + whereClause + `
		ORDER BY ` + r.buildOrderBy(opts) + `
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
		OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	offset := (opts.Page - 1) * opts.Limit
	args = append(args, opts.Limit, offset)

	var rows []templateRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		r.logger.Error("failed to find templates", "error", err)
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	return r.rowsToEntities(rows)
}

// Update updates template
func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	r.logger.Debug("updating template", "id", template.ID)

	row, err := r.toRow(template)
	if err != nil {
		return err
	}

	query := `
		UPDATE templates SET
			name = :name,
			description = :description,
			type = :type,
			status = :status,
			version = :version + 1,
			subject = :subject,
			body_html = :body_html,
			body_text = :body_text,
			language = :language,
			category = :category,
			tags = :tags,
			required_variables = :required_variables,
			optional_variables = :optional_variables,
			default_variables = :default_variables,
			sample_data = :sample_data,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.logger.Error("failed to update template", "error", err)
		r.metrics.IncrementCounter("template.repository.update.failed", 1)
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrTemplateNotFound
	}

	r.metrics.IncrementCounter("template.repository.update.success", 1)
	return nil
}

// Delete deletes template
func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("deleting template", "id", id)

	query := `DELETE FROM templates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete template", "error", err)
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrTemplateNotFound
	}

	r.metrics.IncrementCounter("template.repository.delete.success", 1)
	return nil
}

// IncrementUsage increments template usage
func (r *TemplateRepository) IncrementUsage(ctx context.Context, id string) error {
	query := `
		UPDATE templates
		SET usage_count = usage_count + 1,
		    last_used_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to increment usage", "error", err)
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	return nil
}

// FindMostUsed finds most used templates
func (r *TemplateRepository) FindMostUsed(ctx context.Context, limit int) ([]*domain.Template, error) {
	query := `
		SELECT * FROM templates
		WHERE status = $1
		ORDER BY usage_count DESC, last_used_at DESC
		LIMIT $2
	`

	var rows []templateRow
	err := r.db.SelectContext(ctx, &rows, query, domain.TemplateStatusActive, limit)
	if err != nil {
		r.logger.Error("failed to find most used templates", "error", err)
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	return r.rowsToEntities(rows)
}

// Helper methods

func (r *TemplateRepository) toRow(template *domain.Template) (*templateRow, error) {
	row := &templateRow{
		ID:                template.ID,
		Name:              template.Name,
		Type:              string(template.Type),
		Status:            string(template.Status),
		Version:           template.Version,
		Subject:           template.Subject,
		Language:          template.Language,
		Tags:              pq.StringArray(template.Tags),
		RequiredVariables: pq.StringArray(template.RequiredVariables),
		OptionalVariables: pq.StringArray(template.OptionalVariables),
		CreatedBy:         template.CreatedBy,
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
		UsageCount:        template.UsageCount,
	}

	if template.Description != "" {
		row.Description = sql.NullString{String: template.Description, Valid: true}
	}
	if template.BodyHTML != "" {
		row.BodyHTML = sql.NullString{String: template.BodyHTML, Valid: true}
	}
	if template.BodyText != "" {
		row.BodyText = sql.NullString{String: template.BodyText, Valid: true}
	}
	if template.Category != "" {
		row.Category = sql.NullString{String: template.Category, Valid: true}
	}
	if template.UpdatedBy != "" {
		row.UpdatedBy = sql.NullString{String: template.UpdatedBy, Valid: true}
	}
	if template.LastUsedAt != nil {
		row.LastUsedAt = sql.NullTime{Time: *template.LastUsedAt, Valid: true}
	}

	if template.DefaultVariables != nil {
		data, err := json.Marshal(template.DefaultVariables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default variables: %w", err)
		}
		row.DefaultVariables = data
	}

	if template.SampleData != nil {
		data, err := json.Marshal(template.SampleData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sample data: %w", err)
		}
		row.SampleData = data
	}

	return row, nil
}

func (r *TemplateRepository) toEntity(row *templateRow) (*domain.Template, error) {
	template := &domain.Template{
		ID:                row.ID,
		Name:              row.Name,
		Type:              domain.TemplateType(row.Type),
		Status:            domain.TemplateStatus(row.Status),
		Version:           row.Version,
		Subject:           row.Subject,
		Language:          row.Language,
		Tags:              []string(row.Tags),
		RequiredVariables: []string(row.RequiredVariables),
		OptionalVariables: []string(row.OptionalVariables),
		CreatedBy:         row.CreatedBy,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		UsageCount:        row.UsageCount,
	}

	if row.Description.Valid {
		template.Description = row.Description.String
	}
	if row.BodyHTML.Valid {
		template.BodyHTML = row.BodyHTML.String
	}
	if row.BodyText.Valid {
		template.BodyText = row.BodyText.String
	}
	if row.Category.Valid {
		template.Category = row.Category.String
	}
	if row.UpdatedBy.Valid {
		template.UpdatedBy = row.UpdatedBy.String
	}
	if row.LastUsedAt.Valid {
		template.LastUsedAt = &row.LastUsedAt.Time
	}

	if len(row.DefaultVariables) > 0 {
		if err := json.Unmarshal(row.DefaultVariables, &template.DefaultVariables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal default variables: %w", err)
		}
	}

	if len(row.SampleData) > 0 {
		if err := json.Unmarshal(row.SampleData, &template.SampleData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sample data: %w", err)
		}
	}

	return template, nil
}

func (r *TemplateRepository) rowsToEntities(rows []templateRow) ([]*domain.Template, error) {
	templates := make([]*domain.Template, len(rows))
	for i, row := range rows {
		template, err := r.toEntity(&row)
		if err != nil {
			return nil, err
		}
		templates[i] = template
	}
	return templates, nil
}

func (r *TemplateRepository) buildWhereClause(filters domain.TemplateFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	paramCount := 1

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramCount))
		args = append(args, filters.Status)
		paramCount++
	}

	if filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", paramCount))
		args = append(args, filters.Type)
		paramCount++
	}

	if filters.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", paramCount))
		args = append(args, filters.Category)
		paramCount++
	}

	if filters.Language != "" {
		conditions = append(conditions, fmt.Sprintf("language = $%d", paramCount))
		args = append(args, filters.Language)
		paramCount++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR description ILIKE $%d)",
			paramCount, paramCount+1,
		))
		searchPattern := "%" + filters.Search + "%"
		args = append(args, searchPattern, searchPattern)
		paramCount += 2
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *TemplateRepository) buildOrderBy(opts domain.QueryOptions) string {
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	direction := "ASC"
	if opts.SortDesc {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", sortBy, direction)
}