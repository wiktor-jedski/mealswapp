package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/tag"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/repositories"
	"mealswapp/backend/internal/services/externaldata"

	"github.com/google/uuid"
)

func TestAdminControllerAllowsAdminSummary(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	summary := fakeAdminSummaryService{
		result: handlers.AdminSummary{
			PendingImports:   3,
			PendingItems:     4,
			ActiveUsers:      12,
			RecentAuditCount: 5,
			GeneratedAt:      time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC),
		},
	}
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		AuthService:         auth,
		AdminSummaryService: summary,
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", token, false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected admin summary 200, got %d", res.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	if data["pendingImports"] != float64(3) || data["activeUsers"] != float64(12) {
		t.Fatalf("unexpected admin summary: %#v", data)
	}
}

func TestAdminControllerRejectsNonAdminAndUnauthenticatedUsers(t *testing.T) {
	auth := newFakeAuthService()
	token := auth.issueTokens().AccessToken
	app := NewRouter(ServiceDependencies{
		Config:      config.Config{Environment: "test"},
		AuthService: auth,
	})

	forbidden := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", token, false)
	defer forbidden.Body.Close()
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("expected non-admin 403, got %d", forbidden.StatusCode)
	}
	forbiddenPayload := decodeEnvelope(t, forbidden)
	if forbiddenPayload.Error == nil || forbiddenPayload.Error.Code != "forbidden" || forbiddenPayload.Error.Message != "Forbidden" {
		t.Fatalf("expected audit-safe forbidden envelope, got %#v", forbiddenPayload.Error)
	}

	unauthorized := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", "", false)
	defer unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated 401, got %d", unauthorized.StatusCode)
	}
	unauthorizedPayload := decodeEnvelope(t, unauthorized)
	if unauthorizedPayload.Error == nil || unauthorizedPayload.Error.Code != "unauthorized" {
		t.Fatalf("expected unauthorized envelope, got %#v", unauthorizedPayload.Error)
	}
}

func TestAdminControllerSearchesExternalProviders(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	external := &fakeExternalSearchService{
		result: externaldata.ExternalSearchResult{
			Page:     2,
			PageSize: 15,
			Candidates: []externaldata.NormalizedFoodCandidate{{
				Provider:      externaldata.ProviderOpenFoodFacts,
				ExternalID:    "737628064502",
				Name:          "Organic Tofu",
				PhysicalState: food.PhysicalStateSolid,
				MacrosPer100:  food.MacroValues{ProteinGrams: 12.3, CarbsGrams: 1.7, FatGrams: 6.1},
				ImageURL:      "https://example.test/tofu.jpg",
			}},
		},
	}
	app := NewRouter(ServiceDependencies{
		Config:                config.Config{Environment: "test"},
		AuthService:           auth,
		ExternalSearchService: external,
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/external-search?query=tofu&provider=openfoodfacts&page=2&pageSize=15", "", token, false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected external search 200, got %d", res.StatusCode)
	}
	if external.query.Query != "tofu" || external.query.Provider != externaldata.ProviderOpenFoodFacts || external.query.Page != 2 || external.query.PageSize != 15 {
		t.Fatalf("unexpected external query: %#v", external.query)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	candidates, ok := data["candidates"].([]any)
	if !ok || len(candidates) != 1 {
		t.Fatalf("expected candidate list, got %#v", data["candidates"])
	}
	candidate := dataMap(t, candidates[0])
	if candidate["provider"] != "openfoodfacts" || candidate["name"] != "Organic Tofu" {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
}

func TestAdminControllerMapsExternalProviderErrors(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{
			name:   "bad provider",
			err:    externaldata.ProviderError{Provider: "bad", Kind: externaldata.ProviderErrorInvalidQuery, Message: "Unsupported external provider"},
			status: http.StatusBadRequest,
			code:   "validation_error",
		},
		{
			name:   "rate limit",
			err:    externaldata.ProviderError{Provider: externaldata.ProviderUSDA, Kind: externaldata.ProviderErrorRateLimited, Message: "USDA rate limited", Retryable: true},
			status: http.StatusTooManyRequests,
			code:   "rate_limited",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := NewRouter(ServiceDependencies{
				Config:                config.Config{Environment: "test"},
				AuthService:           auth,
				ExternalSearchService: &fakeExternalSearchService{err: tc.err},
			})
			res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/external-search?query=tofu&provider=usda", "", token, false)
			defer res.Body.Close()
			if res.StatusCode != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, res.StatusCode)
			}
			payload := decodeEnvelope(t, res)
			if payload.Error == nil || payload.Error.Code != tc.code {
				t.Fatalf("expected error code %s, got %#v", tc.code, payload.Error)
			}
		})
	}
}

