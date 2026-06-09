package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// PostgresRegistrationRepository creates accounts and consent records transactionally.
// Implements DESIGN-015 ConsentManager.
type PostgresRegistrationRepository struct {
	db transactionalExecutor
}

// NewPostgresRegistrationRepository creates a registration repository.
// Implements DESIGN-015 ConsentManager.
func NewPostgresRegistrationRepository(db transactionalExecutor) *PostgresRegistrationRepository {
	return &PostgresRegistrationRepository{db: db}
}

// CreateUserWithConsent creates an encrypted user and consent record in one transaction.
// Implements DESIGN-015 ConsentManager.
func (r *PostgresRegistrationRepository) CreateUserWithConsent(ctx context.Context, user EncryptedAuthUser, privacyVersion string, termsVersion string) (uuid.UUID, error) {
	if strings.TrimSpace(privacyVersion) == "" || strings.TrimSpace(termsVersion) == "" {
		return uuid.Nil, validationError("consent versions are required")
	}
	var userID uuid.UUID
	err := withTransaction(ctx, r.db, func(tx transactionalExecutor) error {
		id, err := NewPostgresEncryptedIdentityRepository(tx).CreateUser(ctx, user)
		if err != nil {
			return err
		}
		if _, err := NewPostgresComplianceRepository(tx).RecordConsent(ctx, ConsentRecord{UserID: id, PrivacyPolicyVersion: privacyVersion, TermsVersion: termsVersion}); err != nil {
			return err
		}
		userID = id
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// Implements DESIGN-015 ConsentManager compile-time repository contract.
var _ RegistrationRepository = (*PostgresRegistrationRepository)(nil)
