package repository

// Implements DESIGN-008 SavedDataRepository durable create idempotency verification.

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies(t *testing.T) {
	zeroMacros := mutatedDailyDietCreateResponsePayload(func(body map[string]any) {
		body["aggregateMacros"] = map[string]any{"protein": 0, "carbohydrates": 0, "fat": 0, "calories": 0}
	})
	if _, err := decodeDailyDietCreateResponse(zeroMacros(t)); err != nil {
		t.Fatalf("decodeDailyDietCreateResponse() zero-valued macros error = %v", err)
	}

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
	}{
		{name: "unknown field", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["unknown"] = true })},
		{name: "trailing JSON", payload: func(t *testing.T) []byte { return append(validDailyDietCreateResponsePayload(t), []byte(` {}`)...) }},
		{name: "malformed JSON", payload: staticDailyDietCreateResponsePayload(`{"id":`)},
		{name: "top-level array", payload: staticDailyDietCreateResponsePayload(`[]`)},
		{name: "wrong entry type", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["entries"] = []any{"invalid"} })},
		{name: "wrong macros type", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["aggregateMacros"] = []any{} })},
		{name: "missing macros", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { delete(body, "aggregateMacros") })},
		{name: "null macros", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["aggregateMacros"] = nil })},
		{name: "empty macros", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["aggregateMacros"] = map[string]any{} })},
		{name: "nil diet ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["id"] = nil })},
		{name: "empty diet ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["id"] = "" })},
		{name: "nil entry ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["id"] = nil })},
		{name: "empty entry ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["id"] = "" })},
		{name: "nil Food Object ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["foodObjectId"] = nil })},
		{name: "empty Food Object ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["foodObjectId"] = "" })},
		{name: "invalid Food Object type", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["foodObjectType"] = "ingredient" })},
		{name: "zero quantity", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["quantity"] = 0 })},
		{name: "negative quantity", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["quantity"] = -1 })},
		{name: "invalid unit", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["unit"] = "kg" })},
		{name: "negative position", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["position"] = -1 })},
		{name: "out-of-range position", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["position"] = 100 })},
		{name: "duplicate positions", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) {
			entry := responseEntry(body)
			body["entries"] = append(body["entries"].([]any), map[string]any{"id": uuid.NewString(), "foodObjectId": entry["foodObjectId"], "foodObjectType": entry["foodObjectType"], "quantity": 1, "unit": "g", "position": entry["position"]})
		})},
		{name: "negative protein", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseMacros(body)["protein"] = -1 })},
		{name: "negative carbohydrates", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseMacros(body)["carbohydrates"] = -1 })},
		{name: "negative fat", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseMacros(body)["fat"] = -1 })},
		{name: "negative calories", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseMacros(body)["calories"] = -1 })},
		{name: "legacy ID", payload: staticDailyDietCreateResponsePayload(`{"dailyDietId":"` + uuid.NewString() + `"}`)},
		{name: "dual ID", payload: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["dailyDietId"] = uuid.NewString() })},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeDailyDietCreateResponse(test.payload(t)); !IsKind(err, ErrorKindInternal) {
				t.Fatalf("decodeDailyDietCreateResponse() error = %v, want internal", err)
			}
		})
	}
}

func TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresSavedDataRepository(db)
	mealRepo := NewPostgresMealRepository(db)
	userID := createRepositoryUser(t, ctx, db, "daily-diet-claim@example.test")
	mealID, err := mealRepo.Create(ctx, MealEntity{Type: MealTypeSingle, Name: "Claim Meal", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 10}})
	if err != nil {
		t.Fatalf("create meal: %v", err)
	}
	claim := testDailyDietCreateClaim(userID, mealID, "durable-claim", strings.Repeat("a", 64))
	created, err := repo.ClaimDailyDietCreate(ctx, claim)
	if err != nil || created.Replayed || !reflect.DeepEqual(created.Response, claim.Response) {
		t.Fatalf("ClaimDailyDietCreate() = %+v error=%v", created, err)
	}
	duplicateNameClaim := testDailyDietCreateClaim(userID, mealID, "duplicate-name-claim", strings.Repeat("9", 64))
	if _, err := repo.ClaimDailyDietCreate(ctx, duplicateNameClaim); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("duplicate-name ClaimDailyDietCreate() error=%v, want conflict", err)
	}
	assertNoDailyDietClaimWrites(t, ctx, db, duplicateNameClaim)
	if err := repo.Replace(ctx, userID, SavedDiet{ID: claim.Diet.ID, Name: "Changed", Entries: []SavedDietMealEntry{{MealID: mealID, Quantity: 5, Unit: "g", Position: 0}}}); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE meals SET protein_per_100 = 999 WHERE id = $1`, mealID); err != nil {
		t.Fatalf("change macros: %v", err)
	}
	if err := repo.Delete(ctx, userID, claim.Diet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	replayed, err := repo.GetDailyDietCreateClaim(ctx, userID, claim.Key, claim.BodyHash)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Response, claim.Response) {
		t.Fatalf("GetDailyDietCreateClaim() = %+v error=%v", replayed, err)
	}
	if _, err := repo.GetDailyDietCreateClaim(ctx, userID, claim.Key, strings.Repeat("b", 64)); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("changed-body error = %v, want conflict", err)
	}
	if _, err := db.Exec(ctx, testUserDeleteSQL, userID); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	var remaining int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id = $1`, userID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("claim rows after account cascade=%d error=%v", remaining, err)
	}
}

func TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresSavedDataRepository(db)
	mealRepo := NewPostgresMealRepository(db)
	userID := createRepositoryUser(t, ctx, db, "daily-diet-invalid-claim@example.test")
	mealID, err := mealRepo.Create(ctx, MealEntity{Type: MealTypeSingle, Name: "Rollback Meal", PhysicalState: PhysicalStateSolid})
	if err != nil {
		t.Fatalf("create meal: %v", err)
	}

	persistedBodies := []struct {
		name string
		body []byte
	}{
		{name: "legacy ID", body: []byte(`{"dailyDietId":"` + uuid.NewString() + `"}`)},
		{name: "dual ID", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["dailyDietId"] = uuid.NewString() })(t)},
		{name: "unknown field", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["unknown"] = true })(t)},
		{name: "top-level array", body: []byte(`[]`)},
		{name: "wrong nested type", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["entries"] = []any{"invalid"} })(t)},
		{name: "invalid domain value", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { responseEntry(body)["quantity"] = 0 })(t)},
		{name: "missing macros", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { delete(body, "aggregateMacros") })(t)},
		{name: "null macros", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["aggregateMacros"] = nil })(t)},
		{name: "empty macros", body: mutatedDailyDietCreateResponsePayload(func(body map[string]any) { body["aggregateMacros"] = map[string]any{} })(t)},
	}
	for index, test := range persistedBodies {
		key := "invalid-body-" + string(rune('a'+index))
		if _, err := db.Exec(ctx, `INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body) VALUES ($1, 'POST', '/daily-diets', $2, $3, 201, $4::jsonb)`, userID, key, strings.Repeat("c", 64), test.body); err != nil {
			t.Fatalf("insert %s body: %v", test.name, err)
		}
		if _, err := repo.GetDailyDietCreateClaim(ctx, userID, key, strings.Repeat("c", 64)); !IsKind(err, ErrorKindInternal) {
			t.Fatalf("%s body error=%v, want internal", test.name, err)
		}
		var rows, diets int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id = $1 AND key = $2 AND response_body = $3::jsonb`, userID, key, test.body).Scan(&rows); err != nil || rows != 1 {
			t.Fatalf("%s persisted rows=%d error=%v, want one unchanged row", test.name, rows, err)
		}
		if err := db.QueryRow(ctx, `SELECT count(*) FROM saved_diets WHERE user_id = $1`, userID).Scan(&diets); err != nil || diets != 0 {
			t.Fatalf("%s read wrote diets=%d error=%v", test.name, diets, err)
		}
	}

	claim := testDailyDietCreateClaim(uuid.New(), mealID, "claim-stage-rollback", strings.Repeat("f", 64))
	if _, err := repo.ClaimDailyDietCreate(ctx, claim); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("claim-stage failure = %v, want validation", err)
	}
	var diets int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM saved_diets WHERE id = $1`, claim.Diet.ID).Scan(&diets); err != nil || diets != 0 {
		t.Fatalf("claim-stage rollback left diets=%d error=%v", diets, err)
	}

	claim = testDailyDietCreateClaim(userID, mealID, "parent-stage-rollback", strings.Repeat("1", 64))
	if _, err := db.Exec(ctx, `INSERT INTO saved_diets (id, user_id, name) VALUES ($1, $2, 'Existing')`, claim.Diet.ID, userID); err != nil {
		t.Fatalf("install parent failure: %v", err)
	}
	if _, err := repo.ClaimDailyDietCreate(ctx, claim); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("parent-stage failure = %v, want conflict", err)
	}
	var claims int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id = $1 AND key = $2`, userID, claim.Key).Scan(&claims); err != nil || claims != 0 {
		t.Fatalf("parent-stage rollback left claims=%d error=%v", claims, err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM saved_diets WHERE id = $1`, claim.Diet.ID); err != nil {
		t.Fatalf("remove parent failure fixture: %v", err)
	}

	claim = testDailyDietCreateClaim(userID, uuid.New(), "entry-rollback", strings.Repeat("d", 64))
	if _, err := repo.ClaimDailyDietCreate(ctx, claim); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("entry-stage failure = %v, want validation", err)
	}
	assertNoDailyDietClaimWrites(t, ctx, db, claim)

	if _, err := db.Exec(ctx, `ALTER TABLE saved_items ADD CONSTRAINT task216_reject_saved_diet CHECK (kind <> 'saved_diet')`); err != nil {
		t.Fatalf("install saved-item failure: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), `ALTER TABLE saved_items DROP CONSTRAINT IF EXISTS task216_reject_saved_diet`)
	})
	claim = testDailyDietCreateClaim(userID, mealID, "saved-item-rollback", strings.Repeat("e", 64))
	if _, err := repo.ClaimDailyDietCreate(ctx, claim); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("saved-item-stage failure = %v, want validation", err)
	}
	assertNoDailyDietClaimWrites(t, ctx, db, claim)
}

