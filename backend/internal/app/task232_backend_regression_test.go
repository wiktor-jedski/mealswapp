package app

// Implements DESIGN-004 JobStatusTracker Task 232 backend functional regression gate.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers verifies
// IT-ARCH-004-001, IT-ARCH-004-002, IT-ARCH-004-003, IT-ARCH-004-004,
// IT-ARCH-004-005, and IT-ARCH-004-007, ARCH-004, DESIGN-004, DESIGN-008,
// DESIGN-014, and SW-REQ-006/021/022/023/030/080/082 across live PostgreSQL,
// Redis, authenticated API, queue, worker, and packaged-CLP boundaries.
func TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers(t *testing.T) {
	clpPath, clpVersion := task206CLP(t)
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	task232ResetRedis(t, redisClient)
	t.Cleanup(func() { task232ResetRedis(t, redisClient) })

	cfg := liveDailyDietAPIConfig()
	cfg.CLPExecutable, cfg.CLPVersion = clpPath, clpVersion
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	mealIDs := createTask206Meals(t, db)
	ownerCookies, ownerID := registerLiveDailyDietUser(t, server, cfg, "task-232-owner-"+uuid.NewString()+"@example.test")
	otherCookies, otherID := registerLiveDailyDietUser(t, server, cfg, "task-232-other-"+uuid.NewString()+"@example.test")
	ownerCSRF, ownerCookies := fetchLiveDailyDietCSRF(t, server, ownerCookies)
	otherCSRF, otherCookies := fetchLiveDailyDietCSRF(t, server, otherCookies)
	grantTask206Trial(t, db, ownerID)

	// A Daily Diet create replay is the immutable original 201 projection even
	// after the current aggregate is replaced and deleted.
	dailyDietBody := liveDailyDietBody("Task 232 immutable diet", mealIDs[0], mealIDs[1])
	createKey := "task-232-daily-diet-" + uuid.NewString()
	created := task232Call(t, server, fiber.MethodPost, "/api/v1/daily-diets", dailyDietBody, ownerCookies, createKey, ownerCSRF)
	task232AssertSuccess(t, created, fiber.StatusCreated, "ok")
	dietToDelete := liveUUIDFromData(t, created.Envelope.Data, "id")
	replaced := task232Call(t, server, fiber.MethodPut, "/api/v1/daily-diets/"+dietToDelete.String(), liveDailyDietBody("Task 232 replacement", mealIDs[2], mealIDs[3]), ownerCookies, "", ownerCSRF)
	task232AssertSuccess(t, replaced, fiber.StatusOK, "ok")
	deleted := task232Call(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietToDelete.String(), "", ownerCookies, "", ownerCSRF)
	if deleted.StatusCode != fiber.StatusNoContent || len(deleted.Raw) != 0 {
		t.Fatalf("delete response = %d %q, want exact empty 204", deleted.StatusCode, deleted.Raw)
	}
	replayedDiet := task232Call(t, server, fiber.MethodPost, "/api/v1/daily-diets", dailyDietBody, ownerCookies, createKey, ownerCSRF)
	task232AssertSuccess(t, replayedDiet, fiber.StatusCreated, "ok")
	if !reflect.DeepEqual(replayedDiet.Envelope.Data, created.Envelope.Data) || countLiveSavedDiets(t, db, ownerID) != 0 {
		t.Fatalf("immutable Daily Diet replay data=%+v want=%+v persisted=%d", replayedDiet.Envelope.Data, created.Envelope.Data, countLiveSavedDiets(t, db, ownerID))
	}

	ownerDietID := createTask206SavedDiet(t, server, ownerCookies, ownerCSRF, mealIDs[0])
	otherDietID := createTask206SavedDiet(t, server, otherCookies, otherCSRF, mealIDs[1])
	normalizedKey := "task-232-normalized-" + uuid.NewString()
	firstBody := task232OptimizationBody(ownerDietID, 0, []uuid.UUID{mealIDs[4], mealIDs[3]})
	normalizedBody := fmt.Sprintf("{\n\t\"excludedMealIds\":[%q,%q],\n\t\"tolerancePercent\":-0.0,\n\t\"dailyDietId\":%q\n}", mealIDs[3], mealIDs[4], ownerDietID)
	first := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", firstBody, ownerCookies, normalizedKey, ownerCSRF)
	task232AssertAcknowledgement(t, first)
	replay := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", normalizedBody, ownerCookies, normalizedKey, ownerCSRF)
	task232AssertAcknowledgement(t, replay)
	if !reflect.DeepEqual(first.Envelope.Data, replay.Envelope.Data) || first.Header.Get(fiber.HeaderLocation) != replay.Header.Get(fiber.HeaderLocation) {
		t.Fatalf("normalized replay changed acknowledgement: first=%+v replay=%+v", first.Envelope.Data, replay.Envelope.Data)
	}
	if got := task232StreamLength(t, redisClient); got != 1 {
		t.Fatalf("normalized replay stream entries = %d, want 1", got)
	}
	conflict := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", task232OptimizationBody(ownerDietID, 0.1, []uuid.UUID{mealIDs[3], mealIDs[4]}), ownerCookies, normalizedKey, ownerCSRF)
	task232AssertError(t, conflict, fiber.StatusConflict, "validation", "idempotency_key_conflict", false)

	freeDenied := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", task232OptimizationBody(otherDietID, 0, nil), otherCookies, "task-232-free-"+uuid.NewString(), otherCSRF)
	task232AssertError(t, freeDenied, fiber.StatusForbidden, "entitlement", "entitlement_denied", false)
	grantTask206Trial(t, db, otherID)
	crossOwner := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", task232OptimizationBody(ownerDietID, 0, nil), otherCookies, "task-232-owner-isolation-"+uuid.NewString(), otherCSRF)
	task232AssertError(t, crossOwner, fiber.StatusNotFound, "validation", "not_found", false)

	ctx, cancel := context.WithCancel(context.Background())
	workerDone := startTask206Worker(t, ctx, cfg, db, redisClient)
	normalizedJobID := liveUUIDFromData(t, first.Envelope.Data, "jobId")
	crossPoll := task232Call(t, server, fiber.MethodGet, "/api/v1/optimization/jobs/"+normalizedJobID.String(), "", otherCookies, "", "")
	task232AssertError(t, crossPoll, fiber.StatusNotFound, "validation", "not_found", false)
	normalizedResult := waitTask206Job(t, server, normalizedJobID, ownerCookies, workerDone)
	assertTask206Alternatives(t, normalizedResult.Envelope, mealIDs, 20, 30, 10, []uuid.UUID{mealIDs[3], mealIDs[4]})

	// A pending PostgreSQL acknowledgement models failure after durable claim
	// but before queue publication. Repair must re-check entitlement/ownership,
	// retain the server job ID, and publish exactly once without rate recounting.
	repairKey := "task-232-repair-" + uuid.NewString()
	repairJobID := uuid.New()
	repairBody := task232OptimizationBody(ownerDietID, 0, []uuid.UUID{mealIDs[2], mealIDs[1]})
	task232StorePendingOptimization(t, db, ownerID, repairJobID, ownerDietID, repairKey, 0, []uuid.UUID{mealIDs[1], mealIDs[2]})
	task232AppendEntitlement(t, db, ownerID, "expired", time.Now().UTC().Add(-time.Hour))
	deniedRepair := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", repairBody, ownerCookies, repairKey, ownerCSRF)
	task232AssertError(t, deniedRepair, fiber.StatusForbidden, "entitlement", "entitlement_denied", false)
	if got := task232StreamLength(t, redisClient); got != 0 {
		t.Fatalf("denied repair published %d stream entries", got)
	}
	task232AppendEntitlement(t, db, ownerID, "active", time.Now().UTC().Add(time.Hour))
	repaired := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", repairBody, ownerCookies, repairKey, ownerCSRF)
	task232AssertAcknowledgement(t, repaired)
	if liveUUIDFromData(t, repaired.Envelope.Data, "jobId") != repairJobID || task232StreamLength(t, redisClient) != 1 {
		t.Fatalf("repair acknowledgement=%+v stream=%d, want original job and one publication", repaired.Envelope.Data, task232StreamLength(t, redisClient))
	}
	repairedReplay := task232Call(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", task232OptimizationBody(ownerDietID, -0.0, []uuid.UUID{mealIDs[1], mealIDs[2]}), ownerCookies, repairKey, ownerCSRF)
	task232AssertAcknowledgement(t, repairedReplay)
	if !reflect.DeepEqual(repaired.Envelope.Data, repairedReplay.Envelope.Data) || task232StreamLength(t, redisClient) != 1 {
		t.Fatalf("post-repair replay changed data or republished: %+v / %+v", repaired.Envelope.Data, repairedReplay.Envelope.Data)
	}
	repairedResult := waitTask206Job(t, server, repairJobID, ownerCookies, workerDone)
	assertTask206Alternatives(t, repairedResult.Envelope, mealIDs, 20, 30, 10, []uuid.UUID{mealIDs[1], mealIDs[2]})

	// Unrelated authenticated users may submit concurrently and receive
	// independent jobs while the API remains asynchronous.
	concurrent := task232SubmitConcurrentUsers(t, server, []task232ConcurrentRequest{
		{Body: task232OptimizationBody(ownerDietID, 0, nil), Cookies: ownerCookies, CSRF: ownerCSRF, Key: "task-232-owner-concurrent-" + uuid.NewString()},
		{Body: task232OptimizationBody(otherDietID, 0, nil), Cookies: otherCookies, CSRF: otherCSRF, Key: "task-232-other-concurrent-" + uuid.NewString()},
	})
	jobIDs := make([]uuid.UUID, 0, len(concurrent))
	for _, response := range concurrent {
		task232AssertAcknowledgement(t, response)
		jobIDs = append(jobIDs, liveUUIDFromData(t, response.Envelope.Data, "jobId"))
	}
	if jobIDs[0] == jobIDs[1] {
		t.Fatalf("concurrent users received duplicate job ID %s", jobIDs[0])
	}
	for index, jobID := range jobIDs {
		cookies := ownerCookies
		if index == 1 {
			cookies = otherCookies
		}
		result := waitTask206Job(t, server, jobID, cookies, workerDone)
		assertTask206Alternatives(t, result.Envelope, mealIDs, 20, 30, 10, nil)
	}

	// A dead Redis endpoint is a safe, retryable 503 and never falls back to
	// solver execution inside the API process.
	outageRedis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:63999", DialTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond, WriteTimeout: 50 * time.Millisecond, MaxRetries: 0})
	t.Cleanup(func() { _ = outageRedis.Close() })
	outageServer, err := NewProduction(cfg, db, outageRedis, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() outage error = %v", err)
	}
	outageCSRF, outageCookies := fetchLiveDailyDietCSRF(t, outageServer, ownerCookies)
	outage := task232Call(t, outageServer, fiber.MethodPost, "/api/v1/optimization/jobs", task232OptimizationBody(ownerDietID, 0, nil), outageCookies, "task-232-outage-"+uuid.NewString(), outageCSRF)
	task232AssertError(t, outage, fiber.StatusServiceUnavailable, "dependency", "queue_unavailable", true)

	cancel()
	if err := task206WorkerResult(workerDone); err != nil {
		t.Fatalf("worker shutdown error = %v", err)
	}
}

type task232Response struct {
	StatusCode int
	Header     http.Header
	Envelope   httpapi.Envelope
	Raw        []byte
}

func task232Call(t *testing.T, server *fiber.App, method, path, body string, cookies []*http.Cookie, key, csrf string) task232Response {
	t.Helper()
	response := liveDailyDietRequest(t, server, method, path, body, cookies, key, csrf)
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	result := task232Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Raw: raw}
	if len(raw) > 0 && json.Unmarshal(raw, &result.Envelope) != nil {
		t.Fatalf("decode %s %s response: %s", method, path, raw)
	}
	return result
}

