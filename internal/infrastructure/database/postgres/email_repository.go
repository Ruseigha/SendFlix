package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// EmailRepository implements domain.EmailRepository
type EmailRepository struct {
	db      *sqlx.DB
	logger  logger.Logger
	metrics metrics.MetricsCollector
}

// NewEmailRepository creates new repository
func NewEmailRepository(
	db *sqlx.DB,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) domain.EmailRepository {
	return &EmailRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// emailRow represents database row
type emailRow struct {
	ID                string         `db:"id"`
	To                pq.StringArray `db:"to_addresses"`
	CC                pq.StringArray `db:"cc_addresses"`
	BCC               pq.StringArray `db:"bcc_addresses"`
	From              string         `db:"from_address"`
	ReplyTo           sql.NullString `db:"reply_to"`
	Subject           string         `db:"subject"`
	BodyHTML          sql.NullString `db:"body_html"`
	BodyText          sql.NullString `db:"body_text"`
	TemplateID        sql.NullString `db:"template_id"`
	TemplateData      []byte         `db:"template_data"`
	Priority          string         `db:"priority"`
	Status            string         `db:"status"`
	ProviderName      sql.NullString `db:"provider_name"`
	ProviderMessageID sql.NullString `db:"provider_message_id"`
	ScheduledAt       sql.NullTime   `db:"scheduled_at"`
	RetryCount        int            `db:"retry_count"`
	MaxRetries        int            `db:"max_retries"`
	LastError         sql.NullString `db:"last_error"`
	TrackOpens        bool           `db:"track_opens"`
	TrackClicks       bool           `db:"track_clicks"`
	Metadata          []byte         `db:"metadata"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
	SentAt            sql.NullTime   `db:"sent_at"`
}

// Save saves email
func (r *EmailRepository) Save(ctx context.Context, email *domain.Email) error {
	r.logger.Debug("saving email", "id", email.ID)

	row, err := r.toRow(email)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO emails (
			id, to_addresses, cc_addresses, bcc_addresses,
			from_address, reply_to, subject, body_html, body_text,
			template_id, template_data, priority, status,
			provider_name, provider_message_id, scheduled_at,
			retry_count, max_retries, last_error,
			track_opens, track_clicks, metadata,
			created_at, updated_at, sent_at
		) VALUES (
			:id, :to_addresses, :cc_addresses, :bcc_addresses,
			:from_address, :reply_to, :subject, :body_html, :body_text,
			:template_id, :template_data, :priority, :status,
			:provider_name, :provider_message_id, :scheduled_at,
			:retry_count, :max_retries, :last_error,
			:track_opens, :track_clicks, :metadata,
			:created_at, :updated_at, :sent_at
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			provider_name = EXCLUDED.provider_name,
			provider_message_id = EXCLUDED.provider_message_id,
			retry_count = EXCLUDED.retry_count,
			last_error = EXCLUDED.last_error,
			updated_at = EXCLUDED.updated_at,
			sent_at = EXCLUDED.sent_at
	`

	_, err = r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.logger.Error("failed to save email", "error", err)
		r.metrics.IncrementCounter("email.repository.save.failed", 1)
		return fmt.Errorf("failed to save email: %w", err)
	}

	r.metrics.IncrementCounter("email.repository.save.success", 1)
	return nil
}

// FindByID finds email by ID
func (r *EmailRepository) FindByID(ctx context.Context, id string) (*domain.Email, error) {
	r.logger.Debug("finding email by ID", "id", id)

	query := `SELECT * FROM emails WHERE id = $1`

	var row emailRow
	err := r.db.GetContext(ctx, &row, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrEmailNotFound
		}
		r.logger.Error("failed to find email", "error", err)
		return nil, fmt.Errorf("failed to find email: %w", err)
	}

	email, err := r.toEntity(&row)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("email.repository.find_by_id.success", 1)
	return email, nil
}

// FindByStatus finds emails by status
func (r *EmailRepository) FindByStatus(
	ctx context.Context,
	status domain.EmailStatus,
	opts domain.QueryOptions,
) ([]*domain.Email, error) {
	r.logger.Debug("finding emails by status", "status", status)

	query := `
		SELECT * FROM emails
		WHERE status = $1
		ORDER BY ` + r.buildOrderBy(opts) + `
		LIMIT $2 OFFSET $3
	`

	offset := (opts.Page - 1) * opts.Limit

	var rows []emailRow
	err := r.db.SelectContext(ctx, &rows, query, status, opts.Limit, offset)
	if err != nil {
		r.logger.Error("failed to find emails by status", "error", err)
		return nil, fmt.Errorf("failed to find emails: %w", err)
	}

	emails := make([]*domain.Email, len(rows))
	for i, row := range rows {
		email, err := r.toEntity(&row)
		if err != nil {
			return nil, err
		}
		emails[i] = email
	}

	r.metrics.IncrementCounter("email.repository.find_by_status.success", 1)
	return emails, nil
}

// FindScheduledReady finds emails ready to send
func (r *EmailRepository) FindScheduledReady(ctx context.Context, limit int) ([]*domain.Email, error) {
	r.logger.Debug("finding scheduled emails ready to send")

	query := `
		SELECT * FROM emails
		WHERE status = $1
		AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT $2
	`

	var rows []emailRow
	err := r.db.SelectContext(ctx, &rows, query, domain.EmailStatusScheduled, limit)
	if err != nil {
		r.logger.Error("failed to find scheduled emails", "error", err)
		return nil, fmt.Errorf("failed to find scheduled emails: %w", err)
	}

	emails := make([]*domain.Email, len(rows))
	for i, row := range rows {
		email, err := r.toEntity(&row)
		if err != nil {
			return nil, err
		}
		emails[i] = email
	}

	r.metrics.IncrementCounter("email.repository.find_scheduled_ready.success", 1)
	return emails, nil
}

