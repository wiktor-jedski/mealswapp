// Package tagmanager owns global Food Category and Culinary Role administration.
package tagmanager

import (
	"context"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Service validates global classification hierarchy changes before persistence.
// Implements DESIGN-009 TagManager.
type Service struct {
	reader repository.ClassificationAdminRepository
}

// NewService creates a global classification manager.
// Implements DESIGN-009 TagManager.
func NewService(reader repository.ClassificationAdminRepository) *Service {
	return &Service{reader: reader}
}

// List returns one deterministic active classification hierarchy.
// Implements DESIGN-009 TagManager.
func (s *Service) List(ctx context.Context, kind repository.ClassificationKind) ([]repository.ClassificationEntity, error) {
	return s.reader.List(ctx, kind)
}

// Create validates the optional parent and inserts a unique classification.
// Implements DESIGN-009 TagManager.
func (s *Service) Create(ctx context.Context, repo repository.ClassificationAdminRepository, classification repository.ClassificationEntity) (repository.ClassificationEntity, error) {
	if err := validateParent(ctx, repo, classification.Kind, uuid.Nil, classification.ParentID); err != nil {
		return repository.ClassificationEntity{}, err
	}
	return repo.Create(ctx, classification)
}

// Update validates a rename or reparent operation and returns before/after snapshots.
// Implements DESIGN-009 TagManager.
func (s *Service) Update(ctx context.Context, repo repository.ClassificationAdminRepository, id uuid.UUID, name string, parentID *uuid.UUID) (repository.ClassificationEntity, repository.ClassificationEntity, error) {
	before, err := repo.GetByID(ctx, id)
	if err != nil {
		return repository.ClassificationEntity{}, repository.ClassificationEntity{}, err
	}
	if err := validateParent(ctx, repo, before.Kind, id, parentID); err != nil {
		return repository.ClassificationEntity{}, repository.ClassificationEntity{}, err
	}
	after, err := repo.Update(ctx, repository.ClassificationEntity{ID: id, Name: name, Kind: before.Kind, ParentID: parentID})
	return before, after, err
}

// Delete soft-deletes one unused classification and returns its audit snapshot.
// Implements DESIGN-009 TagManager.
func (s *Service) Delete(ctx context.Context, repo repository.ClassificationAdminRepository, id uuid.UUID) (repository.ClassificationEntity, error) {
	before, err := repo.GetByID(ctx, id)
	if err != nil {
		return repository.ClassificationEntity{}, err
	}
	if err := repo.SoftDelete(ctx, id); err != nil {
		return repository.ClassificationEntity{}, err
	}
	return before, nil
}

// validateParent rejects cross-kind parents and walks ancestors to reject hierarchy cycles.
// Implements DESIGN-009 TagManager.
func validateParent(ctx context.Context, repo repository.ClassificationAdminRepository, kind repository.ClassificationKind, id uuid.UUID, parentID *uuid.UUID) error {
	seen := map[uuid.UUID]struct{}{id: {}}
	for parentID != nil {
		if _, cycle := seen[*parentID]; cycle {
			return repository.NewError(repository.ErrorKindConflict, "classification hierarchy cycle", nil)
		}
		seen[*parentID] = struct{}{}
		parent, err := repo.GetByID(ctx, *parentID)
		if err != nil {
			return err
		}
		if parent.Kind != kind {
			return repository.NewError(repository.ErrorKindValidation, "classification parent kind mismatch", nil)
		}
		parentID = parent.ParentID
	}
	return nil
}
