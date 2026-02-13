package smtp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// SMTPProvider implements domain.Provider for SMTP
type SMTPProvider struct {
	name       string
	pool       *ConnectionPool
	composer   *MessageComposer
	config     *Config
	logger     logger.Logger
	metrics    metrics.MetricsCollector
	rateLimiter *rateLimiter
}

// Config contains SMTP provider configuration
type Config struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	UseTLS     bool
	UseSSL     bool
	SkipVerify bool
	Timeout    time.Duration

	// Pool configuration
	PoolMinSize         int
	PoolMaxSize         int
	PoolMaxLifetime     time.Duration
	PoolMaxIdleTime     time.Duration
	PoolHealthInterval  time.Duration

	// Bulk sending
	BulkWorkers int
	RateLimit   int // emails per second
}

// NewSMTPProvider creates new SMTP provider
func NewSMTPProvider(
	config *Config,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) (domain.Provider, error) {
	logger.Info("creating SMTP provider", "host", config.Host)

	// Create connection pool
	poolConfig := &PoolConfig{
		MinSize:             config.PoolMinSize,
		MaxSize:             config.PoolMaxSize,
		MaxLifetime:         config.PoolMaxLifetime,
		MaxIdleTime:         config.PoolMaxIdleTime,
		HealthCheckInterval: config.PoolHealthInterval,
		Host:                config.Host,
		Port:                config.Port,
		Username:            config.Username,
		Password:            config.Password,
		UseTLS:              config.UseTLS,
		UseSSL:              config.UseSSL,
		SkipVerify:          config.SkipVerify,
	}

	pool, err := NewConnectionPool(poolConfig, logger, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	provider := &SMTPProvider{
		name:     "smtp",
		pool:     pool,
		composer: NewMessageComposer(),
		config:   config,
		logger:   logger,
		metrics:  metrics,
		rateLimiter: newRateLimiter(config.RateLimit),
	}

	logger.Info("SMTP provider created successfully")
	return provider, nil
}

// Name returns provider name
func (p *SMTPProvider) Name() string {
	return p.name
}

// Send sends single email
func (p *SMTPProvider) Send(ctx context.Context, email *domain.Email) (string, error) {
	startTime := time.Now()
	p.logger.Debug("sending email via SMTP", "to", email.To, "subject", email.Subject)

	// Rate limiting
	p.rateLimiter.Wait()

	// Get connection from pool
	conn, err := p.pool.Get()
	if err != nil {
		p.logger.Error("failed to get connection", "error", err)
		p.metrics.IncrementCounter("smtp.send.connection_failed", 1)
		return "", fmt.Errorf("failed to get SMTP connection: %w", err)
	}
	defer p.pool.Put(conn)

	// Set FROM
	fromAddr := p.config.FromEmail
	if email.From != "" {
		fromAddr = email.From
	}

	if err := conn.client.Mail(fromAddr); err != nil {
		p.logger.Error("MAIL FROM failed", "error", err)
		p.metrics.IncrementCounter("smtp.send.mail_from_failed", 1)
		return "", fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set RCPT TO
	recipients := append(append(email.To, email.CC...), email.BCC...)
	for _, rcpt := range recipients {
		if err := conn.client.Rcpt(rcpt); err != nil {
			p.logger.Error("RCPT TO failed", "recipient", rcpt, "error", err)
			p.metrics.IncrementCounter("smtp.send.rcpt_to_failed", 1)
			return "", fmt.Errorf("RCPT TO failed for %s: %w", rcpt, err)
		}
	}

	// Compose message
	message, err := p.composer.Compose(email)
	if err != nil {
		p.logger.Error("failed to compose message", "error", err)
		p.metrics.IncrementCounter("smtp.send.compose_failed", 1)
		return "", fmt.Errorf("failed to compose message: %w", err)
	}

	// Send DATA
	wc, err := conn.client.Data()
	if err != nil {
		p.logger.Error("DATA command failed", "error", err)
		p.metrics.IncrementCounter("smtp.send.data_failed", 1)
		return "", fmt.Errorf("DATA command failed: %w", err)
	}

	_, err = wc.Write(message)
	if err != nil {
		wc.Close()
		p.logger.Error("failed to write message", "error", err)
		p.metrics.IncrementCounter("smtp.send.write_failed", 1)
		return "", fmt.Errorf("failed to write message: %w", err)
	}

	err = wc.Close()
	if err != nil {
		p.logger.Error("failed to close data writer", "error", err)
		p.metrics.IncrementCounter("smtp.send.close_failed", 1)
		return "", fmt.Errorf("failed to close data writer: %w", err)
	}

	// Generate message ID
	messageID := fmt.Sprintf("smtp_%d_%s", time.Now().Unix(), email.ID)

	duration := time.Since(startTime)
	p.logger.Info("email sent via SMTP",
		"to", email.To,
		"message_id", messageID,
		"duration", duration)

	p.metrics.IncrementCounter("smtp.send.success", 1)
	p.metrics.RecordDuration("smtp.send.duration", duration)

	return messageID, nil
}

// SendBulk sends multiple emails in parallel
func (p *SMTPProvider) SendBulk(ctx context.Context, emails []*domain.Email) ([]domain.BulkResult, error) {
	startTime := time.Now()
	p.logger.Info("sending bulk emails via SMTP", "count", len(emails))

	numWorkers := p.config.BulkWorkers
	if numWorkers <= 0 {
		numWorkers = 10
	}
	if numWorkers > len(emails) {
		numWorkers = len(emails)
	}

	results := make([]domain.BulkResult, len(emails))
	workChan := make(chan int, len(emails))
	var wg sync.WaitGroup

	// Populate work channel
	for i := range emails {
		workChan <- i
	}
	close(workChan)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for index := range workChan {
				email := emails[index]
				result := domain.BulkResult{
					Index:     index,
					EmailID:   email.ID,
					Timestamp: time.Now(),
				}

				messageID, err := p.Send(ctx, email)
				if err != nil {
					result.Success = false
					result.Error = err
				} else {
					result.Success = true
					result.MessageID = messageID
				}

				results[index] = result
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(startTime)
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	p.logger.Info("bulk send completed",
		"total", len(emails),
		"success", successCount,
		"failed", len(emails)-successCount,
		"duration", duration)

	p.metrics.IncrementCounter("smtp.bulk.total", int64(len(emails)))
	p.metrics.IncrementCounter("smtp.bulk.success", int64(successCount))
	p.metrics.RecordDuration("smtp.bulk.duration", duration)

	return results, nil
}

// ValidateConfig validates provider configuration
func (p *SMTPProvider) ValidateConfig(ctx context.Context) error {
	p.logger.Info("validating SMTP configuration")

	// Try to get a connection
	conn, err := p.pool.Get()
	if err != nil {
		return fmt.Errorf("failed to establish SMTP connection: %w", err)
	}
	defer p.pool.Put(conn)

	// Try NOOP
	if err := conn.client.Noop(); err != nil {
		return fmt.Errorf("SMTP server not responding: %w", err)
	}

	p.logger.Info("SMTP configuration valid")
	return nil
}

// GetQuota returns quota information
func (p *SMTPProvider) GetQuota(ctx context.Context) (*domain.Quota, error) {
	// SMTP doesn't have quota API
	return &domain.Quota{
		DailyLimit:     -1, // Unlimited
		DailySent:      0,
		DailyRemaining: -1,
		RateLimit:      p.config.RateLimit,
		RatePeriod:     "second",
	}, nil
}

// SupportsFeature checks if provider supports feature
func (p *SMTPProvider) SupportsFeature(feature domain.ProviderFeature) bool {
	switch feature {
	case domain.FeatureAttachments:
		return true
	case domain.FeatureInlineImages:
		return true
	case domain.FeatureBulkSending:
		return true
	default:
		return false
	}
}

// Close closes provider
func (p *SMTPProvider) Close() error {
	p.logger.Info("closing SMTP provider")
	p.pool.Close()
	return nil
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	rate     int
	tokens   chan struct{}
	stopChan chan struct{}
}

func newRateLimiter(rate int) *rateLimiter {
	if rate <= 0 {
		rate = 10 // Default 10 emails/second
	}

	rl := &rateLimiter{
		rate:     rate,
		tokens:   make(chan struct{}, rate),
		stopChan: make(chan struct{}),
	}

	// Fill bucket
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill loop
	go rl.refillLoop()

	return rl
}

func (rl *rateLimiter) Wait() {
	<-rl.tokens
}

func (rl *rateLimiter) refillLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Refill tokens
			for i := 0; i < rl.rate; i++ {
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Bucket full
				}
			}
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *rateLimiter) Stop() {
	close(rl.stopChan)
}