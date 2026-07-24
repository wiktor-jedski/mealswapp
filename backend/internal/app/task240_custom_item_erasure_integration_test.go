package app

// Implements DESIGN-008 AccountDeleter custom-item erasure integration verification.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/deletionworker"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

type task240FailingCachePurger struct{}

func (task240FailingCachePurger) PurgeUser(context.Context, uuid.UUID) error {
	return repository.NewError(repository.ErrorKindConnection, "cache unavailable", nil)
}

type task240FailureRecordRepository struct {
	repository.DeletionRequestRepository
	failed bool
}

type task240ExpiredAttemptPurger struct {
	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
}

func (p task240ExpiredAttemptPurger) PurgeUser(ctx context.Context, _ uuid.UUID) error {
	close(p.started)
	<-ctx.Done()
	close(p.canceled)
	<-p.release
	return ctx.Err()
}

func (r *task240FailureRecordRepository) RecordDeletionFailure(ctx context.Context, requestID uuid.UUID, leaseExpiresAt time.Time, category string, note string, nextAttemptAt *time.Time) error {
	if !r.failed {
		r.failed = true
		return repository.NewError(repository.ErrorKindConnection, "record deletion failure unavailable", nil)
	}
	return r.DeletionRequestRepository.RecordDeletionFailure(ctx, requestID, leaseExpiresAt, category, note, nextAttemptAt)
}