func task232AssertSuccess(t *testing.T, response task232Response, status int, envelopeStatus string) {
	t.Helper()
	if response.StatusCode != status || response.Envelope.Status != envelopeStatus || response.Envelope.RequestID == "" || response.Envelope.Error != nil || response.Envelope.Data == nil {
		t.Fatalf("success envelope = %d %+v", response.StatusCode, response.Envelope)
	}
	task232AssertJSONKeys(t, response.Raw, "status", "requestId", "data")
}

func task232AssertAcknowledgement(t *testing.T, response task232Response) {
	t.Helper()
	task232AssertSuccess(t, response, fiber.StatusAccepted, "accepted")
	if len(response.Envelope.Data) != 3 || response.Envelope.Data["status"] != "queued" {
		t.Fatalf("acknowledgement data = %+v", response.Envelope.Data)
	}
	jobID := liveUUIDFromData(t, response.Envelope.Data, "jobId")
	wantPoll := "/api/v1/optimization/jobs/" + jobID.String()
	if response.Envelope.Data["pollUrl"] != wantPoll || response.Header.Get(fiber.HeaderLocation) != wantPoll {
		t.Fatalf("acknowledgement poll=%v location=%q want=%q", response.Envelope.Data["pollUrl"], response.Header.Get(fiber.HeaderLocation), wantPoll)
	}
}

