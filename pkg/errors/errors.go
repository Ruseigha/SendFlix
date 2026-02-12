package errors

import (
	"fmt"
	"net/http"
)

// AppError represents application error
type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
	Context    map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates new error
func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Context:    make(map[string]interface{}),
	}
}

// Wrap wraps error
func Wrap(err error, code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
		Context:    make(map[string]interface{}),
	}
}

func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// Common errors
var (
	ErrValidationFailed    = New("VALIDATION_FAILED", "Validation failed", http.StatusBadRequest)
	ErrNotFound            = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrEmailNotFound       = New("EMAIL_NOT_FOUND", "Email not found", http.StatusNotFound)
	ErrTemplateNotFound    = New("TEMPLATE_NOT_FOUND", "Template not found", http.StatusNotFound)
	ErrAlreadyExists       = New("ALREADY_EXISTS", "Resource already exists", http.StatusConflict)
	ErrProviderUnavailable = New("PROVIDER_UNAVAILABLE", "Provider unavailable", http.StatusServiceUnavailable)
	ErrInternal            = New("INTERNAL_ERROR", "Internal error", http.StatusInternalServerError)
)