// Update updates email
func (r *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	r.logger.Debug("updating email", "id", email.ID)

	row, err := r.toRow(email)
	if err != nil {
		return err
	}

	query := `
		UPDATE emails SET
			status = :status,
			provider_name = :provider_name,
			provider_message_id = :provider_message_id,
			retry_count = :retry_count,
			last_error = :last_error,
			updated_at = :updated_at,
			sent_at = :sent_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.logger.Error("failed to update email", "error", err)
		r.metrics.IncrementCounter("email.repository.update.failed", 1)
		return fmt.Errorf("failed to update email: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrEmailNotFound
	}

	r.metrics.IncrementCounter("email.repository.update.success", 1)
	return nil
}

// Delete deletes email
func (r *EmailRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("deleting email", "id", id)

	query := `DELETE FROM emails WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete email", "error", err)
		return fmt.Errorf("failed to delete email: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrEmailNotFound
	}

	r.metrics.IncrementCounter("email.repository.delete.success", 1)
	return nil
}

// Count counts emails
func (r *EmailRepository) Count(ctx context.Context, filters domain.EmailFilters) (int64, error) {
	query := `SELECT COUNT(*) FROM emails WHERE 1=1`
	args := []interface{}{}
	paramCount := 1

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", paramCount)
		args = append(args, filters.Status)
		paramCount++
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("failed to count emails", "error", err)
		return 0, fmt.Errorf("failed to count emails: %w", err)
	}

	return count, nil
}

// Helper methods

func (r *EmailRepository) toRow(email *domain.Email) (*emailRow, error) {
	row := &emailRow{
		ID:          email.ID,
		To:          pq.StringArray(email.To),
		CC:          pq.StringArray(email.CC),
		BCC:         pq.StringArray(email.BCC),
		From:        email.From,
		Subject:     email.Subject,
		Priority:    string(email.Priority),
		Status:      string(email.Status),
		RetryCount:  email.RetryCount,
		MaxRetries:  email.MaxRetries,
		TrackOpens:  email.TrackOpens,
		TrackClicks: email.TrackClicks,
		CreatedAt:   email.CreatedAt,
		UpdatedAt:   email.UpdatedAt,
	}

	if email.ReplyTo != "" {
		row.ReplyTo = sql.NullString{String: email.ReplyTo, Valid: true}
	}
	if email.BodyHTML != "" {
		row.BodyHTML = sql.NullString{String: email.BodyHTML, Valid: true}
	}
	if email.BodyText != "" {
		row.BodyText = sql.NullString{String: email.BodyText, Valid: true}
	}
	if email.TemplateID != "" {
		row.TemplateID = sql.NullString{String: email.TemplateID, Valid: true}
	}
	if email.ProviderName != "" {
		row.ProviderName = sql.NullString{String: email.ProviderName, Valid: true}
	}
	if email.ProviderMessageID != "" {
		row.ProviderMessageID = sql.NullString{String: email.ProviderMessageID, Valid: true}
	}
	if email.LastError != "" {
		row.LastError = sql.NullString{String: email.LastError, Valid: true}
	}
	if email.ScheduledAt != nil {
		row.ScheduledAt = sql.NullTime{Time: *email.ScheduledAt, Valid: true}
	}
	if email.SentAt != nil {
		row.SentAt = sql.NullTime{Time: *email.SentAt, Valid: true}
	}

	if email.TemplateData != nil {
		data, err := json.Marshal(email.TemplateData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal template data: %w", err)
		}
		row.TemplateData = data
	}

	if email.Metadata != nil {
		data, err := json.Marshal(email.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		row.Metadata = data
	}

	return row, nil
}

func (r *EmailRepository) toEntity(row *emailRow) (*domain.Email, error) {
	email := &domain.Email{
		ID:          row.ID,
		To:          []string(row.To),
		CC:          []string(row.CC),
		BCC:         []string(row.BCC),
		From:        row.From,
		Subject:     row.Subject,
		Priority:    domain.EmailPriority(row.Priority),
		Status:      domain.EmailStatus(row.Status),
		RetryCount:  row.RetryCount,
		MaxRetries:  row.MaxRetries,
		TrackOpens:  row.TrackOpens,
		TrackClicks: row.TrackClicks,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}

	if row.ReplyTo.Valid {
		email.ReplyTo = row.ReplyTo.String
	}
	if row.BodyHTML.Valid {
		email.BodyHTML = row.BodyHTML.String
	}
	if row.BodyText.Valid {
		email.BodyText = row.BodyText.String
	}
	if row.TemplateID.Valid {
		email.TemplateID = row.TemplateID.String
	}
	if row.ProviderName.Valid {
		email.ProviderName = row.ProviderName.String
	}
	if row.ProviderMessageID.Valid {
		email.ProviderMessageID = row.ProviderMessageID.String
	}
	if row.LastError.Valid {
		email.LastError = row.LastError.String
	}
	if row.ScheduledAt.Valid {
		email.ScheduledAt = &row.ScheduledAt.Time
	}
	if row.SentAt.Valid {
		email.SentAt = &row.SentAt.Time
	}

	if len(row.TemplateData) > 0 {
		if err := json.Unmarshal(row.TemplateData, &email.TemplateData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal template data: %w", err)
		}
	}

	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &email.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return email, nil
}

func (r *EmailRepository) buildOrderBy(opts domain.QueryOptions) string {
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