func task232AssertError(t *testing.T, response task232Response, status int, category, code string, retryable bool) {
	t.Helper()
	errorData := response.Envelope.Error
	wantMessages := map[string]string{
		"idempotency_key_conflict": "Idempotency-Key was already used with a different request body",
		"entitlement_denied":       "an active trial or paid subscription is required for optimization",
		"not_found":                "optimization job not found",
		"queue_unavailable":        "optimization queue is unavailable",
	}
	if response.StatusCode != status || response.Envelope.Status != "error" || response.Envelope.RequestID == "" || response.Envelope.Data != nil || errorData == nil || errorData.Category != category || errorData.Code != code || errorData.Message != wantMessages[code] || errorData.Retryable != retryable || errorData.RequestID != response.Envelope.RequestID {
		t.Fatalf("error envelope = %d %+v", response.StatusCode, response.Envelope)
	}
	task232AssertJSONKeys(t, response.Raw, "status", "requestId", "error")
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(response.Raw, &raw); err != nil {
		t.Fatal(err)
	}
	var rawError map[string]json.RawMessage
	if err := json.Unmarshal(raw["error"], &rawError); err != nil {
		t.Fatal(err)
	}
	task232AssertKeySet(t, rawError, "category", "code", "message", "retryable", "requestId")
}

