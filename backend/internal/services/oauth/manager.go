package oauth

import (
	"context"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
	"github.com/markbates/goth"
)

type Provider interface {
	AuthURL(ctx context.Context, provider string, state string) (string, error)
	Complete(ctx context.Context, provider string, state string, code string) (goth.User, error)
}

type UserStore interface {
	FindByOAuth(ctx context.Context, provider string, providerUserID string) (User, bool, error)
	FindByEmail(ctx context.Context, email string) (User, bool, error)
	CreateOAuthUser(ctx context.Context, profile Profile) (User, error)
	LinkOAuthIdentity(ctx context.Context, userID uuid.UUID, profile Profile) error
}

type TrialStarter interface {
	StartTrial(ctx context.Context, userID uuid.UUID) error
}

type Manager struct {
	provider     Provider
	store        UserStore
	trialStarter TrialStarter
}

type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Role        string
}

type Profile struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
}

type Result struct {
	User           User
	LinkedExisting bool
	Created        bool
}

func NewManager(provider Provider, store UserStore, trialStarter TrialStarter) Manager {
	return Manager{provider: provider, store: store, trialStarter: trialStarter}
}

func (manager Manager) Start(ctx context.Context, provider string, state string) (string, error) {
	authURL, err := manager.provider.AuthURL(ctx, provider, state)
	if err != nil {
		return "", providerError(err)
	}
	return authURL, nil
}

func (manager Manager) StartOAuth(ctx context.Context, provider string, state string) (string, error) {
	return manager.Start(ctx, provider, state)
}

func (manager Manager) Complete(ctx context.Context, provider string, state string, code string) (Result, error) {
	gothUser, err := manager.provider.Complete(ctx, provider, state, code)
	if err != nil {
		return Result{}, providerError(err)
	}

	profile := Profile{
		Provider:       provider,
		ProviderUserID: gothUser.UserID,
		Email:          gothUser.Email,
		DisplayName:    gothUser.Name,
	}

	if user, ok, err := manager.store.FindByOAuth(ctx, profile.Provider, profile.ProviderUserID); err != nil {
		return Result{}, err
	} else if ok {
		return Result{User: user}, nil
	}

	if user, ok, err := manager.store.FindByEmail(ctx, profile.Email); err != nil {
		return Result{}, err
	} else if ok {
		if err := manager.store.LinkOAuthIdentity(ctx, user.ID, profile); err != nil {
			return Result{}, err
		}
		return Result{User: user, LinkedExisting: true}, nil
	}

	user, err := manager.store.CreateOAuthUser(ctx, profile)
	if err != nil {
		return Result{}, err
	}
	if manager.trialStarter != nil {
		if err := manager.trialStarter.StartTrial(ctx, user.ID); err != nil {
			return Result{}, err
		}
	}

	return Result{User: user, Created: true}, nil
}

func (manager Manager) CompleteOAuth(ctx context.Context, provider string, state string, code string) (any, error) {
	return manager.Complete(ctx, provider, state, code)
}

func providerError(err error) apperrors.AppError {
	return apperrors.AppError{
		Category:  apperrors.CategoryDependency,
		Code:      "oauth_provider_failed",
		Message:   "OAuth provider failed",
		Retryable: true,
		Status:    503,
		Cause:     err,
	}
}