// TestTask240CustomItemErasureIntegration verifies IT-ARCH-009-004, ARCH-009,
// DESIGN-008 AccountDeleter, and SW-REQ-043/SW-REQ-072/SW-REQ-073 through
// real HTTP, PostgreSQL, Redis, export, authentication, and deletion collaborators.
func TestTask240CustomItemErasureIntegration(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}

	ownerEmail := "task-240-owner-" + uuid.NewString() + "@example.test"
	ownerCookies, ownerID := registerLiveDailyDietUser(t, server, cfg, ownerEmail)
	otherCookies, otherID := registerLiveDailyDietUser(t, server, cfg, "task-240-other-"+uuid.NewString()+"@example.test")
	ownerCSRF, ownerCookies := fetchLiveDailyDietCSRF(t, server, ownerCookies)
	otherCSRF, otherCookies := fetchLiveDailyDietCSRF(t, server, otherCookies)
	ownerItemID := task240CreateCustomItem(t, server, ownerCookies, ownerCSRF, "Owner private oats")
	ownerSecondItemID := task240CreateCustomItem(t, server, ownerCookies, ownerCSRF, "Owner private lentils")
	otherItemID := task240CreateCustomItem(t, server, otherCookies, otherCSRF, "Other private oats")
	globalID, err := repository.NewPostgresFoodItemRepository(db).Create(ctx, repository.FoodItemEntity{
		Name: "Task 240 curated oats", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 13, Carbohydrates: 68, Fat: 7},
	})
	if err != nil {
		t.Fatalf("create curated item: %v", err)
	}
	otherBefore := task240RowJSON(t, db, "custom_food_items", otherItemID)
	globalBefore := task240RowJSON(t, db, "food_items", globalID)

	ownerPrefix := "user:" + ownerID.String()
	otherCacheKey := "user:" + otherID.String() + ":custom-items"
	ownerCacheKeys := []string{ownerPrefix, ownerPrefix + ":custom-items", ownerPrefix + ":custom-items:" + ownerItemID.String()}
	for _, key := range append(append([]string{}, ownerCacheKeys...), otherCacheKey) {
		if err := redisClient.Set(ctx, key, "cached-private-data", time.Hour).Err(); err != nil {
			t.Fatalf("seed cache key %q: %v", key, err)
		}
	}
	t.Cleanup(func() { task240DeleteRedisKeys(redisClient, append(ownerCacheKeys, otherCacheKey)) })

	deleteResponse := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/account", "", ownerCookies, "", ownerCSRF)
	deleteEnvelope := decodeLiveDailyDietEnvelope(t, deleteResponse)
	deleteResponse.Body.Close()
	if deleteResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("request deletion status=%d body=%+v", deleteResponse.StatusCode, deleteEnvelope)
	}
	requestID := liveUUIDFromData(t, deleteEnvelope.Data, "requestId")

	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/custom-items", task240CustomItemBody("Blocked API write"), ownerCookies, "task-240-blocked-api", ownerCSRF)
	assertLiveDailyDietStatus(t, response, fiber.StatusUnauthorized)
	customRepo := repository.NewPostgresCustomFoodItemRepository(db)
	if _, err := customitem.NewService(customRepo).Create(ctx, ownerID, customitem.CreateRequest{
		Request: customitem.Request{
			Name: "Blocked claimed create", PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100: repository.MacroValues{}, Micros: repository.MicroValues{},
		},
		IdempotencyKey: "task-240-blocked-claim",
	}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("pending idempotent create error=%v, want blocked", err)
	}
	ownerItem, err := customRepo.GetByID(ctx, ownerID, ownerItemID, repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("load pending owner item: %v", err)
	}
	blockedCreate := repository.CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: repository.FoodItemEntity{
		Name: "Blocked direct create", PhysicalState: repository.PhysicalStateSolid, Micros: repository.MicroValues{},
	}}
	if _, err := customRepo.Create(ctx, blockedCreate); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("pending direct create error=%v, want blocked", err)
	}
	ownerItem.Name = "Blocked direct update"
	if err := customRepo.Update(ctx, ownerItem); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("pending direct update error=%v, want blocked", err)
	}
	if err := customRepo.Delete(ctx, ownerID, ownerSecondItemID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("pending direct delete error=%v, want blocked", err)
	}
	if got := task240CountCustomItems(t, db, ownerID); got != 2 {
		t.Fatalf("pending write lockout left %d owner items, want 2", got)
	}

	requests := repository.NewPostgresComplianceRepository(db)
	sessions := repository.NewPostgresSessionRepository(db)
	accounts := repository.NewPostgresEncryptedIdentityRepository(db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	firstAttempt := userdata.NewAccountDeletionService(requests, sessions, accounts, task240FailingCachePurger{})
	claimed, err := firstAttempt.ProcessDueDeletionRequests(ctx, now, 10)
	if err != nil || len(claimed) != 1 || claimed[0].ID != requestID {
		t.Fatalf("first deletion attempt claimed=%+v err=%v", claimed, err)
	}
	var failedStatus string
	var failedUserID *uuid.UUID
	var retryCount int
	var nextAttemptAt *time.Time
	if err := db.QueryRow(ctx, `SELECT status, user_id, retry_count, next_attempt_at FROM data_deletion_requests WHERE id = $1`, requestID).Scan(&failedStatus, &failedUserID, &retryCount, &nextAttemptAt); err != nil {
		t.Fatalf("load failed deletion request: %v", err)
	}
	if failedStatus != "failed" || failedUserID == nil || *failedUserID != ownerID || retryCount != 1 || nextAttemptAt == nil || !nextAttemptAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("retry state status=%s user=%v count=%d next=%v", failedStatus, failedUserID, retryCount, nextAttemptAt)
	}
	if got := task240CountCustomItems(t, db, ownerID); got != 0 {
		t.Fatalf("transactional account cleanup left %d owner custom items", got)
	}
	for _, key := range ownerCacheKeys {
		if exists := redisClient.Exists(ctx, key).Val(); exists != 1 {
			t.Fatalf("failed cache attempt unexpectedly removed %q", key)
		}
	}

	retry := userdata.NewAccountDeletionService(requests, sessions, accounts, redisCachePurger{client: redisClient})
	claimed, err = retry.ProcessDueDeletionRequests(ctx, now.Add(2*time.Minute), 10)
	if err != nil || len(claimed) != 1 || claimed[0].ID != requestID {
		t.Fatalf("retry deletion attempt claimed=%+v err=%v", claimed, err)
	}
	claimed, err = retry.ProcessDueDeletionRequests(ctx, now.Add(3*time.Minute), 10)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("completed deletion was reclaimed: claimed=%+v err=%v", claimed, err)
	}
	for _, key := range ownerCacheKeys {
		if exists := redisClient.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("completed deletion retained cache key %q", key)
		}
	}
	if exists := redisClient.Exists(ctx, otherCacheKey).Val(); exists != 1 {
		t.Fatal("completed deletion removed another user's cache key")
	}
	if got := task240RowJSON(t, db, "custom_food_items", otherItemID); got != otherBefore {
		t.Fatalf("cross-user custom row changed\nbefore=%s\nafter=%s", otherBefore, got)
	}
	if got := task240RowJSON(t, db, "food_items", globalID); got != globalBefore {
		t.Fatalf("global curated row changed\nbefore=%s\nafter=%s", globalBefore, got)
	}

	receipt := task240DeletionReceiptJSON(t, db, requestID)
	for _, forbidden := range []string{ownerID.String(), ownerItemID.String(), ownerSecondItemID.String(), "Owner private", "owner_id", "custom_food"} {
		if strings.Contains(receipt, forbidden) {
			t.Fatalf("pseudonymous receipt contains forbidden owner/custom-item data %q: %s", forbidden, receipt)
		}
	}
	var receiptUserID *uuid.UUID
	var receiptID *uuid.UUID
	var receiptStatus string
	if err := db.QueryRow(ctx, `SELECT user_id, receipt_id, status FROM data_deletion_requests WHERE id = $1`, requestID).Scan(&receiptUserID, &receiptID, &receiptStatus); err != nil {
		t.Fatalf("load completed receipt: %v", err)
	}
	if receiptUserID != nil || receiptID == nil || *receiptID == uuid.Nil || receiptStatus != "completed" {
		t.Fatalf("completed receipt user=%v receipt=%v status=%s", receiptUserID, receiptID, receiptStatus)
	}

	for _, access := range []struct{ method, path string }{
		{fiber.MethodGet, "/api/v1/profile"},
		{fiber.MethodGet, "/api/v1/account/export?format=json"},
		{fiber.MethodGet, "/api/v1/custom-items/" + ownerItemID.String()},
	} {
		response = liveDailyDietRequest(t, server, access.method, access.path, "", ownerCookies, "", "")
		assertLiveDailyDietStatus(t, response, fiber.StatusUnauthorized)
	}
	loginBody := fmt.Sprintf(`{"email":%q,"password":"StrongerPassword1!"}`, ownerEmail)
	response = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/auth/login", loginBody, nil, "", "")
	assertLiveDailyDietStatus(t, response, fiber.StatusUnauthorized)
}

