// Package adminbootstrap owns operator-only first-administrator creation.
// Implements DESIGN-009 AdminController operator-only bootstrap.
package adminbootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Request contains explicit environment confirmation and one account selector.
// Implements DESIGN-009 AdminController operator-only bootstrap.
type Request struct {
	ConfiguredEnvironment string
	TargetEnvironment     string
	ConfirmProduction     bool
	Email                 string
	UserID                *uuid.UUID
	RequestID             string
}

// Service validates the operator boundary before entering persistence.
// Implements DESIGN-009 AdminController operator-only bootstrap.
type Service struct {
	repository repository.AdministratorBootstrapRepository
	digests    *security.LookupDigestService
}

// NewService creates an administrator bootstrap service.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func NewService(repo repository.AdministratorBootstrapRepository, digests *security.LookupDigestService) *Service {
	return &Service{repository: repo, digests: digests}
}

// Bootstrap validates environment and selector semantics, then performs the atomic bootstrap.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func (s *Service) Bootstrap(ctx context.Context, request Request) (repository.AdministratorBootstrapResult, error) {
	configured := strings.TrimSpace(request.ConfiguredEnvironment)
	target := strings.TrimSpace(request.TargetEnvironment)
	if configured == "" || target == "" || configured != target {
		return repository.AdministratorBootstrapResult{}, errors.New("bootstrap environment does not match configured environment")
	}
	if target == "production" && !request.ConfirmProduction {
		return repository.AdministratorBootstrapResult{}, errors.New("production bootstrap requires explicit confirmation")
	}
	if (strings.TrimSpace(request.Email) == "") == (request.UserID == nil) {
		return repository.AdministratorBootstrapResult{}, errors.New("exactly one email or user id selector is required")
	}
	selector := repository.AdministratorBootstrapSelector{UserID: request.UserID}
	if request.UserID == nil {
		normalized, err := security.NormalizeInput(security.InputFieldEmail, request.Email)
		if err != nil {
			return repository.AdministratorBootstrapResult{}, errors.New("bootstrap email is invalid")
		}
		digest, err := s.digests.DigestForWrite(ctx, []byte(normalized.Value))
		if err != nil {
			return repository.AdministratorBootstrapResult{}, errors.New("bootstrap email lookup is unavailable")
		}
		selector.EmailDigest = &repository.LookupDigest{KeyVersion: digest.KeyVersion, Value: digest.Value}
		legacyValue := strings.TrimSpace(request.Email)
		if legacyValue != normalized.Value {
			legacy, err := s.digests.DigestForWrite(ctx, []byte(legacyValue))
			if err != nil {
				return repository.AdministratorBootstrapResult{}, errors.New("bootstrap email lookup is unavailable")
			}
			selector.LegacyEmailDigest = &repository.LookupDigest{KeyVersion: legacy.KeyVersion, Value: legacy.Value}
		}
	}
	return s.repository.BootstrapAdministrator(ctx, selector, request.RequestID)
}
