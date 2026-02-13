package worker

import (
	"context"
	"sync"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// Worker interface
type Worker interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	IsHealthy() bool
}

// Manager manages background workers
type Manager struct {
	workers []Worker
	logger  logger.Logger
	metrics metrics.MetricsCollector
	wg      sync.WaitGroup
	stopCh  chan struct{}
}

// NewManager creates new worker manager
func NewManager(logger logger.Logger, metrics metrics.MetricsCollector) *Manager {
	return &Manager{
		workers: make([]Worker, 0),
		logger:  logger,
		metrics: metrics,
		stopCh:  make(chan struct{}),
	}
}

// Register registers a worker
func (m *Manager) Register(worker Worker) {
	m.workers = append(m.workers, worker)
	m.logger.Info("worker registered", "name", worker.Name())
}

// Start starts all workers
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("starting workers", "count", len(m.workers))

	for _, worker := range m.workers {
		m.wg.Add(1)
		go func(w Worker) {
			defer m.wg.Done()

			m.logger.Info("starting worker", "name", w.Name())

			if err := w.Start(ctx); err != nil {
				m.logger.Error("worker failed", "name", w.Name(), "error", err)
				m.metrics.IncrementCounter("worker.failed", 1, "worker", w.Name())
			}
		}(worker)
	}

	// Start health check
	go m.healthCheck(ctx)

	return nil
}

// Stop stops all workers gracefully
func (m *Manager) Stop() error {
	m.logger.Info("stopping workers")

	close(m.stopCh)

	for _, worker := range m.workers {
		if err := worker.Stop(); err != nil {
			m.logger.Error("failed to stop worker", "name", worker.Name(), "error", err)
		}
	}

	m.wg.Wait()

	m.logger.Info("all workers stopped")
	return nil
}

// healthCheck monitors worker health
func (m *Manager) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, worker := range m.workers {
				healthy := worker.IsHealthy()
				healthyFloat := 0.0
				if healthy {
					healthyFloat = 1.0
				}

				m.metrics.RecordGauge(
					"worker.healthy",
					healthyFloat,
					"worker", worker.Name(),
				)

				if !healthy {
					m.logger.Warn("worker unhealthy", "name", worker.Name())
				}
			}
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}