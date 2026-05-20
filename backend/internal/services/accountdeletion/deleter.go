package accountdeletion

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	DisableUser(ctx context.Context, userID uuid.UUID) error
	DeleteOwnedData(ctx context.Context, userID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	WriteAudit(ctx context.Context, event AuditEvent) error
}

type SessionRevoker interface {
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
}

type CachePurger interface {
	PurgeUserCache(ctx context.Context, userID uuid.UUID) error
}

type AuditEvent struct {
	ActorID   uuid.UUID
	Action    string
	Target    string
	CreatedAt time.Time
}

type Result struct {
	Status          string
	UserDisabled    bool
	SessionsRevoked bool
	CachePurged     bool
	DataDeleted     bool
}

type Deleter struct {
	store          Store
	sessionRevoker SessionRevoker
	cachePurger    CachePurger
	now            func() time.Time
}

func New(store Store, sessionRevoker SessionRevoker, cachePurger CachePurger) Deleter {
	return Deleter{store: store, sessionRevoker: sessionRevoker, cachePurger: cachePurger, now: time.Now}
}

func (deleter Deleter) DeleteAccount(ctx context.Context, userID uuid.UUID) (Result, error) {
	result := Result{Status: "processing"}

	if err := deleter.store.DisableUser(ctx, userID); err != nil {
		return result, err
	}
	result.UserDisabled = true

	if deleter.sessionRevoker != nil {
		if err := deleter.sessionRevoker.RevokeUserSessions(ctx, userID); err != nil {
			return result, err
		}
		result.SessionsRevoked = true
	}

	if deleter.cachePurger != nil {
		if err := deleter.cachePurger.PurgeUserCache(ctx, userID); err != nil {
			return result, err
		}
		result.CachePurged = true
	}

	if err := deleter.store.DeleteOwnedData(ctx, userID); err != nil {
		return result, err
	}
	result.DataDeleted = true

	if err := deleter.store.WriteAudit(ctx, AuditEvent{
		ActorID:   userID,
		Action:    "account.deleted",
		Target:    "user:" + userID.String(),
		CreatedAt: deleter.now().UTC(),
	}); err != nil {
		return result, err
	}

	if err := deleter.store.DeleteUser(ctx, userID); err != nil {
		return result, err
	}

	result.Status = "completed"
	return result, nil
}
