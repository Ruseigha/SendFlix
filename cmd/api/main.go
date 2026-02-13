package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ruseigha/SendFlix/internal/delivery/httpapi"
	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/internal/infrastructure/database/postgres"
	"github.com/Ruseigha/SendFlix/internal/infrastructure/provider"
	"github.com/Ruseigha/SendFlix/internal/infrastructure/provider/smtp"
	"github.com/Ruseigha/SendFlix/pkg/config"
	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/Ruseigha/SendFlix/pkg/metrics"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	emailuc "github.com/Ruseigha/SendFlix/internal/usecase/email"
	templateuc "github.com/Ruseigha/SendFlix/internal/usecase/template"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	log.Info("starting SendFlix",
		"version", cfg.Service.Version,
		"environment", cfg.Environment)

	// Initialize metrics
	metricsCollector := metrics.NewPrometheus()

	// Initialize application
	app, err := NewApplication(cfg, log, metricsCollector)
	if err != nil {
		log.Fatal("failed to initialize application", "error", err)
	}

	// Start application
	if err := app.Start(); err != nil {
		log.Fatal("failed to start application", "error", err)
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully...")

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Error("shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

// Application represents the main application
type Application struct {
	config     *config.Config
	logger     logger.Logger
	metrics    metrics.MetricsCollector
	httpServer *http.Server
	db         *sqlx.DB
	providers  map[string]domain.Provider
}

// NewApplication creates new application instance
func NewApplication(
	cfg *config.Config,
	logger logger.Logger,
	metrics metrics.MetricsCollector,
) (*Application, error) {
	logger.Info("initializing application")

	app := &Application{
		config:    cfg,
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]domain.Provider),
	}

	// Connect to database
	if err := app.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// Initialize providers
	if err := app.initProviders(); err != nil {
		return nil, fmt.Errorf("failed to init providers: %w", err)
	}

	// Initialize HTTP server
	if cfg.HTTP.Enabled {
		if err := app.initHTTPServer(); err != nil {
			return nil, fmt.Errorf("failed to init HTTP server: %w", err)
		}
	}

	logger.Info("application initialized successfully")
	return app, nil
}

// initDatabase initializes database connection
func (app *Application) initDatabase() error {
	app.logger.Info("connecting to database")

	dbConfig := postgres.Config{
		URL:             app.config.Database.URL,
		MaxOpenConns:    app.config.Database.MaxOpenConns,
		MaxIdleConns:    app.config.Database.MaxIdleConns,
		ConnMaxLifetime: app.config.Database.ConnMaxLifetime,
		ConnMaxIdleTime: app.config.Database.ConnMaxIdleTime,
	}

	db, err := postgres.Connect(dbConfig, app.logger)
	if err != nil {
		return err
	}

	app.db = db
	return nil
}

// initProviders initializes email providers
func (app *Application) initProviders() error {
	app.logger.Info("initializing email providers")

	// Initialize SMTP provider
	if app.config.SMTP.Enabled {
		smtpConfig := &smtp.Config{
			Host:                app.config.SMTP.Host,
			Port:                app.config.SMTP.Port,
			Username:            app.config.SMTP.Username,
			Password:            app.config.SMTP.Password,
			FromEmail:           app.config.SMTP.FromEmail,
			FromName:            app.config.SMTP.FromName,
			UseTLS:              app.config.SMTP.UseTLS,
			UseSSL:              app.config.SMTP.UseSSL,
			SkipVerify:          app.config.SMTP.SkipVerify,
			Timeout:             app.config.SMTP.Timeout,
			PoolMinSize:         app.config.SMTP.PoolMinSize,
			PoolMaxSize:         app.config.SMTP.PoolMaxSize,
			PoolMaxLifetime:     app.config.SMTP.PoolMaxLifetime,
			PoolMaxIdleTime:     app.config.SMTP.PoolMaxIdleTime,
			PoolHealthInterval:  app.config.SMTP.PoolHealthInterval,
			BulkWorkers:         app.config.SMTP.BulkWorkers,
			RateLimit:           app.config.SMTP.RateLimit,
		}

		smtpProvider, err := smtp.NewSMTPProvider(smtpConfig, app.logger, app.metrics)
		if err != nil {
			return fmt.Errorf("failed to create SMTP provider: %w", err)
		}

		app.providers["smtp"] = smtpProvider
		app.logger.Info("SMTP provider initialized")
	}

	return nil
}

