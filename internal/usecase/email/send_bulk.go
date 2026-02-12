package email

import (
	"context"
	"sync"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// SendBulkEmailUseCase handles bulk email sending
type SendBulkEmailUseCase struct {
	sendEmailUC *SendEmailUseCase
	logger      logger.Logger
	metrics     metrics.MetricsCollector
}

// NewSendBulkEmailUseCase creates new use case
func NewSendBulkEmailUseCase(
	sendEmailUC *SendEmailUseCase,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) *SendBulkEmailUseCase {
	return &SendBulkEmailUseCase{
		sendEmailUC: sendEmailUC,
		logger:      logger,
		metrics:     metrics,
	}
}

// SendBulkEmailRequest represents bulk send request
type SendBulkEmailRequest struct {
	Emails      []SendEmailRequest
	BatchSize   int
	StopOnError bool
	RateLimit   int // emails per second
	DryRun      bool
}

// SendBulkEmailResponse represents bulk send response
type SendBulkEmailResponse struct {
	TotalEmails  int
	SuccessCount int
	FailureCount int
	Results      []BulkEmailResult
	Duration     time.Duration
}

// BulkEmailResult represents individual result
type BulkEmailResult struct {
	Index     int
	EmailID   string
	Success   bool
	MessageID string
	Error     error
	Timestamp time.Time
}

// Execute sends bulk emails
func (uc *SendBulkEmailUseCase) Execute(ctx context.Context, req SendBulkEmailRequest) (*SendBulkEmailResponse, error) {
	startTime := time.Now()
	uc.logger.Info("sending bulk emails", "count", len(req.Emails))

	// Set defaults
	if req.BatchSize == 0 {
		req.BatchSize = 100
	}

	results := make([]BulkEmailResult, len(req.Emails))
	successCount := 0
	failureCount := 0

	// Worker pool pattern
	numWorkers := 10
	if numWorkers > len(req.Emails) {
		numWorkers = len(req.Emails)
	}

	workChan := make(chan workItem, len(req.Emails))
	resultChan := make(chan BulkEmailResult, len(req.Emails))

	// Populate work channel
	for i, emailReq := range req.Emails {
		workChan <- workItem{
			index: i,
			req:   emailReq,
		}
	}
	close(workChan)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go uc.worker(ctx, i, workChan, resultChan, &wg, req.DryRun)
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		results[result.Index] = result
		if result.Success {
			successCount++
		} else {
			failureCount++
		}

		if req.StopOnError && !result.Success {
			break
		}
	}

	duration := time.Since(startTime)
	uc.logger.Info("bulk send completed",
		"total", len(req.Emails),
		"success", successCount,
		"failed", failureCount,
		"duration", duration)

	uc.metrics.IncrementCounter("email.bulk.total", int64(len(req.Emails)))
	uc.metrics.IncrementCounter("email.bulk.success", int64(successCount))
	uc.metrics.IncrementCounter("email.bulk.failed", int64(failureCount))
	uc.metrics.RecordDuration("email.bulk.duration", duration)

	return &SendBulkEmailResponse{
		TotalEmails:  len(req.Emails),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      results,
		Duration:     duration,
	}, nil
}

type workItem struct {
	index int
	req   SendEmailRequest
}

// worker processes bulk emails
func (uc *SendBulkEmailUseCase) worker(
	ctx context.Context,
	workerID int,
	workChan <-chan workItem,
	resultChan chan<- BulkEmailResult,
	wg *sync.WaitGroup,
	dryRun bool,
) {
	defer wg.Done()

	for work := range workChan {
		result := BulkEmailResult{
			Index:     work.index,
			Timestamp: time.Now(),
		}

		if dryRun {
			result.Success = true
			result.EmailID = "dry_run"
		} else {
			resp, err := uc.sendEmailUC.Execute(ctx, work.req)
			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.Success = true
				result.EmailID = resp.EmailID
				result.MessageID = resp.ProviderMessageID
			}
		}

		resultChan <- result
	}
}