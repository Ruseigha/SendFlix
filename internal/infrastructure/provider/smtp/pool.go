package smtp

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"sync"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
)

// ConnectionPool manages SMTP connections
//
// WHY CONNECTION POOL:
// - Reuses connections (faster than creating new ones)
// - Reduces server load
// - Handles connection lifecycle
// - Automatic health checks
// - Connection expiry
//
// PERFORMANCE:
// Without pool: ~1 email/second
// With pool: ~10-50 emails/second (10-50x improvement)
type ConnectionPool struct {
	config     *PoolConfig
	pool       chan *pooledConnection
	mu         sync.Mutex
	closed     bool
	logger     logger.Logger
	metrics    metrics.MetricsCollector
	stopHealth chan struct{}
}

// PoolConfig contains pool configuration
type PoolConfig struct {
	MinSize             int
	MaxSize             int
	MaxLifetime         time.Duration
	MaxIdleTime         time.Duration
	HealthCheckInterval time.Duration
	
	// SMTP settings
	Host       string
	Port       int
	Username   string
	Password   string
	UseTLS     bool
	UseSSL     bool
	SkipVerify bool
}

// pooledConnection wraps SMTP client with metadata
type pooledConnection struct {
	client     *smtp.Client
	createdAt  time.Time
	lastUsedAt time.Time
	usageCount int
}

// NewConnectionPool creates new connection pool
//
// INITIALIZATION:
// 1. Creates channel buffer (max pool size)
// 2. Pre-creates minimum connections
// 3. Starts health check goroutine
//
// PARAMETERS:
// - config: Pool configuration
// - logger: Logger instance
// - metrics: Metrics collector
//
// RETURNS:
// - *ConnectionPool: Initialized pool
// - error: If initialization fails
func NewConnectionPool(
	config *PoolConfig,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) (*ConnectionPool, error) {
	logger.Info("creating SMTP connection pool",
		"min_size", config.MinSize,
		"max_size", config.MaxSize)

	pool := &ConnectionPool{
		config:     config,
		pool:       make(chan *pooledConnection, config.MaxSize),
		logger:     logger,
		metrics:    metrics,
		stopHealth: make(chan struct{}),
	}

	// Pre-create minimum connections
	for i := 0; i < config.MinSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			logger.Warn("failed to create initial connection", "error", err)
			continue
		}
		pool.pool <- conn
	}

	// Start health check routine
	go pool.healthCheckLoop()

	logger.Info("SMTP connection pool created successfully")
	return pool, nil
}

// Get retrieves connection from pool
//
// PROCESS:
// 1. Try to get from pool (non-blocking)
// 2. If empty, create new connection
// 3. Validate connection health
// 4. Return healthy connection
//
// RETURNS:
// - *pooledConnection: Ready-to-use connection
// - error: If no connection available
func (p *ConnectionPool) Get() (*pooledConnection, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	p.mu.Unlock()

	// Try to get from pool
	select {
	case conn := <-p.pool:
		// Validate health
		if p.isHealthy(conn) {
			conn.lastUsedAt = time.Now()
			conn.usageCount++
			p.metrics.IncrementCounter("smtp.pool.reuse", 1)
			return conn, nil
		}
		// Unhealthy, close and create new
		conn.client.Close()
		p.metrics.IncrementCounter("smtp.pool.unhealthy", 1)
	default:
		// Pool is empty
	}

	// Create new connection
	conn, err := p.createConnection()
	if err != nil {
		p.logger.Error("failed to create connection", "error", err)
		p.metrics.IncrementCounter("smtp.pool.create.failed", 1)
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	p.metrics.IncrementCounter("smtp.pool.create.success", 1)
	return conn, nil
}

// Put returns connection to pool
//
// PROCESS:
// 1. Check if connection is healthy
// 2. If healthy and pool not full, return to pool
// 3. If unhealthy or pool full, close connection
func (p *ConnectionPool) Put(conn *pooledConnection) {
	if conn == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		conn.client.Close()
		return
	}
	p.mu.Unlock()

	// Check health before returning
	if !p.isHealthy(conn) {
		conn.client.Close()
		p.metrics.IncrementCounter("smtp.pool.discard.unhealthy", 1)
		return
	}

	// Try to return to pool
	select {
	case p.pool <- conn:
		// Successfully returned
		p.metrics.RecordGauge("smtp.pool.size", float64(len(p.pool)))
	default:
		// Pool is full, close connection
		conn.client.Close()
		p.metrics.IncrementCounter("smtp.pool.discard.full", 1)
	}
}

