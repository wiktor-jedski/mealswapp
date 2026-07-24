package tagmanager

// Implements DESIGN-009 TagManager service verification.

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

type memoryRepository struct {
	items   map[uuid.UUID]repository.ClassificationEntity
	err     error
	softErr error
}

func newMemoryRepository(items ...repository.ClassificationEntity) *memoryRepository {
	repo := &memoryRepository{items: map[uuid.UUID]repository.ClassificationEntity{}}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (r *memoryRepository) List(_ context.Context, kind repository.ClassificationKind) ([]repository.ClassificationEntity, error) {
	if r.err != nil {
		return nil, r.err
	}
	items := []repository.ClassificationEntity{}
	for _, item := range r.items {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *memoryRepository) GetByID(_ context.Context, id uuid.UUID) (repository.ClassificationEntity, error) {
	if r.err != nil {
		return repository.ClassificationEntity{}, r.err
	}
	item, ok := r.items[id]
	if !ok {
		return repository.ClassificationEntity{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return item, nil
}

func (r *memoryRepository) Create(_ context.Context, item repository.ClassificationEntity) (repository.ClassificationEntity, error) {
	if r.err != nil {
		return repository.ClassificationEntity{}, r.err
	}
	for _, existing := range r.items {
		if existing.Kind == item.Kind && sameParent(existing.ParentID, item.ParentID) && strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(item.Name)) {
			return repository.ClassificationEntity{}, repository.NewError(repository.ErrorKindConflict, "duplicate", nil)
		}
	}
	item.ID = uuid.New()
	r.items[item.ID] = item
	return item, nil
}

func sameParent(left, right *uuid.UUID) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (r *memoryRepository) Update(_ context.Context, item repository.ClassificationEntity) (repository.ClassificationEntity, error) {
	if r.err != nil {
		return repository.ClassificationEntity{}, r.err
	}
	r.items[item.ID] = item
	return item, nil
}

func (r *memoryRepository) SoftDelete(_ context.Context, id uuid.UUID) error {
	if r.softErr != nil {
		return r.softErr
	}
	if r.err != nil {
		return r.err
	}
	delete(r.items, id)
	return nil
}

func TestServiceCreateListUpdateDeleteAndHierarchyValidation(t *testing.T) {
	ctx := context.Background()
	rootID, childID, roleID := uuid.New(), uuid.New(), uuid.New()
	repo := newMemoryRepository(
		repository.ClassificationEntity{ID: rootID, Name: "Fruit", Kind: repository.ClassificationKindFoodCategory},
		repository.ClassificationEntity{ID: childID, Name: "Citrus", Kind: repository.ClassificationKindFoodCategory, ParentID: &rootID},
		repository.ClassificationEntity{ID: roleID, Name: "Snack", Kind: repository.ClassificationKindCulinaryRole},
	)
	service := NewService(repo)

	items, err := service.List(ctx, repository.ClassificationKindFoodCategory)
	if err != nil || len(items) != 2 {
		t.Fatalf("List() = %#v, %v", items, err)
	}
	created, err := service.Create(ctx, repo, repository.ClassificationEntity{Name: "Berries", Kind: repository.ClassificationKindFoodCategory, ParentID: &rootID})
	if err != nil || created.ID == uuid.Nil {
		t.Fatalf("Create() = %#v, %v", created, err)
	}
	if _, err := service.Create(ctx, repo, repository.ClassificationEntity{Name: " berries ", Kind: repository.ClassificationKindFoodCategory, ParentID: &rootID}); !repository.IsKind(err, repository.ErrorKindConflict) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
	if _, err := service.Create(ctx, repo, repository.ClassificationEntity{Name: "Bad", Kind: repository.ClassificationKindFoodCategory, ParentID: &roleID}); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("cross-kind Create() error = %v", err)
	}
	before, after, err := service.Update(ctx, repo, childID, "Sweet citrus", nil)
	if err != nil || before.Name != "Citrus" || after.Name != "Sweet citrus" || after.ParentID != nil {
		t.Fatalf("Update() before=%#v after=%#v err=%v", before, after, err)
	}
	createdID := created.ID
	if _, _, err := service.Update(ctx, repo, rootID, "Fruit", &createdID); !repository.IsKind(err, repository.ErrorKindConflict) {
		t.Fatalf("cycle Update() error = %v", err)
	}
	deleted, err := service.Delete(ctx, repo, created.ID)
	if err != nil || deleted.ID != created.ID {
		t.Fatalf("Delete() = %#v, %v", deleted, err)
	}
}

func TestServicePropagatesRepositoryFailures(t *testing.T) {
	want := repository.NewError(repository.ErrorKindConnection, "down", nil)
	repo := newMemoryRepository()
	repo.err = want
	service := NewService(repo)
	if _, err := service.List(context.Background(), repository.ClassificationKindFoodCategory); err != want {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.Create(context.Background(), repo, repository.ClassificationEntity{Kind: repository.ClassificationKindFoodCategory}); err != want {
		t.Fatalf("Create() error = %v", err)
	}
	parentID := uuid.New()
	if _, err := service.Create(context.Background(), repo, repository.ClassificationEntity{Kind: repository.ClassificationKindFoodCategory, ParentID: &parentID}); err != want {
		t.Fatalf("Create() parent lookup error = %v", err)
	}
	if _, _, err := service.Update(context.Background(), repo, uuid.New(), "x", nil); err != want {
		t.Fatalf("Update() error = %v", err)
	}
	if _, err := service.Delete(context.Background(), repo, uuid.New()); err != want {
		t.Fatalf("Delete() error = %v", err)
	}
	itemID := uuid.New()
	repo = newMemoryRepository(repository.ClassificationEntity{ID: itemID, Kind: repository.ClassificationKindFoodCategory})
	repo.softErr = want
	if _, err := service.Delete(context.Background(), repo, itemID); err != want {
		t.Fatalf("Delete() soft-delete error = %v", err)
	}
}

type emptyAllergens struct{}

func (emptyAllergens) ListActive(context.Context) ([]repository.AllergenVocabularyEntry, error) {
	return []repository.AllergenVocabularyEntry{}, nil
}

func TestCommittedRenameReplacesFilterOptionLabelAfterInvalidation(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	repo := newMemoryRepository(repository.ClassificationEntity{ID: id, Name: "Fruit", Kind: repository.ClassificationKindFoodCategory})
	service := NewService(repo)
	filters := search.NewFilterOptionService(repo, emptyAllergens{})

	before, err := filters.Options(ctx, search.SearchModeSubstitution)
	if err != nil || filterLabel(before.Options, id) != "Fruit" {
		t.Fatalf("initial Options() = %#v, %v", before, err)
	}
	if _, _, err := service.Update(ctx, repo, id, "Produce", nil); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	filters.Invalidate()
	after, err := filters.Options(ctx, search.SearchModeSubstitution)
	if err != nil || filterLabel(after.Options, id) != "Produce" {
		t.Fatalf("renamed Options() = %#v, %v", after, err)
	}
}

func filterLabel(options []search.FilterOption, id uuid.UUID) string {
	for _, option := range options {
		if option.FilterID == id.String() {
			return option.Label
		}
	}
	return ""
}
