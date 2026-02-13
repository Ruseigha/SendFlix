package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Config contains database configuration
type Config struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Connect creates database connection
func Connect(config Config, logger logger.Logger) (*sqlx.DB, error) {
	logger.Info("connecting to database")

	db, err := sqlx.Open("postgres", config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connected successfully")
	return db, nil
}