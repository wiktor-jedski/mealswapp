package oauth

import (
	"context"
	"errors"
	"testing"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
	"github.com/markbates/goth"
)

func TestManagerCompleteCreatesUserAndStartsTrial(t *testing.T) {
	store := newFakeStore()
	trials := &fakeTrialStarter{}
	manager := NewManager(fakeProvider{user: goth.User{UserID: "google-1", Email: "user@example.com", Name: "User"}}, store, trials)

	result, err := manager.Complete(context.Background(), "google", "state", "code")
	if err != nil {
		t.Fatal(err)
	}

	if !result.Created || result.LinkedExisting || result.User.Email != "user@example.com" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if trials.startedUserID != result.User.ID {
		t.Fatalf("expected trial start for created user, got %s", trials.startedUserID)
	}
}

func TestManagerCompleteLinksDuplicateEmail(t *testing.T) {
	store := newFakeStore()
	existing := User{ID: uuid.New(), Email: "user@example.com", DisplayName: "Existing", Role: "user"}
	store.usersByEmail["user@example.com"] = existing
	manager := NewManager(fakeProvider{user: goth.User{UserID: "google-2", Email: "user@example.com", Name: "OAuth User"}}, store, nil)

	result, err := manager.Complete(context.Background(), "google", "state", "code")
	if err != nil {
		t.Fatal(err)
	}

	if !result.LinkedExisting || result.Created || result.User.ID != existing.ID {
		t.Fatalf("expected duplicate email to link existing user, got %#v", result)
	}
	if store.linkedProfile.ProviderUserID != "google-2" || store.linkedUserID != existing.ID {
		t.Fatalf("expected linked oauth identity, got user=%s profile=%#v", store.linkedUserID, store.linkedProfile)
	}
}

func TestManagerCompleteReturnsProviderFailure(t *testing.T) {
	manager := NewManager(fakeProvider{err: errors.New("upstream failed")}, newFakeStore(), nil)

	_, err := manager.Complete(context.Background(), "google", "state", "code")
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != "oauth_provider_failed" || !appErr.Retryable {
		t.Fatalf("unexpected provider error: %#v", appErr)
	}
}

type fakeProvider struct {
	user goth.User
	err  error
}

func (provider fakeProvider) AuthURL(ctx context.Context, providerName string, state string) (string, error) {
	if provider.err != nil {
		return "", provider.err
	}
	return "https://oauth.example/" + providerName + "?state=" + state, nil
}

func (provider fakeProvider) Complete(ctx context.Context, providerName string, state string, code string) (goth.User, error) {
	if provider.err != nil {
		return goth.User{}, provider.err
	}
	return provider.user, nil
}

type fakeStore struct {
	usersByOAuth  map[string]User
	usersByEmail  map[string]User
	linkedUserID  uuid.UUID
	linkedProfile Profile
}

func newFakeStore() *fakeStore {
	return &fakeStore{usersByOAuth: make(map[string]User), usersByEmail: make(map[string]User)}
}

func (store *fakeStore) FindByOAuth(ctx context.Context, provider string, providerUserID string) (User, bool, error) {
	user, ok := store.usersByOAuth[provider+":"+providerUserID]
	return user, ok, nil
}

func (store *fakeStore) FindByEmail(ctx context.Context, email string) (User, bool, error) {
	user, ok := store.usersByEmail[email]
	return user, ok, nil
}

func (store *fakeStore) CreateOAuthUser(ctx context.Context, profile Profile) (User, error) {
	user := User{ID: uuid.New(), Email: profile.Email, DisplayName: profile.DisplayName, Role: "user"}
	store.usersByEmail[user.Email] = user
	store.usersByOAuth[profile.Provider+":"+profile.ProviderUserID] = user
	return user, nil
}

func (store *fakeStore) LinkOAuthIdentity(ctx context.Context, userID uuid.UUID, profile Profile) error {
	store.linkedUserID = userID
	store.linkedProfile = profile
	store.usersByOAuth[profile.Provider+":"+profile.ProviderUserID] = store.usersByEmail[profile.Email]
	return nil
}

type fakeTrialStarter struct {
	startedUserID uuid.UUID
}

func (starter *fakeTrialStarter) StartTrial(ctx context.Context, userID uuid.UUID) error {
	starter.startedUserID = userID
	return nil
}