// TestTask240MigrationUpgradeBackfillsDeletionLockout proves migration 26
// repairs accounts whose deletion workflow predates the durable user marker.
func TestTask240MigrationUpgradeBackfillsDeletionLockout(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	pendingCookies, pendingID := registerLiveDailyDietUser(t, server, cfg, "task-240-upgrade-pending-"+uuid.NewString()+"@example.test")
	processingCookies, processingID := registerLiveDailyDietUser(t, server, cfg, "task-240-upgrade-processing-"+uuid.NewString()+"@example.test")
	pendingCSRF, pendingCookies := fetchLiveDailyDietCSRF(t, server, pendingCookies)
	processingCSRF, processingCookies := fetchLiveDailyDietCSRF(t, server, processingCookies)
	pendingItemID := task240CreateCustomItem(t, server, pendingCookies, pendingCSRF, "Upgrade pending item")
	processingItemID := task240CreateCustomItem(t, server, processingCookies, processingCSRF, "Upgrade processing item")

	task240ExecMigration(t, db, "000026_custom_item_erasure_integration.down.sql")
	requestedAt := time.Date(2026, 7, 20, 9, 30, 0, 0, time.UTC)
	if _, err := db.Exec(ctx, `INSERT INTO data_deletion_requests (user_id, status, requested_at) VALUES ($1, 'pending', $3), ($2, 'processing', $3)`, pendingID, processingID, requestedAt); err != nil {
		t.Fatalf("insert pre-migration deletion requests: %v", err)
	}
	task240ExecMigration(t, db, "000026_custom_item_erasure_integration.up.sql")

	for _, userID := range []uuid.UUID{pendingID, processingID} {
		var marker *time.Time
		if err := db.QueryRow(ctx, `SELECT deletion_requested_at FROM users WHERE id = $1`, userID).Scan(&marker); err != nil || marker == nil || !marker.Equal(requestedAt) {
			t.Fatalf("backfilled marker user=%s marker=%v err=%v", userID, marker, err)
		}
	}
	repo := repository.NewPostgresCustomFoodItemRepository(db)
	if _, err := repo.Create(ctx, repository.CustomFoodItemEntity{OwnerID: pendingID, FoodItemEntity: repository.FoodItemEntity{Name: "Blocked upgraded create", PhysicalState: repository.PhysicalStateSolid}}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("upgraded pending create error=%v, want blocked", err)
	}
	pendingItem, err := repo.GetByID(ctx, pendingID, pendingItemID, repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("load upgraded pending item: %v", err)
	}
	pendingItem.Name = "Blocked upgraded update"
	if err := repo.Update(ctx, pendingItem); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("upgraded pending update error=%v, want blocked", err)
	}
	if err := repo.Delete(ctx, processingID, processingItemID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("upgraded processing delete error=%v, want blocked", err)
	}
	if task240CountCustomItems(t, db, pendingID) != 1 || task240CountCustomItems(t, db, processingID) != 1 {
		t.Fatal("upgrade lockout mutated pre-existing custom items")
	}
}

// TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker proves the
// production write lock and DELETE /account marker update cannot pass each other.
func TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	cookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-240-lock-"+uuid.NewString()+"@example.test")
	csrf, cookies := fetchLiveDailyDietCSRF(t, server, cookies)

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin custom write transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	writeRepo := repository.NewPostgresCustomFoodItemRepository(tx)
	writtenID, err := writeRepo.Create(ctx, repository.CustomFoodItemEntity{OwnerID: userID, FoodItemEntity: repository.FoodItemEntity{
		Name: "Concurrent write", PhysicalState: repository.PhysicalStateSolid, Micros: repository.MicroValues{},
	}})
	if err != nil {
		t.Fatalf("create held custom item: %v", err)
	}

	type deletionResult struct {
		response *http.Response
		err      error
	}
	deleted := make(chan deletionResult, 1)
	go func() {
		request := httptest.NewRequest(fiber.MethodDelete, "/api/v1/account", nil)
		request.Header.Set("X-CSRF-Token", csrf)
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response, requestErr := server.Test(request, -1)
		deleted <- deletionResult{response: response, err: requestErr}
	}()

	select {
	case result := <-deleted:
		if result.response != nil {
			result.response.Body.Close()
		}
		t.Fatalf("DELETE /account passed active custom write lock: %v", result.err)
	case <-time.After(100 * time.Millisecond):
	}
	var marker *time.Time
	if err := db.QueryRow(ctx, `SELECT deletion_requested_at FROM users WHERE id = $1`, userID).Scan(&marker); err != nil || marker != nil {
		t.Fatalf("marker while write is active=%v err=%v", marker, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit custom write: %v", err)
	}
	result := <-deleted
	if result.err != nil {
		t.Fatalf("DELETE /account after write commit: %v", result.err)
	}
	defer result.response.Body.Close()
	if result.response.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(result.response.Body)
		t.Fatalf("DELETE /account status=%d body=%s", result.response.StatusCode, body)
	}
	if err := db.QueryRow(ctx, `SELECT deletion_requested_at FROM users WHERE id = $1`, userID).Scan(&marker); err != nil || marker == nil {
		t.Fatalf("marker after deletion request=%v err=%v", marker, err)
	}
	if task240CountCustomItems(t, db, userID) != 1 {
		t.Fatalf("held custom item %s did not commit before marker", writtenID)
	}
	if _, err := repository.NewPostgresCustomFoodItemRepository(db).Create(ctx, repository.CustomFoodItemEntity{OwnerID: userID, FoodItemEntity: repository.FoodItemEntity{
		Name: "After marker", PhysicalState: repository.PhysicalStateSolid, Micros: repository.MicroValues{},
	}}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("post-marker concurrent write error=%v, want blocked", err)
	}
}

