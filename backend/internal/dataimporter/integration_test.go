package dataimporter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dataimporter"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/testdatabase"
)

// TestCuratedImportTransactionalWorkflow verifies IT-ARCH-009-002 and
// IT-ARCH-009-003, ARCH-009, DESIGN-009 DataImporter, and
// SW-REQ-055/SW-REQ-090 through real PostgreSQL and search collaborators.
// Implements DESIGN-009 DataImporter integration behavior.
func TestCuratedImportTransactionalWorkflow(t *testing.T) {
	migrations, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatal(err)
	}
	db := testdatabase.Reset(t, migrations)
	ctx := context.Background()
	var adminID uuid.UUID
	if err := db.QueryRow(ctx, `INSERT INTO users (email_key_version,email_nonce,email_ciphertext,normalized_email_lookup_key_version,normalized_email_digest,password_hash,password_salt,role) VALUES ('test-v1',decode('01','hex'),decode('02','hex'),'test-v1',$1,'hash','salt','admin') RETURNING id`, "task249-admin").Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	audit := repository.NewPostgresAdminImportAuditRepository(db)
	service := dataimporter.NewService(audit)
	foods := repository.NewPostgresFoodItemRepository(db)
	classifications := repository.NewPostgresClassificationRepository(db)
	categoryID, err := classifications.Upsert(ctx, repository.ClassificationEntity{Name: "Task 249 protein", Kind: repository.ClassificationKindFoodCategory})
	if err != nil {
		t.Fatal(err)
	}
	roleID, err := classifications.Upsert(ctx, repository.ClassificationEntity{Name: "Task 249 staple", Kind: repository.ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatal(err)
	}
	sourceID, err := foods.Create(ctx, repository.FoodItemEntity{Name: "Task 249 source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Fat: 5}, Micros: repository.MicroValues{}})
	if err != nil {
		t.Fatal(err)
	}

	base := dataimporter.Request{SourceProvider: "usda", ExternalID: "task-249-natural", Request: customitem.Request{
		Name: "Task 249 imported tofu", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 18, Carbohydrates: 4, Fat: 8},
		Micros: repository.MicroValues{"Sodium": 4}, FoodCategoryIDs: []uuid.UUID{categoryID}, CulinaryRoleIDs: []uuid.UUID{roleID},
	}}
	created, err := confirm(t, ctx, audit, service, adminID, "", base, false)
	if err != nil || created.Replayed || created.FoodItemID == uuid.Nil || created.ImportID == uuid.Nil {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	replayed, err := confirm(t, ctx, audit, service, adminID, "", base, false)
	if err != nil || !replayed.Replayed || replayed.FoodItemID != created.FoodItemID || replayed.ImportID != created.ImportID {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	assertCounts(t, ctx, db, created.FoodItemID, created.ImportID, 1, 1, 1)

	changedNatural := base
	changedNatural.MacrosPer100.Protein = 17
	if _, err := confirm(t, ctx, audit, service, adminID, "", changedNatural, false); !errors.Is(err, dataimporter.ErrProviderConflict) {
		t.Fatalf("natural body conflict=%v", err)
	}

	withoutIdentity := base
	withoutIdentity.SourceProvider, withoutIdentity.ExternalID, withoutIdentity.Name = "", "", "Task 249 key import"
	keyCreated, err := confirm(t, ctx, audit, service, adminID, "task-249-key-0001", withoutIdentity, false)
	if err != nil {
		t.Fatal(err)
	}
	keyReplay, err := confirm(t, ctx, audit, service, adminID, "task-249-key-0001", withoutIdentity, false)
	if err != nil || !keyReplay.Replayed || keyReplay.ImportID != keyCreated.ImportID {
		t.Fatalf("key replay=%+v err=%v", keyReplay, err)
	}
	assertCounts(t, ctx, db, keyCreated.FoodItemID, keyCreated.ImportID, 1, 1, 1)
	changedKey := withoutIdentity
	changedKey.MacrosPer100.Fat++
	if _, err := confirm(t, ctx, audit, service, adminID, "task-249-key-0001", changedKey, false); !errors.Is(err, dataimporter.ErrIdempotencyConflict) {
		t.Fatalf("key body conflict=%v", err)
	}

	nameConflict := base
	nameConflict.ExternalID = "task-249-name-conflict"
	nameConflict.Name = base.Name
	nameConflict.MacrosPer100.Protein = 21
	if _, err := confirm(t, ctx, audit, service, adminID, "", nameConflict, false); !errors.Is(err, dataimporter.ErrNameConfirmation) {
		t.Fatalf("name conflict=%v", err)
	}
	nameConflict.ConfirmNameConflict = true
	merged, err := confirm(t, ctx, audit, service, adminID, "", nameConflict, false)
	if err != nil || !merged.Merged || merged.FoodItemID != created.FoodItemID {
		t.Fatalf("merged=%+v err=%v", merged, err)
	}

	invalidClassification := base
	invalidClassification.ExternalID, invalidClassification.Name = "task-249-bad-class", "Task 249 bad class"
	invalidClassification.FoodCategoryIDs = []uuid.UUID{uuid.New()}
	if _, err := confirm(t, ctx, audit, service, adminID, "", invalidClassification, false); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("classification error=%v", err)
	}
	invalidMicro := base
	invalidMicro.ExternalID, invalidMicro.Name, invalidMicro.Micros = "task-249-bad-micro", "Task 249 bad micro", repository.MicroValues{"Unknown": 1}
	if _, err := confirm(t, ctx, audit, service, adminID, "", invalidMicro, false); !repository.IsKind(err, repository.ErrorKindInvalidMicronutrientKey) {
		t.Fatalf("micronutrient error=%v", err)
	}
	liquid := base
	liquid.ExternalID, liquid.Name, liquid.PhysicalState = "task-249-liquid", "Task 249 liquid", repository.PhysicalStateLiquid
	if _, err := confirm(t, ctx, audit, service, adminID, "", liquid, false); err == nil {
		t.Fatal("liquid without corrected density accepted")
	}
	liquid.DensityGramsPerMilliliter, liquid.DensitySourceKind = 1.03, "imported"
	if _, err := confirm(t, ctx, audit, service, adminID, "", liquid, false); err == nil {
		t.Fatal("imported liquid density without provider evidence accepted")
	}
	assertNameAbsent(t, ctx, db, liquid.Name)
	liquid.DensitySourceKind = "manual"
	liquidResult, err := confirm(t, ctx, audit, service, adminID, "", liquid, false)
	if err != nil || liquidResult.FoodItemID == uuid.Nil {
		t.Fatalf("corrected liquid=%+v err=%v", liquidResult, err)
	}
	estimated := liquid
	estimated.ExternalID, estimated.Name, estimated.DensitySourceKind = "task-249-liquid-estimated", "Task 249 liquid estimated", "estimated"
	if result, err := confirm(t, ctx, audit, service, adminID, "", estimated, false); err != nil || result.FoodItemID == uuid.Nil {
		t.Fatalf("estimated liquid=%+v err=%v", result, err)
	}
	providerDensity := liquid
	providerDensity.ExternalID, providerDensity.Name, providerDensity.DensitySourceKind = "task-249-liquid-provider", "Task 249 liquid provider", "imported"
	providerDensity.DensitySourceProvider, providerDensity.DensitySourceFoodID = " USDA ", " density-record-1 "
	if result, err := confirm(t, ctx, audit, service, adminID, "", providerDensity, false); err != nil || result.FoodItemID == uuid.Nil {
		t.Fatalf("provider-evidenced liquid=%+v err=%v", result, err)
	}

	rollback := base
	rollback.ExternalID, rollback.Name = "task-249-rollback", "Task 249 rollback"
	if _, err := confirm(t, ctx, audit, service, adminID, "", rollback, true); !errors.Is(err, repository.ErrAdminAuditPersistence) {
		t.Fatalf("audit rollback error=%v", err)
	}
	assertNameAbsent(t, ctx, db, rollback.Name)
	badRepository := withoutIdentity
	badRepository.Name, badRepository.FoodCategoryIDs = "Task 249 repository rollback", []uuid.UUID{uuid.New()}
	if _, err := confirm(t, ctx, audit, service, adminID, "task-249-rollback-key", badRepository, false); err == nil {
		t.Fatal("repository failure error=nil")
	}
	assertNameAbsent(t, ctx, db, badRepository.Name)
	var claims int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id=$1 AND key='task-249-rollback-key'`, adminID).Scan(&claims); err != nil || claims != 0 {
		t.Fatalf("rolled back claims=%d err=%v", claims, err)
	}

	catalog, err := search.NewCatalogService(foods, nil).Search(ctx, search.SearchRequest{Query: base.Name, Mode: search.SearchModeCatalog, Page: 1})
	if err != nil || !slices.ContainsFunc(catalog.Items, func(item repository.FoodItemEntity) bool { return item.ID == created.FoodItemID }) {
		t.Fatalf("catalog=%+v err=%v", catalog, err)
	}
	substitutions, err := search.NewSubstitutionService(foods, nil).Search(ctx, search.SearchRequest{Mode: search.SearchModeSubstitution, Page: 1, SubstitutionInputs: []search.SubstitutionInput{{FoodObjectID: sourceID, Quantity: 100, Unit: "g"}}})
	if err != nil || !slices.ContainsFunc(substitutions.Items, func(item repository.FoodItemEntity) bool { return item.ID == created.FoodItemID }) {
		t.Fatalf("substitutions=%+v err=%v", substitutions, err)
	}

	assertImmutableReplay(t, ctx, db, audit, service, adminID, "", base, created)
	assertImmutableReplay(t, ctx, db, audit, service, adminID, "task-249-key-0001", withoutIdentity, keyCreated)

	for _, status := range []string{"draft", "conflict", "rejected"} {
		statusRequest := base
		statusRequest.ExternalID, statusRequest.Name = "task-249-status-"+status, "Task 249 status "+status
		statusCreated, err := confirm(t, ctx, audit, service, adminID, "", statusRequest, false)
		if err != nil {
			t.Fatalf("create %s status fixture: %v", status, err)
		}
		if _, err := db.Exec(ctx, `UPDATE curated_imports SET status=$2 WHERE id=$1`, statusCreated.ImportID, status); err != nil {
			t.Fatal(err)
		}
		if _, err := confirm(t, ctx, audit, service, adminID, "", statusRequest, false); !errors.Is(err, dataimporter.ErrProviderConflict) {
			t.Fatalf("natural replay status=%s error=%v, want provider conflict", status, err)
		}
		assertCounts(t, ctx, db, statusCreated.FoodItemID, statusCreated.ImportID, 1, 1, 1)
	}

	assertConcurrentNameConfirmation(t, ctx, db, audit, service, adminID, false)
	assertConcurrentNameConfirmation(t, ctx, db, audit, service, adminID, true)
}

// TestCuratedImportConfirmedMergeInvalidatesRedisSimilarity proves merged macro changes cannot reuse stale scores.
// Implements DESIGN-009 DataImporter immediate substitution visibility and DESIGN-011 CacheInvalidator.
func TestCuratedImportConfirmedMergeInvalidatesRedisSimilarity(t *testing.T) {
	migrations, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatal(err)
	}
	db := testdatabase.Reset(t, migrations)
	ctx := context.Background()
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13"
	}
	redisClient, err := cache.Open(redisURL)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	var adminID uuid.UUID
	if err := db.QueryRow(ctx, `INSERT INTO users (email_key_version,email_nonce,email_ciphertext,normalized_email_lookup_key_version,normalized_email_digest,password_hash,password_salt,role) VALUES ('test-v1',decode('01','hex'),decode('02','hex'),'test-v1',$1,'hash','salt','admin') RETURNING id`, "task249-cache-admin").Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	audit := repository.NewPostgresAdminImportAuditRepository(db)
	imports := dataimporter.NewService(audit)
	foods := repository.NewPostgresFoodItemRepository(db)
	sourceID, err := foods.Create(ctx, repository.FoodItemEntity{Name: "Task 249 cache source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}})
	if err != nil {
		t.Fatal(err)
	}
	draft := dataimporter.Request{SourceProvider: "usda", ExternalID: "task-249-cache-first", Request: customitem.Request{
		Name: "Task 249 cache candidate", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{},
	}}
	created, err := confirm(t, ctx, audit, imports, adminID, "", draft, false)
	if err != nil {
		t.Fatal(err)
	}

	generation := cache.NewClassificationGeneration(redisClient)
	similarityStore := cache.SearchResponseStore{Store: cache.GoRedisStore{Client: redisClient}, Generation: generation}
	searchService := search.NewSubstitutionService(foods, nil, similarityStore)
	request := search.SearchRequest{Mode: search.SearchModeSubstitution, Page: 1, SubstitutionInputs: []search.SubstitutionInput{{FoodObjectID: sourceID, Quantity: 100, Unit: "g"}}}
	before, err := searchService.Search(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	beforeScore, ok := scoreForFood(before, created.FoodItemID)
	if !ok {
		t.Fatalf("candidate absent before merge: %+v", before)
	}

	merge := draft
	merge.ExternalID = "task-249-cache-merge"
	merge.ConfirmNameConflict = true
	merge.MacrosPer100 = repository.MacroValues{Protein: 9, Carbohydrates: 1}
	merged, err := confirm(t, ctx, audit, imports, adminID, "", merge, false)
	if err != nil || !merged.Merged || merged.FoodItemID != created.FoodItemID {
		t.Fatalf("merged=%+v err=%v", merged, err)
	}
	cache.NewClassificationInvalidator(nil, redisClient).Invalidate()

	after, err := searchService.Search(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	afterScore, ok := scoreForFood(after, created.FoodItemID)
	if !ok || beforeScore == afterScore {
		t.Fatalf("similarity was not recomputed: before=%v after=%v response=%+v", beforeScore, afterScore, after)
	}
	for _, item := range after.Items {
		if item.ID == created.FoodItemID && item.MacrosPer100 != merge.MacrosPer100 {
			t.Fatalf("merged macros=%+v want=%+v", item.MacrosPer100, merge.MacrosPer100)
		}
	}
}

func scoreForFood(response search.SearchResponse, id uuid.UUID) (float64, bool) {
	for index, item := range response.Items {
		if item.ID == id && index < len(response.SimilarityScores) {
			return response.SimilarityScores[index], true
		}
	}
	return 0, false
}

func assertImmutableReplay(t *testing.T, ctx context.Context, db *pgxpool.Pool, audit *repository.PostgresAdminImportAuditRepository, service *dataimporter.Service, adminID uuid.UUID, key string, req dataimporter.Request, original dataimporter.Result) {
	t.Helper()
	var auditsBefore int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=$1 AND action='import_food'`, original.FoodItemID).Scan(&auditsBefore); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `UPDATE food_items SET name=$2, physical_state='liquid', density_grams_per_milliliter=1, density_source_kind='manual' WHERE id=$1`, original.FoodItemID, original.Name+" mutated"); err != nil {
		t.Fatal(err)
	}
	assertReplayEquals(t, ctx, audit, service, adminID, key, req, original)
	if _, err := db.Exec(ctx, `UPDATE food_items SET deleted_at=now() WHERE id=$1`, original.FoodItemID); err != nil {
		t.Fatal(err)
	}
	assertReplayEquals(t, ctx, audit, service, adminID, key, req, original)
	assertCounts(t, ctx, db, original.FoodItemID, original.ImportID, 1, 1, auditsBefore)
}