func TestAdminControllerExternalSearchRequiresAdmin(t *testing.T) {
	auth := newFakeAuthService()
	token := auth.issueTokens().AccessToken
	app := NewRouter(ServiceDependencies{
		Config:                config.Config{Environment: "test"},
		AuthService:           auth,
		ExternalSearchService: &fakeExternalSearchService{},
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/external-search?query=tofu&provider=usda", "", token, false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected non-admin 403, got %d", res.StatusCode)
	}
}

func TestAdminControllerItemCRUDAndTransitions(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	itemID := uuid.New()
	items := &fakeItemCuratorService{item: adminTestFood(itemID, "Draft Tofu")}
	app := NewRouter(ServiceDependencies{
		Config:             config.Config{Environment: "test"},
		AuthService:        auth,
		ItemCuratorService: items,
	})

	list := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/items?query=tof&page=2&pageSize=5", "", token, false)
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK || items.query != "tof" || items.page != 2 || items.limit != 5 {
		t.Fatalf("unexpected list response/query: status=%d query=%q page=%d limit=%d", list.StatusCode, items.query, items.page, items.limit)
	}

	create := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/items", adminItemJSON("Created Tofu", "draft"), token, true)
	defer create.Body.Close()
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.StatusCode)
	}

	update := performJSONRequest(t, app, http.MethodPatch, "/api/v1/admin/items/"+itemID.String(), adminItemJSON("Edited Tofu", "draft"), token, true)
	defer update.Body.Close()
	if update.StatusCode != http.StatusOK || items.updatedID != itemID || items.updated.Name != "Edited Tofu" {
		t.Fatalf("unexpected update state: status=%d id=%s item=%#v", update.StatusCode, items.updatedID, items.updated)
	}

	approve := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/items/"+itemID.String()+"/approve", "", token, true)
	defer approve.Body.Close()
	if approve.StatusCode != http.StatusOK || items.transition != externaldata.TransitionApprove {
		t.Fatalf("unexpected approve transition: status=%d transition=%s", approve.StatusCode, items.transition)
	}

	deleteRes := performJSONRequest(t, app, http.MethodDelete, "/api/v1/admin/items/"+itemID.String(), "", token, true)
	defer deleteRes.Body.Close()
	if deleteRes.StatusCode != http.StatusNoContent || items.deletedID != itemID {
		t.Fatalf("unexpected delete: status=%d id=%s", deleteRes.StatusCode, items.deletedID)
	}
}

func TestAdminControllerItemValidationAndInvalidTransition(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	itemID := uuid.New()
	app := NewRouter(ServiceDependencies{
		Config:             config.Config{Environment: "test"},
		AuthService:        auth,
		ItemCuratorService: &fakeItemCuratorService{item: adminTestFood(itemID, "Draft Tofu"), transitionErr: externaldata.ErrItemCuratorInvalidTransition},
	})

	invalidCreate := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/items", `{}`, token, true)
	defer invalidCreate.Body.Close()
	if invalidCreate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid create 400, got %d", invalidCreate.StatusCode)
	}

	invalidTransition := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/items/"+itemID.String()+"/archive", "", token, true)
	defer invalidTransition.Body.Close()
	if invalidTransition.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid transition 400, got %d", invalidTransition.StatusCode)
	}
}

