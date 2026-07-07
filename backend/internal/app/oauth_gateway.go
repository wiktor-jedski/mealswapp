package app

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
)

// GoogleOAuthGateway exchanges Google OAuth callbacks through goth.
// Implements DESIGN-006 OAuthHandler production provider boundary.
type GoogleOAuthGateway struct {
	provider goth.Provider
}

// Implements DESIGN-006 OAuthHandler compile-time provider gateway contract.
var _ httpapi.OAuthProviderGateway = GoogleOAuthGateway{}

// NewGoogleOAuthGateway creates a Google-only provider gateway when configured.
// Implements DESIGN-006 OAuthHandler.
func NewGoogleOAuthGateway(cfg config.OAuthConfig) httpapi.OAuthProviderGateway {
	if strings.TrimSpace(cfg.GoogleClientID) == "" || strings.TrimSpace(cfg.GoogleClientSecret) == "" || strings.TrimSpace(cfg.GoogleCallbackURL) == "" {
		return unavailableOAuthGateway{}
	}
	return GoogleOAuthGateway{provider: google.New(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleCallbackURL, "email", "profile")}
}

// StartOAuth builds the Google authorization redirect URL.
// Implements DESIGN-006 OAuthHandler.
func (g GoogleOAuthGateway) StartOAuth(_ context.Context, provider string, state string) (string, error) {
	if provider != "google" || g.provider == nil {
		return "", errors.New("OAuth provider gateway is not configured")
	}
	session, err := g.provider.BeginAuth(state)
	if err != nil {
		return "", err
	}
	return session.GetAuthURL()
}

// CompleteOAuth exchanges the callback code and maps the Google profile.
// Implements DESIGN-006 OAuthHandler.
func (g GoogleOAuthGateway) CompleteOAuth(_ context.Context, provider string, query map[string]string) (auth.OAuthProfile, error) {
	if provider != "google" || g.provider == nil {
		return auth.OAuthProfile{}, errors.New("OAuth provider gateway is not configured")
	}
	session, err := g.provider.BeginAuth("")
	if err != nil {
		return auth.OAuthProfile{}, err
	}
	params := url.Values{}
	for key, value := range query {
		params.Set(key, value)
	}
	if _, err := session.Authorize(g.provider, params); err != nil {
		return auth.OAuthProfile{}, err
	}
	user, err := g.provider.FetchUser(session)
	if err != nil {
		return auth.OAuthProfile{}, err
	}
	return auth.OAuthProfile{
		Provider:       user.Provider,
		ProviderUserID: user.UserID,
		Email:          user.Email,
		DisplayName:    user.Name,
		EmailVerified:  true,
	}, nil
}
