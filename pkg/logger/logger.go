package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger interface
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) Logger
	WithContext(ctx context.Context) Logger
}

// Config for logger
type Config struct {
	Level  string
	Format string
	Output string
}

// ZeroLogger implementation
type ZeroLogger struct {
	logger zerolog.Logger
}

// New creates new logger
func New(config Config) Logger {
	level := parseLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	var output io.Writer = os.Stdout
	if config.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(output).With().Timestamp().Logger()

	return &ZeroLogger{logger: logger}
}

func (l *ZeroLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(l.logger.Debug(), msg, keysAndValues...)
}

func (l *ZeroLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log(l.logger.Info(), msg, keysAndValues...)
}

func (l *ZeroLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(l.logger.Warn(), msg, keysAndValues...)
}

func (l *ZeroLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log(l.logger.Error(), msg, keysAndValues...)
}

func (l *ZeroLogger) Fatal(msg string, keysAndValues ...interface{}) {
	l.log(l.logger.Fatal(), msg, keysAndValues...)
}

func (l *ZeroLogger) With(keysAndValues ...interface{}) Logger {
	logger := l.logger.With()
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := keysAndValues[i].(string)
			value := keysAndValues[i+1]
			logger = logger.Interface(key, value)
		}
	}
	return &ZeroLogger{logger: logger.Logger()}
}

func (l *ZeroLogger) WithContext(ctx context.Context) Logger {
	return &ZeroLogger{logger: l.logger.With().Logger()}
}

func (l *ZeroLogger) log(event *zerolog.Event, msg string, keysAndValues ...interface{}) {
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := keysAndValues[i].(string)
			value := keysAndValues[i+1]
			event = event.Interface(key, value)
		}
	}
	event.Msg(msg)
}

func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// NoopLogger does nothing
type NoopLogger struct{}

func NewNoop() Logger { return &NoopLogger{} }

func (l *NoopLogger) Debug(msg string, keysAndValues ...any) {}
func (l *NoopLogger) Info(msg string, keysAndValues ...any)  {}
func (l *NoopLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *NoopLogger) Error(msg string, keysAndValues ...any) {}
func (l *NoopLogger) Fatal(msg string, keysAndValues ...any) {}
func (l *NoopLogger) With(keysAndValues ...any) Logger        { return l }
func (l *NoopLogger) WithContext(ctx context.Context) Logger          { return l }