func TestAdminControllerTagManagement(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	foodID := uuid.New()
	tagID := uuid.New()
	targetID := uuid.New()
	tags := &fakeTagManagerService{tag: tag.TagEntity{ID: tagID, Name: "Vegan", Kind: tag.KindDiet, Active: true}}
	app := NewRouter(ServiceDependencies{
		Config:            config.Config{Environment: "test"},
		AuthService:       auth,
		TagManagerService: tags,
	})

	list := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/tags?kind=diet", "", token, false)
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK || tags.kind != tag.KindDiet {
		t.Fatalf("unexpected tag list: status=%d kind=%s", list.StatusCode, tags.kind)
	}

	upsert := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/tags", `{"Name":"High protein","Kind":"functionality"}`, token, true)
	defer upsert.Body.Close()
	if upsert.StatusCode != http.StatusOK || tags.upserted.Kind != tag.KindFunctionality {
		t.Fatalf("unexpected tag upsert: status=%d tag=%#v", upsert.StatusCode, tags.upserted)
	}

	assign := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/items/"+foodID.String()+"/tags", `{"tagId":"`+tagID.String()+`"}`, token, true)
	defer assign.Body.Close()
	if assign.StatusCode != http.StatusNoContent || tags.assignedFoodID != foodID || tags.assignedTagID != tagID {
		t.Fatalf("unexpected assign: status=%d food=%s tag=%s", assign.StatusCode, tags.assignedFoodID, tags.assignedTagID)
	}

	remove := performJSONRequest(t, app, http.MethodDelete, "/api/v1/admin/items/"+foodID.String()+"/tags/"+tagID.String(), "", token, true)
	defer remove.Body.Close()
	if remove.StatusCode != http.StatusNoContent || tags.removedFoodID != foodID || tags.removedTagID != tagID {
		t.Fatalf("unexpected remove: status=%d food=%s tag=%s", remove.StatusCode, tags.removedFoodID, tags.removedTagID)
	}

	merge := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/tags/merge", `{"sourceId":"`+tagID.String()+`","targetId":"`+targetID.String()+`"}`, token, true)
	defer merge.Body.Close()
	if merge.StatusCode != http.StatusNoContent || tags.mergeSourceID != tagID || tags.mergeTargetID != targetID {
		t.Fatalf("unexpected merge: status=%d source=%s target=%s", merge.StatusCode, tags.mergeSourceID, tags.mergeTargetID)
	}

	deactivate := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/tags/"+tagID.String()+"/deactivate", "", token, true)
	defer deactivate.Body.Close()
	if deactivate.StatusCode != http.StatusNoContent || tags.deactivatedID != tagID {
		t.Fatalf("unexpected deactivate: status=%d id=%s", deactivate.StatusCode, tags.deactivatedID)
	}
}

func TestAdminControllerRejectsInvalidTagTaxonomy(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	app := NewRouter(ServiceDependencies{
		Config:            config.Config{Environment: "test"},
		AuthService:       auth,
		TagManagerService: &fakeTagManagerService{err: tag.ErrInvalidKind},
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/tags", `{"Name":"Category","Kind":"category"}`, token, true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid taxonomy 400, got %d", res.StatusCode)
	}
}

