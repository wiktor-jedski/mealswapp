package search

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type filterOptionClassificationStub struct {
	mu      sync.Mutex
	entries map[repository.ClassificationKind][]repository.ClassificationEntity
	err     map[repository.ClassificationKind]error
	calls   int
}

func (s *filterOptionClassificationStub) List(_ context.Context, kind repository.ClassificationKind) ([]repository.ClassificationEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return append([]repository.ClassificationEntity(nil), s.entries[kind]...), s.err[kind]
}

type filterOptionAllergenStub struct {
	mu      sync.Mutex
	entries []repository.AllergenVocabularyEntry
	err     error
	calls   int
}

type filterOptionGenerationStub struct {
	mu         sync.Mutex
	generation uint64
}

func (s *filterOptionGenerationStub) Current(context.Context) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation, nil
}

func (s *filterOptionGenerationStub) Advance() {
	s.mu.Lock()
	s.generation++
	s.mu.Unlock()
}

func (s *filterOptionAllergenStub) ListActive(context.Context) ([]repository.AllergenVocabularyEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return append([]repository.AllergenVocabularyEntry(nil), s.entries...), s.err
}

// Implements DESIGN-009 TagManager deterministic filter policy verification.
func TestFilterOptionServiceProjectsDeterministicLocalizedPolicy(t *testing.T) {
	categoryA := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	categoryZ := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roles := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	classifications := &filterOptionClassificationStub{entries: map[repository.ClassificationKind][]repository.ClassificationEntity{
		repository.ClassificationKindFoodCategory: {
			{ID: categoryZ, Name: "Zest", Kind: repository.ClassificationKindFoodCategory},
			{ID: categoryA, Name: "apple", Kind: repository.ClassificationKindFoodCategory},
		},
		repository.ClassificationKindCulinaryRole: {{ID: roles, Name: "Binder", Kind: repository.ClassificationKindCulinaryRole}},
	}}
	allergens := &filterOptionAllergenStub{entries: []repository.AllergenVocabularyEntry{
		{Key: "tree_nut", Name: "Tree nuts", LabelKey: "filter.allergen.tree_nut"},
		{Key: "dairy", Name: "Dairy", LabelKey: "filter.allergen.dairy"},
	}}
	service := NewFilterOptionService(classifications, allergens)

	response, err := service.Options(context.Background(), SearchModeSubstitution)
	if err != nil {
		t.Fatalf("Options() error = %v", err)
	}
	wantPrefix := []struct {
		id   string
		kind SearchFilterKind
	}{
		{"liquid", SearchFilterKindPhysicalState},
		{"solid", SearchFilterKindPhysicalState},
		{categoryA.String(), SearchFilterKindFoodCategory},
		{categoryZ.String(), SearchFilterKindFoodCategory},
		{roles.String(), SearchFilterKindCulinaryRole},
		{"dairy", SearchFilterKindAllergen},
		{"tree_nut", SearchFilterKindAllergen},
	}
	if len(response.Options) != len(wantPrefix)+5 {
		t.Fatalf("Options() count = %d, want %d", len(response.Options), len(wantPrefix)+5)
	}
	for i, want := range wantPrefix {
		if response.Options[i].FilterID != want.id || response.Options[i].Kind != want.kind {
			t.Fatalf("Options()[%d] = %#v, want id=%q kind=%q", i, response.Options[i], want.id, want.kind)
		}
	}
	for _, option := range response.Options {
		if option.Label == "" {
			t.Fatalf("option lacks fallback label: %#v", option)
		}
		if option.Kind != SearchFilterKindFoodCategory && option.Kind != SearchFilterKindCulinaryRole && option.LabelKey == "" {
			t.Fatalf("backend policy option lacks localization key: %#v", option)
		}
	}
	vegan := findFilterOption(t, response.Options, SearchFilterKindDietaryPreset, string(DietaryPresetVegan))
	wantExcludes := []FilterOptionReference{{FilterID: "animal_product", Kind: SearchFilterKindAllergen}, {FilterID: "dairy", Kind: SearchFilterKindAllergen}, {FilterID: "egg", Kind: SearchFilterKindAllergen}}
	if vegan.IncludeAllowed || !vegan.ExcludeAllowed || !reflect.DeepEqual(vegan.Excludes, wantExcludes) {
		t.Fatalf("vegan policy = %#v, want excludes %#v", vegan, wantExcludes)
	}
	dairy := findFilterOption(t, response.Options, SearchFilterKindAllergen, "dairy")
	if dairy.IncludeAllowed || !dairy.ExcludeAllowed {
		t.Fatalf("allergen policy = %#v", dairy)
	}
}