func task232AssertJSONKeys(t *testing.T, raw []byte, keys ...string) {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode exact envelope: %v", err)
	}
	task232AssertKeySet(t, value, keys...)
}

func task232AssertKeySet(t *testing.T, value map[string]json.RawMessage, keys ...string) {
	t.Helper()
	if len(value) != len(keys) {
		t.Fatalf("JSON keys = %v, want %v", task232SortedKeys(value), keys)
	}
	for _, key := range keys {
		if _, found := value[key]; !found {
			t.Fatalf("JSON keys = %v, missing %q", task232SortedKeys(value), key)
		}
	}
}

func task232SortedKeys(value map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func task232OptimizationBody(dietID uuid.UUID, tolerance float64, excluded []uuid.UUID) string {
	ids := make([]string, 0, len(excluded))
	for _, id := range excluded {
		ids = append(ids, fmt.Sprintf("%q", id))
	}
	return fmt.Sprintf(`{"dailyDietId":%q,"tolerancePercent":%v,"excludedMealIds":[%s]}`, dietID, tolerance, strings.Join(ids, ","))
}

func task232StorePendingOptimization(t *testing.T, db *pgxpool.Pool, userID, jobID, dietID uuid.UUID, key string, tolerance float64, excluded []uuid.UUID) {
	t.Helper()
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].String() < excluded[j].String() })
	canonical := struct {
		DailyDietID      uuid.UUID   `json:"dailyDietId"`
		TolerancePercent float64     `json:"tolerancePercent"`
		ExcludedMealIDs  []uuid.UUID `json:"excludedMealIds"`
	}{dietID, tolerance, excluded}
	payload, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	acknowledgement := fmt.Sprintf(`{"jobId":%q,"status":"queued","pollUrl":%q,"publicationState":"pending"}`, jobID, "/api/v1/optimization/jobs/"+jobID.String())
	record := repository.CheckoutIdempotencyRecord{UserID: userID, Method: fiber.MethodPost, Route: "/optimization/jobs", Key: key, BodyHash: hex.EncodeToString(sum[:]), StatusCode: fiber.StatusAccepted, ResponseBody: []byte(acknowledgement)}
	if err := repository.NewPostgresCheckoutIdempotencyRepository(db).StoreCheckoutIdempotency(context.Background(), record); err != nil {
		t.Fatalf("store pending optimization acknowledgement: %v", err)
	}
}