func TestAdminControllerUserAdminActions(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	userID := uuid.New()
	users := &fakeUserAdminService{
		detail: externaldata.UserAdminDetail{
			User:        repositories.UserEntity{ID: userID, Email: "user@example.com", DisplayName: "User"},
			Entitlement: &repositories.EntitlementEntity{UserID: userID, Plan: "paid", Status: "active"},
		},
		list:  externaldata.UserAdminListResult{Users: []repositories.UserEntity{{ID: userID, Email: "user@example.com"}}, Total: 1, Page: 2, Limit: 5},
		audit: externaldata.UserAuditHistory{Entries: []repositories.AuditLogEntity{{ID: uuid.New(), Target: "user:" + userID.String(), Action: "admin.disable_user"}}, Total: 1, Page: 1, Limit: 10},
	}
	app := NewRouter(ServiceDependencies{
		Config:           config.Config{Environment: "test"},
		AuthService:      auth,
		UserAdminService: users,
	})

	list := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/users?query=user&page=2&pageSize=5", "", token, false)
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK || users.query != "user" || users.page != 2 || users.limit != 5 {
		t.Fatalf("unexpected user list: status=%d query=%q page=%d limit=%d", list.StatusCode, users.query, users.page, users.limit)
	}

	detail := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/users/"+userID.String(), "", token, false)
	defer detail.Body.Close()
	if detail.StatusCode != http.StatusOK || users.detailID != userID {
		t.Fatalf("unexpected user detail: status=%d id=%s", detail.StatusCode, users.detailID)
	}

	disable := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/users/"+userID.String()+"/disable", "", token, true)
	defer disable.Body.Close()
	if disable.StatusCode != http.StatusOK || users.disabledID != userID {
		t.Fatalf("unexpected disable: status=%d id=%s", disable.StatusCode, users.disabledID)
	}

	reset := performJSONRequest(t, app, http.MethodPost, "/api/v1/admin/users/"+userID.String()+"/reset-lockout", "", token, true)
	defer reset.Body.Close()
	if reset.StatusCode != http.StatusNoContent || users.resetID != userID {
		t.Fatalf("unexpected reset lockout: status=%d id=%s", reset.StatusCode, users.resetID)
	}

	audit := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/users/"+userID.String()+"/audit", "", token, false)
	defer audit.Body.Close()
	if audit.StatusCode != http.StatusOK || users.auditID != userID {
		t.Fatalf("unexpected audit history: status=%d id=%s", audit.StatusCode, users.auditID)
	}
}

func TestAdminControllerUserAdminRequiresAdmin(t *testing.T) {
	auth := newFakeAuthService()
	token := auth.issueTokens().AccessToken
	app := NewRouter(ServiceDependencies{
		Config:           config.Config{Environment: "test"},
		AuthService:      auth,
		UserAdminService: &fakeUserAdminService{},
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/users", "", token, false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected user admin forbidden for non-admin, got %d", res.StatusCode)
	}
}

type fakeAdminSummaryService struct {
	result handlers.AdminSummary
}

func (service fakeAdminSummaryService) Summary(ctx context.Context, admin handlers.AdminContext) (handlers.AdminSummary, error) {
	return service.result, nil
}

type fakeExternalSearchService struct {
	query  externaldata.ExternalSearchQuery
	result externaldata.ExternalSearchResult
	err    error
}

type fakeItemCuratorService struct {
	item          food.FoodItemEntity
	query         string
	page          int
	limit         int
	updatedID     uuid.UUID
	updated       food.FoodItemEntity
	transitionID  uuid.UUID
	transition    externaldata.CurationTransition
	transitionErr error
	deletedID     uuid.UUID
}

func (service *fakeItemCuratorService) List(ctx context.Context, query string, page int, limit int) (externaldata.ItemListResult, error) {
	service.query = query
	service.page = page
	service.limit = limit
	return externaldata.ItemListResult{Items: []food.FoodItemEntity{service.item}, Total: 1, Page: page, Limit: limit}, nil
}

func (service *fakeItemCuratorService) Get(ctx context.Context, id uuid.UUID) (food.FoodItemEntity, error) {
	return service.item, nil
}

func (service *fakeItemCuratorService) Create(ctx context.Context, item food.FoodItemEntity) (food.FoodItemEntity, error) {
	if err := item.Validate(); err != nil {
		return food.FoodItemEntity{}, err
	}
	item.ID = uuid.New()
	return item, nil
}

func (service *fakeItemCuratorService) Update(ctx context.Context, id uuid.UUID, item food.FoodItemEntity) (food.FoodItemEntity, error) {
	if err := item.Validate(); err != nil {
		return food.FoodItemEntity{}, err
	}
	service.updatedID = id
	service.updated = item
	item.ID = id
	return item, nil
}

func (service *fakeItemCuratorService) Transition(ctx context.Context, id uuid.UUID, transition externaldata.CurationTransition) (food.FoodItemEntity, error) {
	service.transitionID = id
	service.transition = transition
	if service.transitionErr != nil {
		return food.FoodItemEntity{}, service.transitionErr
	}
	item := service.item
	item.ID = id
	item.Source.CurationState = string(transition)
	return item, nil
}

func (service *fakeItemCuratorService) Delete(ctx context.Context, id uuid.UUID) error {
	service.deletedID = id
	return nil
}