// TestTask240APIToProductionDeletionWorker proves the deployed scheduler consumes
// a request created by DELETE /account without a manual ProcessDue call.
func TestTask240APIToProductionDeletionWorker(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	cookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-240-worker-"+uuid.NewString()+"@example.test")
	csrf, cookies := fetchLiveDailyDietCSRF(t, server, cookies)
	itemID := task240CreateCustomItem(t, server, cookies, csrf, "Worker-erased item")
	cacheKey := "user:" + userID.String() + ":custom-items:" + itemID.String()
	if err := redisClient.Set(ctx, cacheKey, "private", time.Hour).Err(); err != nil {
		t.Fatalf("seed worker cache: %v", err)
	}
	t.Cleanup(func() { task240DeleteRedisKeys(redisClient, []string{cacheKey}) })

	response := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/account", "", cookies, "", csrf)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("request deletion status=%d body=%+v", response.StatusCode, envelope)
	}
	requestID := liveUUIDFromData(t, envelope.Data, "requestId")
	service := userdata.NewAccountDeletionService(
		repository.NewPostgresComplianceRepository(db), repository.NewPostgresSessionRepository(db),
		repository.NewPostgresEncryptedIdentityRepository(db), cache.NewUserPurger(redisClient),
	)
	workerCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- deletionworker.RunAccountDeletionProcessor(workerCtx, service, 10*time.Millisecond, 10, &observability.MemorySink{})
	}()
	task240WaitForDeletionCompletion(t, db, requestID)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("deletion worker shutdown error = %v", err)
	}
	if got := task240CountCustomItems(t, db, userID); got != 0 {
		t.Fatalf("production worker left %d custom items", got)
	}
	if redisClient.Exists(ctx, cacheKey).Val() != 0 {
		t.Fatal("production worker retained custom-item cache")
	}
}