func staticDailyDietCreateResponsePayload(payload string) func(*testing.T) []byte {
	return func(*testing.T) []byte { return []byte(payload) }
}

func mutatedDailyDietCreateResponsePayload(mutate func(map[string]any)) func(*testing.T) []byte {
	return func(t *testing.T) []byte {
		t.Helper()
		var body map[string]any
		if err := json.Unmarshal(validDailyDietCreateResponsePayload(t), &body); err != nil {
			t.Fatalf("decode valid response fixture: %v", err)
		}
		mutate(body)
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode mutated response fixture: %v", err)
		}
		return payload
	}
}

func validDailyDietCreateResponsePayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(testDailyDietCreateClaim(uuid.New(), uuid.New(), "strict-response", strings.Repeat("a", 64)).Response)
	if err != nil {
		t.Fatalf("encode valid response fixture: %v", err)
	}
	return payload
}

func responseEntry(body map[string]any) map[string]any {
	return body["entries"].([]any)[0].(map[string]any)
}

func responseMacros(body map[string]any) map[string]any {
	return body["aggregateMacros"].(map[string]any)
}

func testDailyDietCreateClaim(userID, mealID uuid.UUID, key, hash string) DailyDietCreateClaim {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	dietID, entryID := uuid.New(), uuid.New()
	entry := SavedDietMealEntry{ID: entryID, SavedDietID: dietID, FoodObjectID: mealID, FoodObjectType: FoodObjectTypeMeal, Quantity: 100, Unit: "g", Position: 0, CreatedAt: now}
	responseEntry := DailyDietCreateResponseEntry{ID: entryID, FoodObjectID: mealID, FoodObjectType: FoodObjectTypeMeal, Quantity: 100, Unit: "g", Position: 0}
	response := DailyDietCreateResponse{ID: dietID, Name: "Original", Entries: []DailyDietCreateResponseEntry{responseEntry}, AggregateMacros: DailyDietCreateResponseMacros{Protein: 10, Calories: 40}, CreatedAt: now, UpdatedAt: now}
	return DailyDietCreateClaim{UserID: userID, Key: key, BodyHash: hash, Diet: SavedDiet{ID: dietID, UserID: userID, Name: "Original", Entries: []SavedDietMealEntry{entry}, CreatedAt: now, UpdatedAt: now}, Response: response, StatusCode: 201}
}

func assertNoDailyDietClaimWrites(t *testing.T, ctx context.Context, db *pgxpool.Pool, claim DailyDietCreateClaim) {
	t.Helper()
	var claims, diets int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id = $1 AND key = $2`, claim.UserID, claim.Key).Scan(&claims); err != nil {
		t.Fatalf("count claims: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM saved_diets WHERE id = $1`, claim.Diet.ID).Scan(&diets); err != nil {
		t.Fatalf("count diets: %v", err)
	}
	if claims != 0 || diets != 0 {
		t.Fatalf("rollback left claims=%d diets=%d", claims, diets)
	}
}