// Close closes pool and all connections
//
// GRACEFUL SHUTDOWN:
// 1. Mark pool as closed
// 2. Stop health check routine
// 3. Drain pool and close all connections
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	p.logger.Info("closing SMTP connection pool")

	// Stop health check
	close(p.stopHealth)

	// Drain and close all connections
	close(p.pool)
	for conn := range p.pool {
		conn.client.Close()
	}

	p.logger.Info("SMTP connection pool closed")
}

// createConnection creates new SMTP connection
//
// CONNECTION PROCESS:
// 1. Dial SMTP server
// 2. StartTLS if required
// 3. Authenticate
// 4. Wrap in pooledConnection
//
// RETURNS:
// - *pooledConnection: New connection
// - error: If connection fails
func (p *ConnectionPool) createConnection() (*pooledConnection, error) {
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	p.logger.Debug("creating SMTP connection", "addr", addr)

	var client *smtp.Client
	var err error

	if p.config.UseSSL {
		// SSL connection (implicit TLS)
		tlsConfig := &tls.Config{
			ServerName:         p.config.Host,
			InsecureSkipVerify: p.config.SkipVerify,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to dial with SSL: %w", err)
		}

		client, err = smtp.NewClient(conn, p.config.Host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Plain connection
		client, err = smtp.Dial(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
		}

		// STARTTLS if required
		if p.config.UseTLS {
			tlsConfig := &tls.Config{
				ServerName:         p.config.Host,
				InsecureSkipVerify: p.config.SkipVerify,
			}

			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return nil, fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	// Authenticate
	if p.config.Username != "" && p.config.Password != "" {
		auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
		if err := client.Auth(auth); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	now := time.Now()
	return &pooledConnection{
		client:     client,
		createdAt:  now,
		lastUsedAt: now,
		usageCount: 0,
	}, nil
}

// isHealthy checks if connection is healthy
//
// HEALTH CHECKS:
// 1. Connection age < MaxLifetime
// 2. Idle time < MaxIdleTime
// 3. NOOP command succeeds
//
// RETURNS:
// - bool: true if healthy
func (p *ConnectionPool) isHealthy(conn *pooledConnection) bool {
	now := time.Now()

	// Check age
	if now.Sub(conn.createdAt) > p.config.MaxLifetime {
		p.logger.Debug("connection expired by age",
			"age", now.Sub(conn.createdAt),
			"max_lifetime", p.config.MaxLifetime)
		return false
	}

	// Check idle time
	if now.Sub(conn.lastUsedAt) > p.config.MaxIdleTime {
		p.logger.Debug("connection expired by idle time",
			"idle_time", now.Sub(conn.lastUsedAt),
			"max_idle_time", p.config.MaxIdleTime)
		return false
	}

	// Ping server with NOOP
	if err := conn.client.Noop(); err != nil {
		p.logger.Debug("connection failed NOOP check", "error", err)
		return false
	}

	return true
}

// healthCheckLoop periodically checks connection health
//
// PROCESS:
// 1. Sleep for HealthCheckInterval
// 2. Check each connection in pool
// 3. Remove unhealthy connections
// 4. Log pool statistics
func (p *ConnectionPool) healthCheckLoop() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		case <-p.stopHealth:
			return
		}
	}
}

// performHealthCheck checks all connections
func (p *ConnectionPool) performHealthCheck() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	poolSize := len(p.pool)
	p.logger.Debug("performing health check", "pool_size", poolSize)

	// Drain pool temporarily
	var connections []*pooledConnection
	for {
		select {
		case conn := <-p.pool:
			connections = append(connections, conn)
		default:
			goto checkDone
		}
	}

checkDone:
	// Check each connection and return healthy ones
	healthyCount := 0
	for _, conn := range connections {
		if p.isHealthy(conn) {
			p.pool <- conn
			healthyCount++
		} else {
			conn.client.Close()
		}
	}

	removedCount := len(connections) - healthyCount

	if removedCount > 0 {
		p.logger.Info("health check completed",
			"checked", len(connections),
			"healthy", healthyCount,
			"removed", removedCount)
		p.metrics.IncrementCounter("smtp.pool.health_check.removed", int64(removedCount))
	}

	// Record pool size
	p.metrics.RecordGauge("smtp.pool.size", float64(len(p.pool)))
}