package accountdeletion

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDeleterCoordinatesAccountDeletion(t *testing.T) {
	userID := uuid.New()
	store := &fakeDeletionStore{}
	sessions := &fakeSessionRevoker{}
	cache := &fakeCachePurger{}
	deleter := New(store, sessions, cache)

	result, err := deleter.DeleteAccount(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != "completed" || !result.UserDisabled || !result.SessionsRevoked || !result.CachePurged || !result.DataDeleted {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !store.disabled[userID] || !store.deletedUsers[userID] || !store.deletedData[userID] || !sessions.revoked[userID] || !cache.purged[userID] {
		t.Fatalf("expected all deletion collaborators called")
	}
	if len(store.audit) != 1 || store.audit[0].Action != "account.deleted" {
		t.Fatalf("expected audit event, got %#v", store.audit)
	}
}

func TestDeletedUserCannotLogin(t *testing.T) {
	userID := uuid.New()
	store := &fakeDeletionStore{}
	deleter := New(store, nil, nil)

	if _, err := deleter.DeleteAccount(context.Background(), userID); err != nil {
		t.Fatal(err)
	}

	if store.CanLogin(userID) {
		t.Fatal("expected deleted user to be unable to login")
	}
}

func TestOwnedDataRemovedAndAuditPreserved(t *testing.T) {
	userID := uuid.New()
	store := &fakeDeletionStore{ownedData: map[uuid.UUID]int{userID: 3}}
	deleter := New(store, nil, nil)

	if _, err := deleter.DeleteAccount(context.Background(), userID); err != nil {
		t.Fatal(err)
	}

	if store.ownedData[userID] != 0 {
		t.Fatalf("expected owned data removed, got %d", store.ownedData[userID])
	}
	if len(store.audit) != 1 {
		t.Fatalf("expected audit preserved, got %#v", store.audit)
	}
}

type fakeDeletionStore struct {
	disabled     map[uuid.UUID]bool
	deletedUsers map[uuid.UUID]bool
	deletedData  map[uuid.UUID]bool
	ownedData    map[uuid.UUID]int
	audit        []AuditEvent
}

func (store *fakeDeletionStore) ensure() {
	if store.disabled == nil {
		store.disabled = make(map[uuid.UUID]bool)
	}
	if store.deletedUsers == nil {
		store.deletedUsers = make(map[uuid.UUID]bool)
	}
	if store.deletedData == nil {
		store.deletedData = make(map[uuid.UUID]bool)
	}
	if store.ownedData == nil {
		store.ownedData = make(map[uuid.UUID]int)
	}
}

func (store *fakeDeletionStore) DisableUser(ctx context.Context, userID uuid.UUID) error {
	store.ensure()
	store.disabled[userID] = true
	return nil
}

func (store *fakeDeletionStore) DeleteOwnedData(ctx context.Context, userID uuid.UUID) error {
	store.ensure()
	store.deletedData[userID] = true
	store.ownedData[userID] = 0
	return nil
}

func (store *fakeDeletionStore) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	store.ensure()
	store.deletedUsers[userID] = true
	return nil
}

func (store *fakeDeletionStore) WriteAudit(ctx context.Context, event AuditEvent) error {
	store.audit = append(store.audit, event)
	return nil
}

func (store *fakeDeletionStore) CanLogin(userID uuid.UUID) bool {
	store.ensure()
	return !store.disabled[userID] && !store.deletedUsers[userID]
}

type fakeSessionRevoker struct {
	revoked map[uuid.UUID]bool
}

func (revoker *fakeSessionRevoker) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	if revoker.revoked == nil {
		revoker.revoked = make(map[uuid.UUID]bool)
	}
	revoker.revoked[userID] = true
	return nil
}

type fakeCachePurger struct {
	purged map[uuid.UUID]bool
}

func (purger *fakeCachePurger) PurgeUserCache(ctx context.Context, userID uuid.UUID) error {
	if purger.purged == nil {
		purger.purged = make(map[uuid.UUID]bool)
	}
	purger.purged[userID] = true
	return nil
}
