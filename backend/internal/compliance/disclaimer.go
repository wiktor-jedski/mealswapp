package compliance

import (
	"context"
	"errors"
	"strings"
)

// DisclaimerProvider loads configured disclaimer content.
// Implements DESIGN-015 DisclaimerRenderer.
type DisclaimerProvider interface {
	GetDisclaimer(context.Context, string) (DisclaimerContent, error)
}

// DisclaimerContent is stable Markdown disclaimer content for account surfaces.
// Implements DESIGN-015 DisclaimerRenderer.
type DisclaimerContent struct {
	Location string
	Version  string
	Markdown string
	Fallback bool
	Alert    string
}

// Implements DESIGN-015 DisclaimerRenderer fallback Markdown content.
const fallbackAccountDisclaimer = "Mealswapp account tools do not replace professional medical or nutritional advice. Review important account actions before continuing."

// Implements DESIGN-015 DisclaimerRenderer fallback Markdown content.
const fallbackLoginDisclaimer = "Mealswapp provides nutrition planning support only and does not replace professional medical or nutritional advice."

// DisclaimerService returns configured or bundled fallback disclaimer content.
// Implements DESIGN-015 DisclaimerRenderer.
type DisclaimerService struct {
	provider DisclaimerProvider
}

// NewDisclaimerService creates disclaimer retrieval behavior.
// Implements DESIGN-015 DisclaimerRenderer.
func NewDisclaimerService(provider DisclaimerProvider) *DisclaimerService {
	return &DisclaimerService{provider: provider}
}

// GetDisclaimer returns content for login or account surfaces.
// Implements DESIGN-015 DisclaimerRenderer.
func (s *DisclaimerService) GetDisclaimer(ctx context.Context, location string) (DisclaimerContent, error) {
	normalized := strings.ToLower(strings.TrimSpace(location))
	if normalized == "" {
		normalized = "login"
	}
	if normalized != "login" && normalized != "account" {
		return DisclaimerContent{}, errors.New("disclaimer location is invalid")
	}
	if s.provider != nil {
		content, err := s.provider.GetDisclaimer(ctx, normalized)
		if err == nil && strings.TrimSpace(content.Markdown) != "" {
			content.Location = normalized
			return content, nil
		}
	}
	return DisclaimerContent{Location: normalized, Version: "fallback-v1", Markdown: fallbackDisclaimerMarkdown(normalized), Fallback: true, Alert: "configured_disclaimer_unavailable"}, nil
}

// fallbackDisclaimerMarkdown returns bundled safe disclaimer content.
// Implements DESIGN-015 DisclaimerRenderer.
func fallbackDisclaimerMarkdown(location string) string {
	if location == "account" {
		return fallbackAccountDisclaimer
	}
	return fallbackLoginDisclaimer
}
