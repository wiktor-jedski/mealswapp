package userdata

// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type keyLoader struct {
	active  string
	entries map[string][]byte
}

func (l keyLoader) ActiveKey(context.Context) (string, []byte, error) {
	return l.active, l.entries[l.active], nil
}

func (l keyLoader) Key(_ context.Context, version string) ([]byte, error) {
	key, ok := l.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}

type memorySavedRepository struct {
	items   []repository.SavedItem
	history []repository.EncryptedSearchHistoryEntry
	cleared uuid.UUID
}

func (r *memorySavedRepository) SaveItem(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memorySavedRepository) RemoveItem(_ context.Context, userID uuid.UUID, itemID uuid.UUID, kind repository.SavedItemKind) error {
	for i, item := range r.items {
		if item.UserID == userID && item.ItemID == itemID && item.Kind == kind {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return repository.NewError(repository.ErrorKindNotFound, "saved item not found", nil)
}

func (r *memorySavedRepository) ListItems(_ context.Context, userID uuid.UUID, kind *repository.SavedItemKind) ([]repository.SavedItem, error) {
	result := []repository.SavedItem{}
	for _, item := range r.items {
		if item.UserID == userID && (kind == nil || item.Kind == *kind) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *memorySavedRepository) AddHistory(_ context.Context, entry repository.SearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memorySavedRepository) ListHistory(context.Context, uuid.UUID, int) ([]repository.SearchHistoryEntry, error) {
	return nil, nil
}

func (r *memorySavedRepository) ClearHistory(_ context.Context, userID uuid.UUID) error {
	r.cleared = userID
	return nil
}

func (r *memorySavedRepository) AddEncryptedHistory(_ context.Context, entry repository.EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	entry.ID = uuid.New()
	r.history = append([]repository.EncryptedSearchHistoryEntry{entry}, r.history...)
	return entry.ID, nil
}

func (r *memorySavedRepository) ListEncryptedHistory(_ context.Context, userID uuid.UUID, limit int) ([]repository.EncryptedSearchHistoryEntry, error) {
	result := []repository.EncryptedSearchHistoryEntry{}
	for _, entry := range r.history {
		if entry.UserID == userID {
			result = append(result, entry)
		}
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

// TestServiceSavedDataAndHistory verifies DESIGN-008 user data service behavior.
func TestServiceSavedDataAndHistory(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	itemID := uuid.New()
	repo := &memorySavedRepository{items: []repository.SavedItem{
		{ID: uuid.New(), UserID: userID, ItemID: itemID, Kind: repository.SavedItemKindFavorite},
		{ID: uuid.New(), UserID: otherUserID, ItemID: uuid.New(), Kind: repository.SavedItemKindFavorite},
	}}
	service := NewService(repo, repo, repo, security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")}}))
	items, err := service.ListSaved(ctx, userID, nil)
	if err != nil || len(items) != 1 || items[0].UserID != userID {
		t.Fatalf("ListSaved() = %#v, %v", items, err)
	}
	if err := service.DeleteSaved(ctx, userID, itemID, repository.SavedItemKindFavorite); err != nil {
		t.Fatalf("DeleteSaved() error = %v", err)
	}
	if err := service.DeleteSaved(ctx, userID, uuid.New(), repository.SavedItemKindSavedDiet); err == nil {
		t.Fatal("DeleteSaved() accepted saved diet write")
	}
	if _, err := service.AddHistory(ctx, userID, " tomato ", "search", "filters"); err != nil {
		t.Fatalf("AddHistory() first error = %v", err)
	}
	if _, err := service.AddHistory(ctx, userID, " tomato ", "search", "filters-2"); err != nil {
		t.Fatalf("AddHistory() duplicate error = %v", err)
	}
	if string(repo.history[0].Query.Ciphertext) == "tomato" {
		t.Fatal("history query was stored as plaintext")
	}
	history, err := service.ListHistory(ctx, userID, 100)
	if err != nil || len(history) != 2 || history[0].Query != "tomato" || history[1].Query != "tomato" {
		t.Fatalf("ListHistory() = %#v, %v", history, err)
	}
	if err := service.ClearHistory(ctx, userID); err != nil || repo.cleared != userID {
		t.Fatalf("ClearHistory() err=%v cleared=%s", err, repo.cleared)
	}
}