// Implements DESIGN-009 TagManager empty vocabulary and cache invalidation verification.
func TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration(t *testing.T) {
	classifications := &filterOptionClassificationStub{entries: map[repository.ClassificationKind][]repository.ClassificationEntity{}}
	allergens := &filterOptionAllergenStub{}
	service := NewFilterOptionService(classifications, allergens)

	first, err := service.Options(context.Background(), SearchModeSubstitution)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Options) != 7 {
		t.Fatalf("empty vocabulary option count = %d, want physical states plus dietary presets", len(first.Options))
	}
	first.Options[0].FilterID = "caller-mutation"
	second, err := service.Options(context.Background(), SearchModeSubstitution)
	if err != nil || second.Options[0].FilterID == "caller-mutation" || classifications.calls != 2 || allergens.calls != 1 {
		t.Fatalf("cached copy = %#v err=%v classificationCalls=%d allergenCalls=%d", second, err, classifications.calls, allergens.calls)
	}

	newID := uuid.New()
	classifications.mu.Lock()
	classifications.entries[repository.ClassificationKindFoodCategory] = []repository.ClassificationEntity{{ID: newID, Name: "New admin label", Kind: repository.ClassificationKindFoodCategory}}
	classifications.mu.Unlock()
	service.Invalidate()
	third, err := service.Options(context.Background(), SearchModeSubstitution)
	if err != nil {
		t.Fatal(err)
	}
	if findFilterOption(t, third.Options, SearchFilterKindFoodCategory, newID.String()).Label != "New admin label" || classifications.calls != 4 || allergens.calls != 2 {
		t.Fatalf("invalidated options = %#v classificationCalls=%d allergenCalls=%d", third, classifications.calls, allergens.calls)
	}
}

// Implements DESIGN-009 TagManager cross-instance cache invalidation verification.
func TestFilterOptionServiceInvalidationReachesPeerInstance(t *testing.T) {
	id := uuid.New()
	createdID := uuid.New()
	classifications := &filterOptionClassificationStub{entries: map[repository.ClassificationKind][]repository.ClassificationEntity{
		repository.ClassificationKindFoodCategory: {{ID: id, Name: "Old label", Kind: repository.ClassificationKindFoodCategory}},
	}}
	generation := &filterOptionGenerationStub{}
	first := NewVersionedFilterOptionService(classifications, &filterOptionAllergenStub{}, generation)
	second := NewVersionedFilterOptionService(classifications, &filterOptionAllergenStub{}, generation)

	if _, err := first.Options(context.Background(), SearchModeSubstitution); err != nil {
		t.Fatalf("first Options() error = %v", err)
	}
	if _, err := second.Options(context.Background(), SearchModeSubstitution); err != nil {
		t.Fatalf("second Options() error = %v", err)
	}
	classifications.mu.Lock()
	classifications.entries[repository.ClassificationKindFoodCategory][0].Name = "New label"
	classifications.mu.Unlock()
	generation.Advance()
	first.Invalidate()

	refreshed, err := second.Options(context.Background(), SearchModeSubstitution)
	if err != nil {
		t.Fatalf("peer Options() error = %v", err)
	}
	if got := findFilterOption(t, refreshed.Options, SearchFilterKindFoodCategory, id.String()).Label; got != "New label" {
		t.Fatalf("peer label = %q, want shared invalidation result", got)
	}

	classifications.mu.Lock()
	classifications.entries[repository.ClassificationKindFoodCategory] = append(classifications.entries[repository.ClassificationKindFoodCategory], repository.ClassificationEntity{ID: createdID, Name: "Created label", Kind: repository.ClassificationKindFoodCategory})
	classifications.mu.Unlock()
	generation.Advance()
	first.Invalidate()
	created, err := second.Options(context.Background(), SearchModeSubstitution)
	if err != nil || !hasFilterOption(created.Options, SearchFilterKindFoodCategory, createdID.String()) {
		t.Fatalf("peer create options=%#v err=%v", created.Options, err)
	}

	classifications.mu.Lock()
	classifications.entries[repository.ClassificationKindFoodCategory] = classifications.entries[repository.ClassificationKindFoodCategory][:1]
	classifications.mu.Unlock()
	generation.Advance()
	first.Invalidate()
	deleted, err := second.Options(context.Background(), SearchModeSubstitution)
	if err != nil || hasFilterOption(deleted.Options, SearchFilterKindFoodCategory, createdID.String()) {
		t.Fatalf("peer delete options=%#v err=%v", deleted.Options, err)
	}
}

