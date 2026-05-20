package saveddata

import (
	"context"
	"sort"
	"testing"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestManagerCRUDAndOwnership(t *testing.T) {
	repo := newFakeSavedRepository()
	manager := NewManager(repo)
	userID := uuid.New()
	otherUserID := uuid.New()

	id, err := manager.Create(context.Background(), repositories.SavedDataEntity{UserID: userID, Kind: "favorite", Label: "Tofu", Payload: []byte(`{"itemId":"1"}`)})
	if err != nil {
		t.Fatal(err)
	}

	saved, err := manager.Get(context.Background(), userID, id)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Label != "Tofu" {
		t.Fatalf("unexpected saved data: %#v", saved)
	}

	err = manager.Update(context.Background(), userID, repositories.SavedDataEntity{ID: id, Kind: "favorite", Label: "Updated Tofu", Payload: []byte(`{"itemId":"1"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if repo.items[id].Label != "Updated Tofu" {
		t.Fatalf("expected update, got %#v", repo.items[id])
	}

	if _, err := manager.Get(context.Background(), otherUserID, id); err == nil {
		t.Fatal("expected cross-user get denied")
	}
	if err := manager.Delete(context.Background(), otherUserID, id); err == nil {
		t.Fatal("expected cross-user delete denied")
	}
	if err := manager.Delete(context.Background(), userID, id); err != nil {
		t.Fatal(err)
	}
}

func TestManagerListsByUserKindInDescendingOrder(t *testing.T) {
	repo := newFakeSavedRepository()
	manager := NewManager(repo)
	userID := uuid.New()
	firstID := repo.seed(repositories.SavedDataEntity{UserID: userID, Kind: "favorite", Label: "First", CreatedAt: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)})
	secondID := repo.seed(repositories.SavedDataEntity{UserID: userID, Kind: "favorite", Label: "Second", CreatedAt: time.Date(2026, 5, 19, 11, 0, 0, 0, time.UTC)})
	repo.seed(repositories.SavedDataEntity{UserID: uuid.New(), Kind: "favorite", Label: "Other", CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)})

	items, err := manager.List(context.Background(), userID, "favorite")
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 || items[0].ID != secondID || items[1].ID != firstID {
		t.Fatalf("expected user-scoped descending order, got %#v", items)
	}
}

func TestManagerDeduplicatesSearchHistoryByLabel(t *testing.T) {
	repo := newFakeSavedRepository()
	manager := NewManager(repo)
	userID := uuid.New()
	existingID := repo.seed(repositories.SavedDataEntity{UserID: userID, Kind: "search_history", Label: "tofu"})

	id, err := manager.Create(context.Background(), repositories.SavedDataEntity{UserID: userID, Kind: "search_history", Label: "tofu"})
	if err != nil {
		t.Fatal(err)
	}

	if id != existingID || len(repo.items) != 1 {
		t.Fatalf("expected search history dedupe, got id=%s items=%d", id, len(repo.items))
	}
}

func TestManagerRejectsInvalidKind(t *testing.T) {
	manager := NewManager(newFakeSavedRepository())

	_, err := manager.Create(context.Background(), repositories.SavedDataEntity{UserID: uuid.New(), Kind: "bad", Label: "Bad"})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "validation_error" {
		t.Fatalf("expected validation error, got %v", err)
	}
}

type fakeSavedRepository struct {
	items map[uuid.UUID]repositories.SavedDataEntity
}

func newFakeSavedRepository() *fakeSavedRepository {
	return &fakeSavedRepository{items: make(map[uuid.UUID]repositories.SavedDataEntity)}
}

func (repo *fakeSavedRepository) Create(ctx context.Context, saved repositories.SavedDataEntity) (uuid.UUID, error) {
	return repo.seed(saved), nil
}

func (repo *fakeSavedRepository) seed(saved repositories.SavedDataEntity) uuid.UUID {
	if saved.ID == uuid.Nil {
		saved.ID = uuid.New()
	}
	if saved.CreatedAt.IsZero() {
		saved.CreatedAt = time.Now().UTC()
	}
	repo.items[saved.ID] = saved
	return saved.ID
}

func (repo *fakeSavedRepository) GetByID(ctx context.Context, id uuid.UUID) (repositories.SavedDataEntity, error) {
	return repo.items[id], nil
}

func (repo *fakeSavedRepository) ListByUser(ctx context.Context, userID uuid.UUID, kind string) ([]repositories.SavedDataEntity, error) {
	var items []repositories.SavedDataEntity
	for _, item := range repo.items {
		if item.UserID == userID && (kind == "" || item.Kind == kind) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (repo *fakeSavedRepository) Update(ctx context.Context, saved repositories.SavedDataEntity) error {
	repo.items[saved.ID] = saved
	return nil
}

func (repo *fakeSavedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(repo.items, id)
	return nil
}
