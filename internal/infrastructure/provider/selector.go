package provider

import (
	"context"
	"fmt"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/Ruseigha/SendFlix/pkg/logger"
)

// ProviderSelector selects appropriate email provider
type ProviderSelector struct {
	providers map[string]domain.Provider
	default_  string
	logger    logger.Logger
}

// NewProviderSelector creates new selector
func NewProviderSelector(
	providers map[string]domain.Provider,
	defaultProvider string,
	logger logger.Logger,
) domain.ProviderSelector {
	return &ProviderSelector{
		providers: providers,
		default_:  defaultProvider,
		logger:    logger,
	}
}

// SelectProvider selects provider for email
func (s *ProviderSelector) SelectProvider(ctx context.Context, email *domain.Email) (domain.Provider, error) {
	// If email specifies provider, use that
	if email.ProviderName != "" {
		provider, exists := s.providers[email.ProviderName]
		if exists {
			s.logger.Debug("using requested provider", "provider", email.ProviderName)
			return provider, nil
		}
		s.logger.Warn("requested provider not found, using default",
			"requested", email.ProviderName,
			"default", s.default_)
	}

	// Use default provider
	provider, exists := s.providers[s.default_]
	if !exists {
		return nil, fmt.Errorf("default provider '%s' not found", s.default_)
	}

	s.logger.Debug("using default provider", "provider", s.default_)
	return provider, nil
}