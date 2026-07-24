package repository

// Implements DESIGN-006 AuthController and DESIGN-008 SearchHistoryRepository error-path verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccountVerificationRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wantErr := errors.New("database failed")
	repo := NewPostgresAccountVerificationRepository(&fakeSQLExecutor{})
	if err := repo.MarkEmailVerified(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("MarkEmailVerified() validation error = %v", err)
	}
	if err := repo.UpdatePassword(ctx, uuid.Nil, "", ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdatePassword() validation error = %v", err)
	}
	if err := repo.CreatePasswordResetToken(ctx, PasswordResetToken{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreatePasswordResetToken() validation error = %v", err)
	}
	if _, err := repo.ConsumePasswordResetToken(ctx, "", time.Time{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ConsumePasswordResetToken() validation error = %v", err)
	}

	repo = NewPostgresAccountVerificationRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if err := repo.MarkEmailVerified(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("MarkEmailVerified() database error = %v", err)
	}
	if err := repo.UpdatePassword(ctx, userID, "hash", "salt"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdatePassword() database error = %v", err)
	}
	if _, err := repo.ConsumePasswordResetToken(ctx, "token", time.Now()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ConsumePasswordResetToken() database error = %v", err)
	}
	repo = NewPostgresAccountVerificationRepository(&fakeSQLExecutor{execErr: wantErr})
	if err := repo.CreatePasswordResetToken(ctx, PasswordResetToken{TokenHash: "token", UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CreatePasswordResetToken() database error = %v", err)
	}
}

func TestRegistrationSessionAndHistoryRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wantErr := errors.New("database failed")

	registration := NewPostgresRegistrationRepository(&fakeSQLExecutor{})
	if _, err := registration.CreateUserWithConsent(ctx, EncryptedAuthUser{}, "", ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateUserWithConsent() validation error = %v", err)
	}
	registration = NewPostgresRegistrationRepository(&fakeSQLExecutor{beginErr: wantErr})
	if _, err := registration.CreateUserWithConsent(ctx, EncryptedAuthUser{}, "privacy", "terms"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CreateUserWithConsent() begin error = %v", err)
	}

	sessions := NewPostgresSessionRepository(&fakeSQLExecutor{})
	if err := sessions.RevokeUserSessions(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RevokeUserSessions() validation error = %v", err)
	}
	sessions = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if err := sessions.RevokeUserSessions(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RevokeUserSessions() database error = %v", err)
	}

	saved := NewPostgresSavedDataRepository(&fakeSQLExecutor{})
	if err := saved.ClearHistory(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ClearHistory() validation error = %v", err)
	}
	saved = NewPostgresSavedDataRepository(&fakeSQLExecutor{execErr: wantErr})
	if err := saved.ClearHistory(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ClearHistory() database error = %v", err)
	}
}

func TestEncryptedIdentityRepositoryRemainingValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wantErr := errors.New("database failed")
	field := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("nonce"), Ciphertext: []byte("ciphertext")}
	digest := LookupDigest{KeyVersion: "lookup-v1", Value: "digest"}
	repo := NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{})

	if err := repo.DeleteUserAccount(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("DeleteUserAccount() validation error = %v", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{execErr: wantErr})
	if err := repo.DeleteUserAccount(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("DeleteUserAccount() database error = %v", err)
	}

	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{})
	if _, err := repo.GetOAuthIdentity(ctx, "google", LookupDigest{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOAuthIdentity() digest error = %v", err)
	}
	if _, err := repo.GetOrCreateEncryptedProfile(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOrCreateEncryptedProfile() validation error = %v", err)
	}
	badField := EncryptedField{KeyVersion: "pii-v1"}
	profiles := []EncryptedUserProfile{
		{UserID: userID, DisplayName: &badField, UnitSystem: UnitSystemMetric, ThemePreference: "system"},
		{UserID: userID, UnitSystem: "bad", ThemePreference: "system"},
		{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "bad"},
	}
	for _, profile := range profiles {
		if _, err := repo.UpdateEncryptedProfile(ctx, profile); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("UpdateEncryptedProfile(%+v) error = %v", profile, err)
		}
	}

	invalidOAuth := []EncryptedOAuthIdentity{
		{UserID: userID},
		{UserID: userID, Provider: "google", ProviderUserID: badField, ProviderUserIDDigest: digest, Email: field},
		{UserID: userID, Provider: "google", ProviderUserID: field, ProviderUserIDDigest: LookupDigest{}, Email: field},
	}
	for _, identity := range invalidOAuth {
		if _, err := repo.UpsertOAuthIdentity(ctx, identity); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("UpsertOAuthIdentity(%+v) error = %v", identity, err)
		}
	}
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: badField, Mode: "food"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddEncryptedHistory() envelope error = %v", err)
	}
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddEncryptedHistory() mode error = %v", err)
	}
	if _, err := repo.ListEncryptedHistory(ctx, uuid.Nil, 1); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListEncryptedHistory() validation error = %v", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{queryErr: wantErr})
	if _, err := repo.ListEncryptedHistory(ctx, userID, 0); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListEncryptedHistory() query error = %v", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: wantErr}})
	if _, err := repo.ListEncryptedHistory(ctx, userID, 101); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListEncryptedHistory() scan error = %v", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{rows: &fakeRows{err: wantErr}})
	if _, err := repo.ListEncryptedHistory(ctx, userID, 1); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListEncryptedHistory() iteration error = %v", err)
	}
}

func TestComplianceRepositoryRemainingErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	requestID := uuid.New()
	receiptID := uuid.New()
	wantErr := errors.New("database failed")

	repo := NewPostgresComplianceRepository(&fakeSQLExecutor{})
	if _, err := repo.ListConsent(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListConsent() validation error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{queryErr: wantErr})
	if _, err := repo.ListConsent(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListConsent() query error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: wantErr}})
	if _, err := repo.ListConsent(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListConsent() scan error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{err: wantErr}})
	if _, err := repo.ListConsent(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListConsent() iteration error = %v", err)
	}

	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{queryErr: wantErr})
	if _, err := repo.ClaimDeletionRequests(ctx, time.Now(), 0); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ClaimDeletionRequests() query error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: wantErr}})
	if _, err := repo.ClaimDeletionRequests(ctx, time.Now(), 1); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ClaimDeletionRequests() scan error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{err: wantErr}})
	if _, err := repo.ClaimDeletionRequests(ctx, time.Now(), 1); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ClaimDeletionRequests() iteration error = %v", err)
	}

	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{})
	leaseExpiresAt := time.Now().Add(time.Minute)
	if repo.WithDeletionLeaseDuration(0) != repo || repo.deletionLeaseDuration != 5*time.Minute {
		t.Fatalf("non-positive deletion lease changed duration: %v", repo.deletionLeaseDuration)
	}
	repo.WithDeletionLeaseDuration(time.Second)
	if repo.deletionLeaseDuration != time.Second {
		t.Fatalf("configured deletion lease = %v", repo.deletionLeaseDuration)
	}
	if err := repo.RecordDeletionFailure(ctx, uuid.Nil, leaseExpiresAt, "transient", "", nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordDeletionFailure() id error = %v", err)
	}
	if err := repo.RecordDeletionFailure(ctx, requestID, time.Time{}, "transient", "", nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordDeletionFailure() lease error = %v", err)
	}
	if err := repo.RecordDeletionFailure(ctx, requestID, leaseExpiresAt, "bad", "", nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordDeletionFailure() category error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if err := repo.RecordDeletionFailure(ctx, requestID, leaseExpiresAt, "transient", "", nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordDeletionFailure() write error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{values: []any{requestID}}, execErr: wantErr})
	if err := repo.RecordDeletionFailure(ctx, requestID, leaseExpiresAt, "unknown", "", nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordDeletionFailure() audit error = %v", err)
	}

	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{})
	if err := repo.CompleteDeletionRequest(ctx, uuid.Nil, leaseExpiresAt, receiptID, time.Now()); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CompleteDeletionRequest() validation error = %v", err)
	}
	if err := repo.CompleteDeletionRequest(ctx, requestID, time.Time{}, receiptID, time.Now()); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CompleteDeletionRequest() lease error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if err := repo.CompleteDeletionRequest(ctx, requestID, leaseExpiresAt, receiptID, time.Now()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CompleteDeletionRequest() write error = %v", err)
	}
	repo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{values: []any{requestID}}, execErr: wantErr})
	if err := repo.CompleteDeletionRequest(ctx, requestID, leaseExpiresAt, receiptID, time.Now()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CompleteDeletionRequest() audit error = %v", err)
	}
}

func TestRepositoryFinalErrorBranches(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wantErr := errors.New("database failed")
	field := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("nonce"), Ciphertext: []byte("ciphertext")}
	digest := LookupDigest{KeyVersion: "lookup-v1", Value: "digest"}

	lockouts := NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if _, err := lockouts.ResetFailedLogins(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ResetFailedLogins() database error = %v", err)
	}

	identities := NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{})
	if _, err := identities.CreateUser(ctx, EncryptedAuthUser{Email: field}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateUser() digest error = %v", err)
	}
	if err := identities.DeleteUserAccount(ctx, userID); err != nil {
		t.Fatalf("DeleteUserAccount() success error = %v", err)
	}
	identities = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if _, err := identities.GetOAuthIdentity(ctx, "google", digest); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetOAuthIdentity() scan error = %v", err)
	}

	registrationTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{{values: []any{userID}}, {err: wantErr}}}}
	registration := NewPostgresRegistrationRepository(&fakeSQLExecutor{tx: registrationTx})
	if _, err := registration.CreateUserWithConsent(ctx, EncryptedAuthUser{Email: field, NormalizedEmailDigest: digest}, "privacy", "terms"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CreateUserWithConsent() consent error = %v", err)
	}

	sessions := NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if err := sessions.RevokeSessionFamily(ctx, uuid.New()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RevokeSessionFamily() database error = %v", err)
	}
}
