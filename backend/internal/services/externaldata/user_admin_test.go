package externaldata

import (
	"context"
	"testing"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestUserAdminPanelListsUsersAndShowsEntitlements(t *testing.T) {
	userID := uuid.New()
	store := &fakeUserAdminStore{users: []repositories.UserEntity{{ID: userID, Email: "user@example.com", DisplayName: "User"}}}
	entitlements := &fakeUserEntitlementStore{entitlements: map[uuid.UUID]repositories.EntitlementEntity{
		userID: {UserID: userID, Plan: "paid", Status: "active"},
	}}
	panel := NewUserAdminPanel(store, entitlements, nil, nil)

	list, err := panel.List(context.Background(), "user", 2, 5)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if list.Total != 1 || list.Page != 2 || store.lastQuery.Text != "user" || store.lastQuery.Offset != 5 {
		t.Fatalf("unexpected list/query: result=%#v query=%#v", list, store.lastQuery)
	}

	detail, err := panel.Detail(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected detail error: %v", err)
	}
	if detail.User.ID != userID || detail.Entitlement == nil || detail.Entitlement.Plan != "paid" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestUserAdminPanelDisablesAccountAndResetLockout(t *testing.T) {
	userID := uuid.New()
	store := &fakeUserAdminStore{users: []repositories.UserEntity{{ID: userID, Email: "USER@example.com", DisplayName: "User"}}}
	lockouts := &fakeLockoutResetter{}
	panel := NewUserAdminPanel(store, nil, nil, lockouts)

	disabled, err := panel.Disable(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected disable error: %v", err)
	}
	if !disabled.Disabled || !store.updated.Disabled {
		t.Fatalf("expected user disabled, got disabled=%#v updated=%#v", disabled, store.updated)
	}
	if err := panel.ResetLockout(context.Background(), userID); err != nil {
		t.Fatalf("unexpected reset error: %v", err)
	}
	if lockouts.accountKey != "user@example.com" {
		t.Fatalf("expected normalized account key, got %q", lockouts.accountKey)
	}
}

func TestUserAdminPanelAuditHistory(t *testing.T) {
	userID := uuid.New()
	audits := &fakeUserAuditStore{entries: []repositories.AuditLogEntity{{ID: uuid.New(), Target: "user:" + userID.String(), Action: "admin.disable_user"}}}
	panel := NewUserAdminPanel(&fakeUserAdminStore{users: []repositories.UserEntity{{ID: userID}}}, nil, audits, nil)

	history, err := panel.AuditHistory(context.Background(), userID, 1, 10)
	if err != nil {
		t.Fatalf("unexpected audit error: %v", err)
	}
	if history.Total != 1 || audits.target != "user:"+userID.String() {
		t.Fatalf("unexpected audit history: %#v target=%s", history, audits.target)
	}
}

type fakeUserAdminStore struct {
	users     []repositories.UserEntity
	lastQuery repositories.PageQuery
	updated   repositories.UserEntity
}

func (store *fakeUserAdminStore) GetByID(ctx context.Context, id uuid.UUID) (repositories.UserEntity, error) {
	for _, user := range store.users {
		if user.ID == id {
			return user, nil
		}
	}
	return repositories.UserEntity{}, ErrUserAdminInvalidUser
}

func (store *fakeUserAdminStore) List(ctx context.Context, query repositories.PageQuery) ([]repositories.UserEntity, int, error) {
	store.lastQuery = query
	return store.users, len(store.users), nil
}

func (store *fakeUserAdminStore) Update(ctx context.Context, user repositories.UserEntity) error {
	store.updated = user
	for index := range store.users {
		if store.users[index].ID == user.ID {
			store.users[index] = user
			return nil
		}
	}
	return ErrUserAdminInvalidUser
}

type fakeUserEntitlementStore struct {
	entitlements map[uuid.UUID]repositories.EntitlementEntity
}

func (store *fakeUserEntitlementStore) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error) {
	return store.entitlements[userID], nil
}

type fakeUserAuditStore struct {
	target  string
	entries []repositories.AuditLogEntity
}

func (store *fakeUserAuditStore) ListByTarget(ctx context.Context, target string, query repositories.PageQuery) ([]repositories.AuditLogEntity, int, error) {
	store.target = target
	return store.entries, len(store.entries), nil
}

type fakeLockoutResetter struct {
	accountKey string
}

func (resetter *fakeLockoutResetter) ResetAccount(accountKey string) {
	resetter.accountKey = accountKey
}
