package repository

import (
	"context"
	_ "embed"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-006 AuthController session create query.
//
//go:embed sql/session_create.sql
var sessionCreateSQL string

// Implements DESIGN-006 AuthController session lookup query.
//
//go:embed sql/session_get_by_refresh_hash.sql
var sessionGetByRefreshHashSQL string

// Implements DESIGN-006 AuthController session revoke query.
//
//go:embed sql/session_revoke.sql
var sessionRevokeSQL string

// Implements DESIGN-006 AuthController session-family revoke query.
//
//go:embed sql/session_revoke_family.sql
var sessionRevokeFamilySQL string

// Implements DESIGN-006 AuthController password-reset session-family revocation query.
//
//go:embed sql/session_revoke_user.sql
var sessionRevokeUserSQL string

// PostgresSessionRepository persists refresh-token session metadata.
// Implements DESIGN-006 AuthController.
type PostgresSessionRepository struct {
	db sqlExecutor
}

// NewPostgresSessionRepository creates a PostgreSQL session repository.
// Implements DESIGN-006 AuthController.
func NewPostgresSessionRepository(db sqlExecutor) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

// CreateSession stores refresh-token rotation metadata.
// Implements DESIGN-006 AuthController.
func (r *PostgresSessionRepository) CreateSession(ctx context.Context, session UserSession) (uuid.UUID, error) {
	if session.UserID == uuid.Nil || session.RefreshFamilyID == uuid.Nil || strings.TrimSpace(session.RefreshTokenHash) == "" {
		return uuid.Nil, validationError("session identity is required")
	}
	if session.AccessExpiresAt.IsZero() || session.RefreshExpiresAt.IsZero() || !session.RefreshExpiresAt.After(session.AccessExpiresAt) {
		return uuid.Nil, validationError("session expiry is invalid")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, sessionCreateSQL, session.UserID, session.RefreshTokenHash, session.RefreshFamilyID, session.AccessExpiresAt, session.RefreshExpiresAt).Scan(&id); err != nil {
		return uuid.Nil, mapPostgresError(err, "create session")
	}
	return id, nil
}

// GetSessionByRefreshTokenHash loads one session by stored refresh hash.
// Implements DESIGN-006 AuthController.
func (r *PostgresSessionRepository) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (UserSession, error) {
	if strings.TrimSpace(refreshTokenHash) == "" {
		return UserSession{}, validationError("refresh token hash is required")
	}
	row := r.db.QueryRow(ctx, sessionGetByRefreshHashSQL, refreshTokenHash)
	return scanUserSession(row)
}

// RevokeSession revokes one session idempotently.
// Implements DESIGN-006 AuthController.
func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return validationError("session id is required")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, sessionRevokeSQL, sessionID).Scan(&id); err != nil {
		return mapPostgresError(err, "revoke session")
	}
	return nil
}

// RevokeSessionFamily revokes every session in a refresh family.
// Implements DESIGN-006 AuthController.
func (r *PostgresSessionRepository) RevokeSessionFamily(ctx context.Context, refreshFamilyID uuid.UUID) error {
	if refreshFamilyID == uuid.Nil {
		return validationError("refresh family id is required")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, sessionRevokeFamilySQL, refreshFamilyID).Scan(&id); err != nil {
		return mapPostgresError(err, "revoke session family")
	}
	return nil
}

// RevokeUserSessions revokes every session owned by one user.
// Implements DESIGN-006 AuthController.
func (r *PostgresSessionRepository) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, sessionRevokeUserSQL, userID).Scan(&id); err != nil {
		return mapPostgresError(err, "revoke user sessions")
	}
	return nil
}

// scanUserSession reads refresh-token metadata from PostgreSQL.
// Implements DESIGN-006 AuthController.
func scanUserSession(row pgx.Row) (UserSession, error) {
	var session UserSession
	if err := row.Scan(&session.ID, &session.UserID, &session.RefreshTokenHash, &session.RefreshFamilyID, &session.AccessExpiresAt, &session.RefreshExpiresAt, &session.RevokedAt, &session.CreatedAt); err != nil {
		return UserSession{}, mapPostgresError(err, "scan session")
	}
	return session, nil
}

// Implements DESIGN-006 AuthController compile-time session repository contract.
var _ SessionRepository = (*PostgresSessionRepository)(nil)
