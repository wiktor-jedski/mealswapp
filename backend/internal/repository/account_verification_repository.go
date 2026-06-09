package repository

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Implements DESIGN-006 AuthController verification projection query.
//
//go:embed sql/user_mark_verified.sql
var userMarkVerifiedSQL string

// Implements DESIGN-006 AuthController password-reset password update query.
//
//go:embed sql/user_update_password.sql
var userUpdatePasswordSQL string

// Implements DESIGN-006 AuthController password-reset token create query.
//
//go:embed sql/password_reset_create.sql
var passwordResetCreateSQL string

// Implements DESIGN-006 AuthController password-reset token consume query.
//
//go:embed sql/password_reset_consume.sql
var passwordResetConsumeSQL string

// PostgresAccountVerificationRepository persists verification and reset state.
// Implements DESIGN-006 AuthController.
type PostgresAccountVerificationRepository struct {
	db sqlExecutor
}

// NewPostgresAccountVerificationRepository creates a verification repository.
// Implements DESIGN-006 AuthController.
func NewPostgresAccountVerificationRepository(db sqlExecutor) *PostgresAccountVerificationRepository {
	return &PostgresAccountVerificationRepository{db: db}
}

// MarkEmailVerified updates the verified-login projection.
// Implements DESIGN-006 AuthController.
func (r *PostgresAccountVerificationRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, userMarkVerifiedSQL, userID).Scan(&id); err != nil {
		return mapPostgresError(err, "mark email verified")
	}
	return nil
}

// UpdatePassword persists a newly hashed password after reset.
// Implements DESIGN-006 AuthController.
func (r *PostgresAccountVerificationRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, passwordSalt string) error {
	if userID == uuid.Nil || strings.TrimSpace(passwordHash) == "" || strings.TrimSpace(passwordSalt) == "" {
		return validationError("password reset input is required")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, userUpdatePasswordSQL, userID, passwordHash, passwordSalt).Scan(&id); err != nil {
		return mapPostgresError(err, "update password")
	}
	return nil
}

// CreatePasswordResetToken stores a hashed reset token.
// Implements DESIGN-006 AuthController.
func (r *PostgresAccountVerificationRepository) CreatePasswordResetToken(ctx context.Context, token PasswordResetToken) error {
	if strings.TrimSpace(token.TokenHash) == "" || token.UserID == uuid.Nil || token.ExpiresAt.IsZero() {
		return validationError("password reset token is invalid")
	}
	if _, err := r.db.Exec(ctx, passwordResetCreateSQL, token.TokenHash, token.UserID, token.ExpiresAt); err != nil {
		return mapPostgresError(err, "create password reset token")
	}
	return nil
}

// ConsumePasswordResetToken marks a valid reset token used.
// Implements DESIGN-006 AuthController.
func (r *PostgresAccountVerificationRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (PasswordResetToken, error) {
	if strings.TrimSpace(tokenHash) == "" || usedAt.IsZero() {
		return PasswordResetToken{}, validationError("password reset token is invalid")
	}
	var token PasswordResetToken
	if err := r.db.QueryRow(ctx, passwordResetConsumeSQL, tokenHash, usedAt).Scan(&token.TokenHash, &token.UserID, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt); err != nil {
		return PasswordResetToken{}, mapPostgresError(err, "consume password reset token")
	}
	return token, nil
}

// Implements DESIGN-006 AuthController compile-time repository contracts.
var _ AccountVerificationRepository = (*PostgresAccountVerificationRepository)(nil)

// Implements DESIGN-006 AuthController compile-time reset-token repository contract.
var _ PasswordResetTokenRepository = (*PostgresAccountVerificationRepository)(nil)