// TestTask240ProcessingLeaseRecoversFailureRecordOutage proves a failed attempt
// cannot remain stranded in processing when failure-state persistence also fails.
func TestTask240ProcessingLeaseRecoversFailureRecordOutage(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	cookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-240-recovery-"+uuid.NewString()+"@example.test")
	csrf, cookies := fetchLiveDailyDietCSRF(t, server, cookies)
	itemID := task240CreateCustomItem(t, server, cookies, csrf, "Recovery item")
	cacheKey := "user:" + userID.String() + ":custom-items:" + itemID.String()
	if err := redisClient.Set(ctx, cacheKey, "private", time.Hour).Err(); err != nil {
		t.Fatalf("seed recovery cache: %v", err)
	}
	t.Cleanup(func() { task240DeleteRedisKeys(redisClient, []string{cacheKey}) })
	response := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/account", "", cookies, "", csrf)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	requestID := liveUUIDFromData(t, envelope.Data, "requestId")

	realRequests := repository.NewPostgresComplianceRepository(db)
	flakyRequests := &task240FailureRecordRepository{DeletionRequestRepository: realRequests}
	now := time.Now().UTC().Truncate(time.Microsecond)
	failing := userdata.NewAccountDeletionService(flakyRequests, repository.NewPostgresSessionRepository(db), repository.NewPostgresEncryptedIdentityRepository(db), task240FailingCachePurger{})
	if _, err := failing.ProcessDueDeletionRequests(ctx, now, 10); !repository.IsKind(err, repository.ErrorKindConnection) {
		t.Fatalf("failure-record outage error=%v, want connection", err)
	}
	var status string
	var lease *time.Time
	if err := db.QueryRow(ctx, `SELECT status, next_attempt_at FROM data_deletion_requests WHERE id = $1`, requestID).Scan(&status, &lease); err != nil {
		t.Fatalf("load stranded request: %v", err)
	}
	if status != "processing" || lease == nil || !lease.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("processing recovery lease status=%s lease=%v", status, lease)
	}
	if claimed, err := realRequests.ClaimDeletionRequests(ctx, now.Add(5*time.Minute-time.Nanosecond), 10); err != nil || len(claimed) != 0 {
		t.Fatalf("processing request reclaimed before lease: claimed=%+v err=%v", claimed, err)
	}
	recovery := userdata.NewAccountDeletionService(realRequests, repository.NewPostgresSessionRepository(db), repository.NewPostgresEncryptedIdentityRepository(db), cache.NewUserPurger(redisClient))
	claimed, err := recovery.ProcessDueDeletionRequests(ctx, now.Add(5*time.Minute), 10)
	if err != nil || len(claimed) != 1 || claimed[0].ID != requestID {
		t.Fatalf("processing lease recovery claimed=%+v err=%v", claimed, err)
	}
	task240AssertDeletionCompleted(t, db, requestID)
	if redisClient.Exists(ctx, cacheKey).Val() != 0 {
		t.Fatal("recovery retained custom-item cache")
	}
}

// TestTask240ExpiredAttemptCannotFinalizeReclaimedWork proves a timed-out
// processor cannot overwrite the completion owned by a subsequent worker.
func TestTask240ExpiredAttemptCannotFinalizeReclaimedWork(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	cookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-240-two-workers-"+uuid.NewString()+"@example.test")
	csrf, cookies := fetchLiveDailyDietCSRF(t, server, cookies)
	request := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/account", "", cookies, "", csrf)
	envelope := decodeLiveDailyDietEnvelope(t, request)
	request.Body.Close()
	if request.StatusCode != fiber.StatusOK {
		t.Fatalf("request deletion status=%d envelope=%+v", request.StatusCode, envelope)
	}
	requestID := liveUUIDFromData(t, envelope.Data, "requestId")

	requests := repository.NewPostgresComplianceRepository(db).WithDeletionLeaseDuration(500 * time.Millisecond)
	blocked := task240ExpiredAttemptPurger{started: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{})}
	released := false
	t.Cleanup(func() {
		if !released {
			close(blocked.release)
		}
	})
	first := userdata.NewAccountDeletionService(requests, repository.NewPostgresSessionRepository(db), repository.NewPostgresEncryptedIdentityRepository(db), blocked)
	firstNow := time.Now().UTC()
	firstDone := make(chan error, 1)
	go func() {
		_, processErr := first.ProcessDueDeletionRequests(ctx, firstNow, 1)
		firstDone <- processErr
	}()
	select {
	case <-blocked.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first deletion worker did not reach cache purge")
	}

	second := userdata.NewAccountDeletionService(requests, repository.NewPostgresSessionRepository(db), repository.NewPostgresEncryptedIdentityRepository(db), cache.NewUserPurger(redisClient))
	claimed, err := second.ProcessDueDeletionRequests(ctx, firstNow.Add(250*time.Millisecond), 1)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("second worker claimed active lease: claimed=%+v err=%v", claimed, err)
	}
	select {
	case <-blocked.canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("first deletion worker did not stop at its lease deadline")
	}
	claimed, err = second.ProcessDueDeletionRequests(ctx, time.Now().UTC(), 1)
	if err != nil || len(claimed) != 1 || claimed[0].ID != requestID {
		t.Fatalf("second worker reclaim: claimed=%+v err=%v", claimed, err)
	}
	close(blocked.release)
	released = true
	if err := <-firstDone; !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("expired worker finalization error=%v, want guarded not found", err)
	}
	task240AssertDeletionCompleted(t, db, requestID)
	if task240CountCustomItems(t, db, userID) != 0 {
		t.Fatal("two-worker completion retained owner custom items")
	}
}

