package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents complete application configuration
type Config struct {
	Environment string        `mapstructure:"environment"`
	Service     ServiceConfig `mapstructure:"service"`
	HTTP        HTTPConfig    `mapstructure:"http"`
	GRPC        GRPCConfig    `mapstructure:"grpc"`
	Database    DatabaseConfig `mapstructure:"database"`
	SMTP        SMTPConfig    `mapstructure:"smtp"`
	Workers     WorkersConfig `mapstructure:"workers"`
	Logging     LoggingConfig `mapstructure:"logging"`
	Metrics     MetricsConfig `mapstructure:"metrics"`
}

// ServiceConfig contains service metadata
type ServiceConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// HTTPConfig contains HTTP server configuration
type HTTPConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// GRPCConfig contains gRPC server configuration
type GRPCConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// SMTPConfig contains SMTP provider configuration
type SMTPConfig struct {
	Enabled             bool          `mapstructure:"enabled"`
	Host                string        `mapstructure:"host"`
	Port                int           `mapstructure:"port"`
	Username            string        `mapstructure:"username"`
	Password            string        `mapstructure:"password"`
	FromEmail           string        `mapstructure:"from_email"`
	FromName            string        `mapstructure:"from_name"`
	UseTLS              bool          `mapstructure:"use_tls"`
	UseSSL              bool          `mapstructure:"use_ssl"`
	SkipVerify          bool          `mapstructure:"skip_verify"`
	Timeout             time.Duration `mapstructure:"timeout"`
	PoolMinSize         int           `mapstructure:"pool_min_size"`
	PoolMaxSize         int           `mapstructure:"pool_max_size"`
	PoolMaxLifetime     time.Duration `mapstructure:"pool_max_lifetime"`
	PoolMaxIdleTime     time.Duration `mapstructure:"pool_max_idle_time"`
	PoolHealthInterval  time.Duration `mapstructure:"pool_health_interval"`
	BulkWorkers         int           `mapstructure:"bulk_workers"`
	RateLimit           int           `mapstructure:"rate_limit"`
}

// WorkersConfig contains worker configuration
type WorkersConfig struct {
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Retry     RetryConfig     `mapstructure:"retry"`
	Cleanup   CleanupConfig   `mapstructure:"cleanup"`
}

// SchedulerConfig contains scheduler worker configuration
type SchedulerConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batch_size"`
}

// RetryConfig contains retry worker configuration
type RetryConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Interval   time.Duration `mapstructure:"interval"`
	BatchSize  int           `mapstructure:"batch_size"`
	MaxRetries int           `mapstructure:"max_retries"`
	BaseDelay  time.Duration `mapstructure:"base_delay"`
}

// CleanupConfig contains cleanup worker configuration
type CleanupConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Interval      time.Duration `mapstructure:"interval"`
	RetentionDays int           `mapstructure:"retention_days"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// Load loads configuration from file and environment
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables override
	v.SetEnvPrefix("SENDFLIX")
	v.AutomaticEnv()

	// Unmarshal
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Service
	v.SetDefault("service.name", "sendflix")
	v.SetDefault("service.version", "1.0.0")
	v.SetDefault("environment", "development")

	// HTTP
	v.SetDefault("http.enabled", true)
	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", "30s")
	v.SetDefault("http.write_timeout", "30s")
	v.SetDefault("http.idle_timeout", "120s")

	// gRPC
	v.SetDefault("grpc.enabled", false)
	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 50051)

	// Database
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("database.conn_max_idle_time", "10m")

	// SMTP
	v.SetDefault("smtp.enabled", true)
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.use_tls", true)
	v.SetDefault("smtp.timeout", "30s")
	v.SetDefault("smtp.pool_min_size", 5)
	v.SetDefault("smtp.pool_max_size", 20)
	v.SetDefault("smtp.pool_max_lifetime", "30m")
	v.SetDefault("smtp.pool_max_idle_time", "10m")
	v.SetDefault("smtp.pool_health_interval", "2m")
	v.SetDefault("smtp.bulk_workers", 10)
	v.SetDefault("smtp.rate_limit", 10)

	// Workers
	v.SetDefault("workers.scheduler.enabled", true)
	v.SetDefault("workers.scheduler.interval", "1m")
	v.SetDefault("workers.scheduler.batch_size", 100)

	v.SetDefault("workers.retry.enabled", true)
	v.SetDefault("workers.retry.interval", "5m")
	v.SetDefault("workers.retry.batch_size", 50)
	v.SetDefault("workers.retry.max_retries", 3)
	v.SetDefault("workers.retry.base_delay", "1m")

	v.SetDefault("workers.cleanup.enabled", true)
	v.SetDefault("workers.cleanup.interval", "24h")
	v.SetDefault("workers.cleanup.retention_days", 90)

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Metrics
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.port", 9090)
	v.SetDefault("metrics.path", "/metrics")
}

// Validate validates configuration
func (c *Config) Validate() error {
	// Validate environment
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}
	if !validEnvs[c.Environment] {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}

	// Validate database URL
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Validate SMTP if enabled
	if c.SMTP.Enabled {
		if c.SMTP.Host == "" {
			return fmt.Errorf("SMTP host is required")
		}
		if c.SMTP.FromEmail == "" {
			return fmt.Errorf("SMTP from_email is required")
		}
	}

	return nil
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}