func adminTestFood(id uuid.UUID, name string) food.FoodItemEntity {
	item := food.FoodItemEntity{
		ID:             id,
		Name:           name,
		PhysicalState:  food.PhysicalStateSolid,
		ServingUnit:    food.ServingUnitGram,
		ServingSize:    100,
		CaloriesPer100: 120,
		MacrosPer100:   food.MacroValues{ProteinGrams: 12, CarbsGrams: 2, FatGrams: 6},
		Micros:         map[string]float64{},
		Source:         food.SourceMetadata{CurationState: "draft"},
	}
	return item
}

func adminItemJSON(name string, state string) string {
	return `{"name":"` + name + `","physicalState":"solid","servingUnit":"gram","servingSize":100,"caloriesPer100":120,"macrosPer100":{"ProteinGrams":12,"CarbsGrams":2,"FatGrams":6},"micros":{},"source":{"CurationState":"` + state + `"}}`
}

type fakeTagManagerService struct {
	tag            tag.TagEntity
	err            error
	kind           tag.Kind
	upserted       tag.TagEntity
	assignedFoodID uuid.UUID
	assignedTagID  uuid.UUID
	removedFoodID  uuid.UUID
	removedTagID   uuid.UUID
	mergeSourceID  uuid.UUID
	mergeTargetID  uuid.UUID
	deactivatedID  uuid.UUID
}

type fakeUserAdminService struct {
	query      string
	page       int
	limit      int
	list       externaldata.UserAdminListResult
	detailID   uuid.UUID
	detail     externaldata.UserAdminDetail
	disabledID uuid.UUID
	resetID    uuid.UUID
	auditID    uuid.UUID
	audit      externaldata.UserAuditHistory
}

func (service *fakeUserAdminService) List(ctx context.Context, query string, page int, limit int) (externaldata.UserAdminListResult, error) {
	service.query = query
	service.page = page
	service.limit = limit
	return service.list, nil
}

func (service *fakeUserAdminService) Detail(ctx context.Context, userID uuid.UUID) (externaldata.UserAdminDetail, error) {
	service.detailID = userID
	return service.detail, nil
}

func (service *fakeUserAdminService) Disable(ctx context.Context, userID uuid.UUID) (repositories.UserEntity, error) {
	service.disabledID = userID
	user := service.detail.User
	user.ID = userID
	user.Disabled = true
	return user, nil
}

func (service *fakeUserAdminService) ResetLockout(ctx context.Context, userID uuid.UUID) error {
	service.resetID = userID
	return nil
}

func (service *fakeUserAdminService) AuditHistory(ctx context.Context, userID uuid.UUID, page int, limit int) (externaldata.UserAuditHistory, error) {
	service.auditID = userID
	return service.audit, nil
}

func (service *fakeTagManagerService) List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error) {
	service.kind = kind
	if service.err != nil {
		return nil, service.err
	}
	return []tag.TagEntity{service.tag}, nil
}

func (service *fakeTagManagerService) Upsert(ctx context.Context, entity tag.TagEntity) (tag.TagEntity, error) {
	service.upserted = entity
	if service.err != nil {
		return tag.TagEntity{}, service.err
	}
	entity.ID = uuid.New()
	entity.Active = true
	return entity, nil
}

func (service *fakeTagManagerService) Assign(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	service.assignedFoodID = foodItemID
	service.assignedTagID = tagID
	return service.err
}

func (service *fakeTagManagerService) Remove(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	service.removedFoodID = foodItemID
	service.removedTagID = tagID
	return service.err
}

func (service *fakeTagManagerService) Deactivate(ctx context.Context, id uuid.UUID) error {
	service.deactivatedID = id
	return service.err
}

func (service *fakeTagManagerService) Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error {
	service.mergeSourceID = sourceID
	service.mergeTargetID = targetID
	return service.err
}

func (service *fakeExternalSearchService) SearchExternalFoods(ctx context.Context, query externaldata.ExternalSearchQuery) (externaldata.ExternalSearchResult, error) {
	service.query = query
	if service.err != nil {
		return externaldata.ExternalSearchResult{}, service.err
	}
	return service.result, nil
}
