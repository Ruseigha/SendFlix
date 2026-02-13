package worker

import (
	"context"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// CleanupWorker handles old email cleanup
type CleanupWorker struct {
	emailRepo     domain.EmailRepository
	interval      time.Duration
	retentionDays int
	logger        logger.Logger
	metrics       metrics.MetricsCollector
	stopCh        chan struct{}
	healthy       bool
}

// NewCleanupWorker creates new cleanup worker
func NewCleanupWorker(
	emailRepo domain.EmailRepository,
	interval time.Duration,
	retentionDays int,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *CleanupWorker {
	return &CleanupWorker{
		emailRepo:     emailRepo,
		interval:      interval,
		retentionDays: retentionDays,
		logger:        logger,
		metrics:       metrics,
		stopCh:        make(chan struct{}),
		healthy:       true,
	}
}

// Name returns worker name
func (w *CleanupWorker) Name() string {
	return "cleanup"
}

// Start starts the worker
func (w *CleanupWorker) Start(ctx context.Context) error {
	w.logger.Info("cleanup worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.performCleanup(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		}
	}
}

// Stop stops the worker
func (w *CleanupWorker) Stop() error {
	w.logger.Info("stopping cleanup worker")
	close(w.stopCh)
	return nil
}

// IsHealthy returns worker health status
func (w *CleanupWorker) IsHealthy() bool {
	return w.healthy
}

// performCleanup removes old emails
func (w *CleanupWorker) performCleanup(ctx context.Context) {
	startTime := time.Now()
	w.logger.Info("performing cleanup")

	cutoffDate := time.Now().AddDate(0, 0, -w.retentionDays)

	// Clean sent emails older than retention period
	sentCount, err := w.cleanupStatus(ctx, domain.EmailStatusSent, cutoffDate)
	if err != nil {
		w.logger.Error("failed to cleanup sent emails", "error", err)
		w.healthy = false
		return
	}

	// Clean failed emails (30 days)
	failedCutoff := time.Now().AddDate(0, 0, -30)
	failedCount, err := w.cleanupStatus(ctx, domain.EmailStatusFailed, failedCutoff)
	if err != nil {
		w.logger.Error("failed to cleanup failed emails", "error", err)
		w.healthy = false
		return
	}

	// Clean cancelled emails immediately
	cancelledCount, err := w.cleanupStatus(ctx, domain.EmailStatusCancelled, time.Now())
	if err != nil {
		w.logger.Error("failed to cleanup cancelled emails", "error", err)
		w.healthy = false
		return
	}

	w.healthy = true

	duration := time.Since(startTime)
	totalCleaned := sentCount + failedCount + cancelledCount

	w.logger.Info("cleanup completed",
		"sent_cleaned", sentCount,
		"failed_cleaned", failedCount,
		"cancelled_cleaned", cancelledCount,
		"total", totalCleaned,
		"duration", duration)

	w.metrics.IncrementCounter("cleanup.total", int64(totalCleaned))
	w.metrics.RecordDuration("cleanup.duration", duration)
}

// cleanupStatus removes emails with given status older than cutoff
func (w *CleanupWorker) cleanupStatus(
	_ context.Context,
	status domain.EmailStatus,
	cutoff time.Time,
) (int, error) {
	// This would require a new repository method
	// For now, simplified implementation
	w.logger.Debug("cleaning up emails",
		"status", status,
		"cutoff", cutoff)

	// TODO: Implement batch deletion in repository
	return 0, nil
}