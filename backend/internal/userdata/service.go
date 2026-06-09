package userdata

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Service owns authenticated saved data and search history behavior.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
type Service struct {
	saved      repository.SavedItemRepository
	history    repository.EncryptedSearchHistoryRepository
	clearer    repository.SearchHistoryRepository
	encryption *security.EncryptionService
}

// NewService creates user data behavior.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
func NewService(saved repository.SavedItemRepository, history repository.EncryptedSearchHistoryRepository, clearer repository.SearchHistoryRepository, encryption *security.EncryptionService) *Service {
	return &Service{saved: saved, history: history, clearer: clearer, encryption: encryption}
}

// SearchHistoryEntry is decrypted history data at the service boundary.
// Implements DESIGN-008 SearchHistoryRepository.
type SearchHistoryEntry struct {
	ID          uuid.UUID
	Query       string
	Mode        string
	FiltersHash string
}

// ListSaved returns favorites and saved meals for the authenticated user.
// Implements DESIGN-008 SavedDataRepository.
func (s *Service) ListSaved(ctx context.Context, userID uuid.UUID, kind *repository.SavedItemKind) ([]repository.SavedItem, error) {
	return s.saved.ListItems(ctx, userID, kind)
}

// DeleteSaved removes one saved item for the authenticated user.
// Implements DESIGN-008 SavedDataRepository.
func (s *Service) DeleteSaved(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind repository.SavedItemKind) error {
	if kind == repository.SavedItemKindSavedDiet {
		return errors.New("saved diet writes are deferred until Phase 07")
	}
	return s.saved.RemoveItem(ctx, userID, itemID, kind)
}

// AddHistory stores encrypted search query text.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
func (s *Service) AddHistory(ctx context.Context, userID uuid.UUID, query string, mode string, filtersHash string) (uuid.UUID, error) {
	normalized := strings.TrimSpace(query)
	if normalized == "" || strings.ContainsRune(normalized, '\x00') {
		return uuid.Nil, errors.New("history query is required")
	}
	encrypted, err := s.encryption.EncryptPII(ctx, []byte(normalized))
	if err != nil {
		return uuid.Nil, err
	}
	return s.history.AddEncryptedHistory(ctx, repository.EncryptedSearchHistoryEntry{UserID: userID, Query: repository.EncryptedField{KeyVersion: encrypted.KeyVersion, Nonce: encrypted.Nonce, Ciphertext: encrypted.Ciphertext}, Mode: mode, FiltersHash: filtersHash})
}

// ListHistory returns latest encrypted history entries decrypted at the service boundary.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
func (s *Service) ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistoryEntry, error) {
	entries, err := s.history.ListEncryptedHistory(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]SearchHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		plain, err := s.encryption.DecryptPII(ctx, security.EncryptionEnvelope{KeyVersion: entry.Query.KeyVersion, Nonce: entry.Query.Nonce, Ciphertext: entry.Query.Ciphertext})
		if err != nil {
			return nil, err
		}
		result = append(result, SearchHistoryEntry{ID: entry.ID, Query: string(plain), Mode: entry.Mode, FiltersHash: entry.FiltersHash})
	}
	return result, nil
}

// ClearHistory removes all history for the authenticated user.
// Implements DESIGN-008 SearchHistoryRepository.
func (s *Service) ClearHistory(ctx context.Context, userID uuid.UUID) error {
	return s.clearer.ClearHistory(ctx, userID)
}