func assertReplayEquals(t *testing.T, ctx context.Context, audit *repository.PostgresAdminImportAuditRepository, service *dataimporter.Service, adminID uuid.UUID, key string, req dataimporter.Request, original dataimporter.Result) {
	t.Helper()
	replay, err := confirm(t, ctx, audit, service, adminID, key, req, false)
	if err != nil || !replay.Replayed || replay.ImportID != original.ImportID || replay.FoodItemID != original.FoodItemID || replay.Name != original.Name || replay.PhysicalState != original.PhysicalState || replay.Merged != original.Merged {
		t.Fatalf("immutable replay=%+v original=%+v err=%v", replay, original, err)
	}
}

func assertConcurrentNameConfirmation(t *testing.T, ctx context.Context, db *pgxpool.Pool, audit *repository.PostgresAdminImportAuditRepository, service *dataimporter.Service, adminID uuid.UUID, confirmed bool) {
	t.Helper()
	suffix := "unconfirmed"
	if confirmed {
		suffix = "confirmed"
	}
	requests := []dataimporter.Request{
		{SourceProvider: "usda", ExternalID: "task-249-concurrent-" + suffix + "-a", ConfirmNameConflict: confirmed, Request: customitem.Request{Name: "Task 249 concurrent " + suffix, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}}},
		{SourceProvider: "openfoodfacts", ExternalID: "task-249-concurrent-" + suffix + "-b", ConfirmNameConflict: confirmed, Request: customitem.Request{Name: "  TASK 249 CONCURRENT " + suffix + "  ", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 11}, Micros: repository.MicroValues{}}},
	}
	type outcome struct {
		result dataimporter.Result
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(requests))
	var ready sync.WaitGroup
	ready.Add(len(requests))
	for _, request := range requests {
		go func(req dataimporter.Request) {
			ready.Done()
			<-start
			result, err := confirm(t, ctx, audit, service, adminID, "", req, false)
			outcomes <- outcome{result: result, err: err}
		}(request)
	}
	ready.Wait()
	close(start)
	got := []outcome{<-outcomes, <-outcomes}
	successes, confirmations, merges := 0, 0, 0
	for _, item := range got {
		switch {
		case item.err == nil:
			successes++
			if item.result.Merged {
				merges++
			}
		case errors.Is(item.err, dataimporter.ErrNameConfirmation):
			confirmations++
		default:
			t.Fatalf("concurrent %s outcome=%+v", suffix, item)
		}
	}
	wantSuccesses, wantConfirmations, wantMerges := 1, 1, 0
	if confirmed {
		wantSuccesses, wantConfirmations, wantMerges = 2, 0, 1
	}
	if successes != wantSuccesses || confirmations != wantConfirmations || merges != wantMerges {
		t.Fatalf("concurrent %s successes=%d confirmations=%d merges=%d", suffix, successes, confirmations, merges)
	}
	var foods, imports, audits int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM food_items WHERE normalized_name=lower(btrim($1))`, requests[0].Name).Scan(&foods); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM curated_imports WHERE external_id LIKE $1`, "task-249-concurrent-"+suffix+"-%").Scan(&imports); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM admin_audit_entries WHERE action='import_food' AND entity_id IN (SELECT id FROM food_items WHERE normalized_name=lower(btrim($1)))`, requests[0].Name).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if foods != 1 || imports != wantSuccesses || audits != wantSuccesses {
		t.Fatalf("concurrent %s foods=%d imports=%d audits=%d", suffix, foods, imports, audits)
	}
}

func confirm(t *testing.T, ctx context.Context, audit *repository.PostgresAdminImportAuditRepository, service *dataimporter.Service, adminID uuid.UUID, key string, req dataimporter.Request, badAudit bool) (result dataimporter.Result, err error) {
	t.Helper()
	err = audit.WithMutationAudit(ctx, repository.AdminAuditEntry{AdminUserID: adminID, Action: "import_food", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx repository.AdminMutationExecutor) (repository.AdminAuditChanges, error) {
		result, err = service.Confirm(ctx, tx, adminID, key, req)
		if err != nil {
			return repository.AdminAuditChanges{}, err
		}
		if result.Replayed {
			return repository.AdminAuditChanges{Replayed: true}, nil
		}
		after := []byte(`{"physicalState":"solid","status":"imported"}`)
		if result.PhysicalState == repository.PhysicalStateLiquid {
			after = []byte(`{"physicalState":"liquid","status":"imported"}`)
		}
		if badAudit {
			after = []byte(`{"providerPayload":"forbidden"}`)
		}
		return repository.AdminAuditChanges{EntityID: &result.FoodItemID, After: after}, nil
	})
	return result, err
}

func assertCounts(t *testing.T, ctx context.Context, db *pgxpool.Pool, foodID, importID uuid.UUID, foods, imports, audits int) {
	t.Helper()
	queries := []struct {
		sql  string
		args []any
		want int
	}{
		{`SELECT count(*) FROM food_items WHERE id=$1`, []any{foodID}, foods},
		{`SELECT count(*) FROM curated_imports WHERE id=$1`, []any{importID}, imports},
		{`SELECT count(*) FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=$1 AND action='import_food'`, []any{foodID}, audits},
	}
	for _, query := range queries {
		var got int
		if err := db.QueryRow(ctx, query.sql, query.args...).Scan(&got); err != nil || got != query.want {
			t.Fatalf("count query=%q got=%d want=%d err=%v", query.sql, got, query.want, err)
		}
	}
}

func assertNameAbsent(t *testing.T, ctx context.Context, db *pgxpool.Pool, name string) {
	t.Helper()
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM food_items WHERE normalized_name=lower(btrim($1))`, name).Scan(&count); err != nil || count != 0 {
		t.Fatalf("name=%q count=%d err=%v", name, count, err)
	}
}