func task240CreateCustomItem(t *testing.T, server *fiber.App, cookies []*http.Cookie, csrf, name string) uuid.UUID {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/custom-items", task240CustomItemBody(name), cookies, "task-240-"+uuid.NewString(), csrf)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create custom item %q status=%d body=%+v", name, response.StatusCode, envelope)
	}
	return liveUUIDFromData(t, envelope.Data, "id")
}

func task240CustomItemBody(name string) string {
	return fmt.Sprintf(`{"name":%q,"physicalState":"solid","prepTimeMinutes":0,"macrosPer100":{"protein":10,"carbohydrates":20,"fat":5},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`, name)
}

func task240CountCustomItems(t *testing.T, db *pgxpool.Pool, ownerID uuid.UUID) int {
	t.Helper()
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM custom_food_items WHERE owner_id = $1`, ownerID).Scan(&count); err != nil {
		t.Fatalf("count custom items: %v", err)
	}
	return count
}

func task240RowJSON(t *testing.T, db *pgxpool.Pool, table string, id uuid.UUID) string {
	t.Helper()
	if table != "custom_food_items" && table != "food_items" {
		t.Fatalf("unsupported fixture table %q", table)
	}
	var value string
	query := "SELECT to_jsonb(row_data)::text FROM (SELECT * FROM " + table + " WHERE id = $1) AS row_data"
	if err := db.QueryRow(context.Background(), query, id).Scan(&value); err != nil {
		t.Fatalf("load %s row %s: %v", table, id, err)
	}
	return value
}

func task240DeletionReceiptJSON(t *testing.T, db *pgxpool.Pool, requestID uuid.UUID) string {
	t.Helper()
	var value string
	if err := db.QueryRow(context.Background(), `SELECT to_jsonb(request_row)::text FROM (SELECT * FROM data_deletion_requests WHERE id = $1) AS request_row`, requestID).Scan(&value); err != nil {
		t.Fatalf("load deletion receipt JSON: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(value), &document); err != nil {
		t.Fatalf("decode deletion receipt JSON: %v", err)
	}
	return value
}

func task240DeleteRedisKeys(client *redis.Client, keys []string) {
	if client != nil && len(keys) > 0 {
		_ = client.Del(context.Background(), keys...).Err()
	}
}

func task240ExecMigration(t *testing.T, db *pgxpool.Pool, name string) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("../../../database/migrations", name))
	if err != nil {
		t.Fatalf("resolve migration %s: %v", name, err)
	}
	sql, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := db.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("execute migration %s: %v", name, err)
	}
}

func task240WaitForDeletionCompletion(t *testing.T, db *pgxpool.Pool, requestID uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		if err := db.QueryRow(context.Background(), `SELECT status FROM data_deletion_requests WHERE id = $1`, requestID).Scan(&status); err == nil && status == "completed" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("deletion request %s did not complete", requestID)
}

func task240AssertDeletionCompleted(t *testing.T, db *pgxpool.Pool, requestID uuid.UUID) {
	t.Helper()
	var status string
	var userID *uuid.UUID
	if err := db.QueryRow(context.Background(), `SELECT status, user_id FROM data_deletion_requests WHERE id = $1`, requestID).Scan(&status, &userID); err != nil {
		t.Fatalf("load completed deletion %s: %v", requestID, err)
	}
	if status != "completed" || userID != nil {
		t.Fatalf("deletion request %s status=%s user=%v", requestID, status, userID)
	}
}
