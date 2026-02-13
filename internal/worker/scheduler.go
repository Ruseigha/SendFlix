package worker

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// SchedulerWorker handles scheduled email sending
type SchedulerWorker struct {
	emailRepo        domain.EmailRepository
	providerSelector domain.ProviderSelector
	interval         time.Duration
	batchSize        int
	logger           logger.Logger
	metrics          metrics.MetricsCollector
	stopCh           chan struct{}
	healthy          bool
}

// NewSchedulerWorker creates new scheduler worker
func NewSchedulerWorker(
	emailRepo domain.EmailRepository,
	providerSelector domain.ProviderSelector,
	interval time.Duration,
	batchSize int,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *SchedulerWorker {
	return &SchedulerWorker{
		emailRepo:        emailRepo,
		providerSelector: providerSelector,
		interval:         interval,
		batchSize:        batchSize,
		logger:           logger,
		metrics:          metrics,
		stopCh:           make(chan struct{}),
		healthy:          true,
	}
}

// Name returns worker name
func (w *SchedulerWorker) Name() string {
	return "scheduler"
}

// Start starts the worker
func (w *SchedulerWorker) Start(ctx context.Context) error {
	w.logger.Info("scheduler worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processScheduledEmails(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		}
	}
}

// Stop stops the worker
func (w *SchedulerWorker) Stop() error {
	w.logger.Info("stopping scheduler worker")
	close(w.stopCh)
	return nil
}

// IsHealthy returns worker health status
func (w *SchedulerWorker) IsHealthy() bool {
	return w.healthy
}

// processScheduledEmails processes scheduled emails
func (w *SchedulerWorker) processScheduledEmails(ctx context.Context) {
	startTime := time.Now()
	w.logger.Debug("processing scheduled emails")

	// Find emails ready to send
	emails, err := w.emailRepo.FindScheduledReady(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("failed to find scheduled emails", "error", err)
		w.healthy = false
		w.metrics.IncrementCounter("scheduler.find.failed", 1)
		return
	}

	w.healthy = true

	if len(emails) == 0 {
		w.logger.Debug("no scheduled emails to process")
		return
	}

	w.logger.Info("processing scheduled emails", "count", len(emails))

	successCount := 0
	failedCount := 0

	for _, email := range emails {
		// Update status to queued
		email.Status = domain.EmailStatusQueued
		if err := w.emailRepo.Update(ctx, email); err != nil {
			w.logger.Error("failed to update email status", "email_id", email.ID, "error", err)
			continue
		}

		// Select provider
		provider, err := w.providerSelector.SelectProvider(ctx, email)
		if err != nil {
			w.logger.Error("provider selection failed", "email_id", email.ID, "error", err)
			email.Status = domain.EmailStatusFailed
			email.LastError = err.Error()
			w.emailRepo.Update(ctx, email)
			failedCount++
			continue
		}

		// Send email
		email.Status = domain.EmailStatusSending
		w.emailRepo.Update(ctx, email)

		messageID, err := provider.Send(ctx, email)
		if err != nil {
			w.logger.Error("send failed", "email_id", email.ID, "error", err)
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
	w.logger.Info("scheduled emails processed",
		"total", len(emails),
		"success", successCount,
		"failed", failedCount,
		"duration", duration)

	w.metrics.IncrementCounter("scheduler.processed", int64(len(emails)))
	w.metrics.IncrementCounter("scheduler.success", int64(successCount))
	w.metrics.IncrementCounter("scheduler.failed", int64(failedCount))
	w.metrics.RecordDuration("scheduler.duration", duration)
}