// Implements DESIGN-009 TagManager validation and dependency failure verification.
func TestFilterOptionServiceValidatesModeAndReturnsDependencyFailures(t *testing.T) {
	classifications := &filterOptionClassificationStub{entries: map[repository.ClassificationKind][]repository.ClassificationEntity{}, err: map[repository.ClassificationKind]error{}}
	allergens := &filterOptionAllergenStub{}
	service := NewFilterOptionService(classifications, allergens)
	if _, err := service.Options(context.Background(), SearchModeCatalog); !repository.IsKind(err, repository.ErrorKindValidation) || classifications.calls != 0 {
		t.Fatalf("unsupported mode error = %v calls=%d", err, classifications.calls)
	}

	dependencyErr := repository.NewError(repository.ErrorKindConnection, "sensitive dependency detail", errors.New("socket detail"))
	classifications.err[repository.ClassificationKindFoodCategory] = dependencyErr
	if _, err := service.Options(context.Background(), SearchModeSubstitution); !errors.Is(err, dependencyErr) || allergens.calls != 0 {
		t.Fatalf("classification failure = %v allergenCalls=%d", err, allergens.calls)
	}
	delete(classifications.err, repository.ClassificationKindFoodCategory)
	classifications.err[repository.ClassificationKindCulinaryRole] = dependencyErr
	if _, err := service.Options(context.Background(), SearchModeSubstitution); !errors.Is(err, dependencyErr) || allergens.calls != 0 {
		t.Fatalf("culinary-role failure = %v allergenCalls=%d", err, allergens.calls)
	}
	delete(classifications.err, repository.ClassificationKindCulinaryRole)
	allergens.err = dependencyErr
	if _, err := service.Options(context.Background(), SearchModeSubstitution); !errors.Is(err, dependencyErr) {
		t.Fatalf("allergen failure = %v", err)
	}
}

// Implements DESIGN-009 TagManager deterministic tie-break ordering verification.
func TestSortFilterOptionsBreaksEqualLabelTiesByPersistedID(t *testing.T) {
	options := []FilterOption{
		{FilterID: "b", Kind: SearchFilterKindAllergen, Label: "Same"},
		{FilterID: "a", Kind: SearchFilterKindAllergen, Label: "same"},
	}
	sortFilterOptions(options)
	if options[0].FilterID != "a" || options[1].FilterID != "b" {
		t.Fatalf("sortFilterOptions() = %#v", options)
	}
}

func findFilterOption(t *testing.T, options []FilterOption, kind SearchFilterKind, id string) FilterOption {
	t.Helper()
	for _, option := range options {
		if option.Kind == kind && option.FilterID == id {
			return option
		}
	}
	t.Fatalf("option kind=%q id=%q not found", kind, id)
	return FilterOption{}
}
