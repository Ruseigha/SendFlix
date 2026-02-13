package worker

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// RetryWorker handles email retry with exponential backoff
type RetryWorker struct {
	emailRepo        domain.EmailRepository
	providerSelector domain.ProviderSelector
	interval         time.Duration
	batchSize        int
	maxRetries       int
	baseDelay        time.Duration
	logger           logger.Logger
	metrics          metrics.MetricsCollector
	stopCh           chan struct{}
	healthy          bool
}

// NewRetryWorker creates new retry worker
func NewRetryWorker(
	emailRepo domain.EmailRepository,
	providerSelector domain.ProviderSelector,
	interval time.Duration,
	batchSize int,
	maxRetries int,
	baseDelay time.Duration,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *RetryWorker {
	return &RetryWorker{
		emailRepo:        emailRepo,
		providerSelector: providerSelector,
		interval:         interval,
		batchSize:        batchSize,
		maxRetries:       maxRetries,
		baseDelay:        baseDelay,
		logger:           logger,
		metrics:          metrics,
		stopCh:           make(chan struct{}),
		healthy:          true,
	}
}

// Name returns worker name
func (w *RetryWorker) Name() string {
	return "retry"
}

// Start starts the worker
func (w *RetryWorker) Start(ctx context.Context) error {
	w.logger.Info("retry worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processRetries(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		}
	}
}

// Stop stops the worker
func (w *RetryWorker) Stop() error {
	w.logger.Info("stopping retry worker")
	close(w.stopCh)
	return nil
}

// IsHealthy returns worker health status
func (w *RetryWorker) IsHealthy() bool {
	return w.healthy
}

// processRetries processes failed emails for retry
func (w *RetryWorker) processRetries(ctx context.Context) {
	startTime := time.Now()
	w.logger.Debug("processing retries")

	// Find failed emails ready for retry
	opts := domain.QueryOptions{
		Page:  1,
		Limit: w.batchSize,
	}

	emails, err := w.emailRepo.FindByStatus(ctx, domain.EmailStatusFailed, opts)
	if err != nil {
		w.logger.Error("failed to find failed emails", "error", err)
		w.healthy = false
		w.metrics.IncrementCounter("retry.find.failed", 1)
		return
	}

	w.healthy = true

	// Filter emails ready for retry
	readyForRetry := make([]*domain.Email, 0)
	for _, email := range emails {
		if w.isReadyForRetry(email) {
			readyForRetry = append(readyForRetry, email)
		}
	}

	if len(readyForRetry) == 0 {
		w.logger.Debug("no emails ready for retry")
		return
	}

	w.logger.Info("processing retries", "count", len(readyForRetry))

	successCount := 0
	failedCount := 0
	deadLetterCount := 0

	for _, email := range readyForRetry {
		// Check max retries
		if email.RetryCount >= w.maxRetries {
			email.Status = domain.EmailStatusDeadLetter
			w.emailRepo.Update(ctx, email)
			deadLetterCount++
			w.logger.Warn("email moved to dead letter queue",
				"email_id", email.ID,
				"retry_count", email.RetryCount)
			continue
		}

		// Select provider
		provider, err := w.providerSelector.SelectProvider(ctx, email)
		if err != nil {
			w.logger.Error("provider selection failed", "email_id", email.ID, "error", err)
			failedCount++
			continue
		}

		// Retry send
		email.Status = domain.EmailStatusSending
		w.emailRepo.Update(ctx, email)

		messageID, err := provider.Send(ctx, email)
		if err != nil {
			w.logger.Error("retry send failed", "email_id", email.ID, "error", err)
			email.MarkAsFailed(err)
			w.emailRepo.Update(ctx, email)
			failedCount++
			continue
		}

		// Mark as sent
		email.MarkAsSent(messageID)
		w.emailRepo.Update(ctx, email)
		successCount++
	}

	duration := time.Since(startTime)
	w.logger.Info("retries processed",
		"total", len(readyForRetry),
		"success", successCount,
		"failed", failedCount,
		"dead_letter", deadLetterCount,
		"duration", duration)

	w.metrics.IncrementCounter("retry.processed", int64(len(readyForRetry)))
	w.metrics.IncrementCounter("retry.success", int64(successCount))
	w.metrics.IncrementCounter("retry.failed", int64(failedCount))
	w.metrics.IncrementCounter("retry.dead_letter", int64(deadLetterCount))
	w.metrics.RecordDuration("retry.duration", duration)
}

// isReadyForRetry checks if email is ready for retry
func (w *RetryWorker) isReadyForRetry(email *domain.Email) bool {
	if !email.CanRetry() {
		return false
	}

	// Calculate next retry time with exponential backoff
	// delay = baseDelay * 2^retryCount
	delay := w.baseDelay * time.Duration(1<<uint(email.RetryCount))

	nextRetry := email.UpdatedAt.Add(delay)
	return time.Now().After(nextRetry)
}