func task232AppendEntitlement(t *testing.T, db *pgxpool.Pool, userID uuid.UUID, status string, expiresAt time.Time) {
	t.Helper()
	if err := repository.NewPostgresEntitlementRepository(db).AppendEntitlement(context.Background(), repository.Entitlement{UserID: userID, Tier: "trial", Status: status, AllowedModes: []string{"daily_diet_alternative"}, ExpiresAt: &expiresAt}); err != nil {
		t.Fatalf("append %s entitlement: %v", status, err)
	}
}

func task232StreamLength(t *testing.T, client *redis.Client) int64 {
	t.Helper()
	length, err := client.XLen(context.Background(), queue.DefaultStream).Result()
	if err != nil {
		t.Fatalf("read optimization stream length: %v", err)
	}
	return length
}

func task232ResetRedis(t *testing.T, client *redis.Client) {
	t.Helper()
	var cursor uint64
	for {
		keys, next, err := client.Scan(context.Background(), cursor, "mealswapp:optimization:*", 100).Result()
		if err != nil {
			t.Fatalf("scan Task 232 Redis cleanup: %v", err)
		}
		if len(keys) > 0 {
			if err := client.Del(context.Background(), keys...).Err(); err != nil {
				t.Fatalf("delete Task 232 Redis cleanup: %v", err)
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

type task232ConcurrentRequest struct {
	Body    string
	Cookies []*http.Cookie
	CSRF    string
	Key     string
}

func task232SubmitConcurrentUsers(t *testing.T, server *fiber.App, requests []task232ConcurrentRequest) []task232Response {
	t.Helper()
	start := make(chan struct{})
	type indexedResponse struct {
		index    int
		response *http.Response
	}
	responses := make(chan indexedResponse, len(requests))
	errors := make(chan error, len(requests))
	var wait sync.WaitGroup
	for index, input := range requests {
		index, input := index, input
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			request := httptest.NewRequest(fiber.MethodPost, "/api/v1/optimization/jobs", strings.NewReader(input.Body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			request.Header.Set("Idempotency-Key", input.Key)
			request.Header.Set("X-CSRF-Token", input.CSRF)
			for _, cookie := range input.Cookies {
				request.AddCookie(cookie)
			}
			response, err := server.Test(request)
			if err != nil {
				errors <- err
				return
			}
			responses <- indexedResponse{index: index, response: response}
		}()
	}
	close(start)
	wait.Wait()
	close(responses)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent optimization submit: %v", err)
		}
	}
	result := make([]task232Response, len(requests))
	for indexed := range responses {
		response := indexed.response
		raw, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		item := task232Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Raw: raw}
		if err := json.Unmarshal(raw, &item.Envelope); err != nil {
			t.Fatalf("decode concurrent optimization response: %v", err)
		}
		result[indexed.index] = item
	}
	return result
}