// initHTTPServer initializes HTTP server
func (app *Application) initHTTPServer() error {
	app.logger.Info("initializing HTTP server")

	// Create repositories
	emailRepo := postgres.NewEmailRepository(app.db, app.logger, app.metrics)
	templateRepo := postgres.NewTemplateRepository(app.db, app.logger, app.metrics)

	// Create provider selector
	providerSelector := provider.NewProviderSelector(
		app.providers,
		"smtp",
		app.logger,
	)

	// Create use cases
	sendEmailUC := emailuc.NewSendEmailUseCase(
		emailRepo,
		templateRepo,
		providerSelector,
		app.logger,
		app.metrics,
	)

	sendBulkUC := emailuc.NewSendBulkEmailUseCase(
		sendEmailUC,
		app.logger,
		app.metrics,
	)

	scheduleEmailUC := emailuc.NewScheduleEmailUseCase(
		emailRepo,
		templateRepo,
		app.logger,
		app.metrics,
	)

	getEmailUC := emailuc.NewGetEmailUseCase(
		emailRepo,
		app.logger,
		app.metrics,
	)

	createTemplateUC := templateuc.NewCreateTemplateUseCase(
		templateRepo,
		app.logger,
		app.metrics,
	)

	getTemplateUC := templateuc.NewGetTemplateUseCase(
		templateRepo,
		app.logger,
		app.metrics,
	)

	updateTemplateUC := templateuc.NewUpdateTemplateUseCase(
		templateRepo,
		app.logger,
		app.metrics,
	)

	deleteTemplateUC := templateuc.NewDeleteTemplateUseCase(
		templateRepo,
		emailRepo,
		app.logger,
		app.metrics,
	)

	activateTemplateUC := templateuc.NewActivateTemplateUseCase(
		templateRepo,
		app.logger,
		app.metrics,
	)

	previewTemplateUC := templateuc.NewPreviewTemplateUseCase(
		templateRepo,
		app.logger,
		app.metrics,
	)

	// Create HTTP server
	serverConfig := httpapi.ServerConfig{
		Host:         app.config.HTTP.Host,
		Port:         app.config.HTTP.Port,
		ReadTimeout:  app.config.HTTP.ReadTimeout,
		WriteTimeout: app.config.HTTP.WriteTimeout,
		IdleTimeout:  app.config.HTTP.IdleTimeout,
	}

	useCases := httpapi.UseCases{
		SendEmail:        sendEmailUC,
		SendBulkEmail:    sendBulkUC,
		ScheduleEmail:    scheduleEmailUC,
		GetEmail:         getEmailUC,
		CreateTemplate:   createTemplateUC,
		GetTemplate:      getTemplateUC,
		UpdateTemplate:   updateTemplateUC,
		DeleteTemplate:   deleteTemplateUC,
		ActivateTemplate: activateTemplateUC,
		PreviewTemplate:  previewTemplateUC,
	}

	server := httpapi.NewServer(serverConfig, useCases, app.logger)
	app.httpServer = server.HTTPServer()

	return nil
}

// Start starts the application
func (app *Application) Start() error {
	app.logger.Info("starting application")

	// Start metrics server if enabled
	if app.config.Metrics.Enabled {
		go app.startMetricsServer()
	}

	// Start HTTP server
	if app.config.HTTP.Enabled {
		go func() {
			addr := fmt.Sprintf("%s:%d", app.config.HTTP.Host, app.config.HTTP.Port)
			app.logger.Info("HTTP server starting", "addr", addr)

			if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				app.logger.Fatal("HTTP server failed", "error", err)
			}
		}()
	}

	return nil
}

// startMetricsServer starts Prometheus metrics server
func (app *Application) startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle(app.config.Metrics.Path, promhttp.Handler())

	addr := fmt.Sprintf(":%d", app.config.Metrics.Port)
	app.logger.Info("metrics server starting", "addr", addr, "path", app.config.Metrics.Path)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		app.logger.Error("metrics server failed", "error", err)
	}
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("shutting down application")

	// Shutdown HTTP server
	if app.httpServer != nil {
		if err := app.httpServer.Shutdown(ctx); err != nil {
			app.logger.Error("HTTP server shutdown failed", "error", err)
			return err
		}
	}

	// Close database
	if app.db != nil {
		if err := app.db.Close(); err != nil {
			app.logger.Error("database close failed", "error", err)
			return err
		}
	}

	// Close providers
	for name, p := range app.providers {
		if closer, ok := p.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				app.logger.Error("provider close failed", "provider", name, "error", err)
			}
		}
	}

	return nil
}