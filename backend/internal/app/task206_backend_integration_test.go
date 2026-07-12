package app

// Implements DESIGN-004 JobQueueManager Task 206 backend integration gate.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// TestTask206BackendIntegrationGate verifies IT-ARCH-004-001 and
// IT-ARCH-004-004, ARCH-004, DESIGN-004, and
// SW-REQ-006/SW-REQ-021/SW-REQ-022/SW-REQ-023/SW-REQ-030 across the saved-diet,
// API, Redis, worker, and native solver boundaries.
func TestTask206BackendIntegrationGate(t *testing.T) {
	clpPath, clpVersion := task206CLP(t)
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	resetTask206Redis(t, redisClient)
	t.Cleanup(func() { resetTask206Redis(t, redisClient) })

	cfg := liveDailyDietAPIConfig()
	cfg.CLPExecutable = clpPath
	cfg.CLPVersion = clpVersion
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}

	mealIDs := createTask206Meals(t, db)
	userCookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-206-owner-"+uuid.NewString()+"@example.test")
	grantTask206Trial(t, db, userID)
	otherCookies, otherUserID := registerLiveDailyDietUser(t, server, cfg, "task-206-other-"+uuid.NewString()+"@example.test")
	_ = otherUserID
	var csrfToken string
	csrfToken, userCookies = fetchLiveDailyDietCSRF(t, server, userCookies)
	_, otherCookies = fetchLiveDailyDietCSRF(t, server, otherCookies)
	dietID := createTask206SavedDiet(t, server, userCookies, csrfToken, mealIDs[0])

	ctx, cancel := context.WithCancel(context.Background())
	workerDone := startTask206Worker(t, ctx, cfg, db, redisClient)

	// Server-supplied target macros are intentionally wrong. The worker must
	// reload the saved diet and derive the authoritative target from PostgreSQL.
	nominalBody := task206OptimizationBody(dietID, optimization.MacroTarget{Protein: 999, Carbohydrates: 999, Fat: 999}, nil)
	nominalJobID := submitTask206Job(t, server, nominalBody, userCookies, csrfToken, "task-206-nominal-"+uuid.NewString())
	nominal := waitTask206Job(t, server, nominalJobID, userCookies, workerDone)
	assertTask206Alternatives(t, nominal.Envelope, mealIDs, 20, 30, 10, nil)

	// A successful request with one excluded candidate proves the exclusion
	// list survives API serialization, Redis job state, and worker reload.
	successfulExcluded := []uuid.UUID{mealIDs[1]}
	excludedJobID := submitTask206Job(t, server, task206OptimizationBody(dietID, optimization.MacroTarget{Protein: 999, Carbohydrates: 999, Fat: 999}, successfulExcluded), userCookies, csrfToken, "task-206-successful-exclusion-"+uuid.NewString())
	excludedResult := waitTask206Job(t, server, excludedJobID, userCookies, workerDone)
	assertTask206Alternatives(t, excludedResult.Envelope, mealIDs, 20, 30, 10, successfulExcluded)

	// Polling is user-scoped even when the job ID is valid.
	otherPoll := pollTask206Job(t, server, nominalJobID, otherCookies)
	if otherPoll.StatusCode != fiber.StatusNotFound {
		t.Fatalf("cross-user poll status = %d, want 404", otherPoll.StatusCode)
	}

	// A second stream delivery for a completed logical job must be acknowledged
	// without rerunning or changing the authoritative alternatives.
	cancel()
	if err := task206WorkerResult(workerDone); err != nil {
		t.Fatalf("worker pause before duplicate delivery: %v", err)
	}
	beforeDuplicate := task206AlternativesJSON(t, nominal.Envelope)
	manager := queue.NewJobQueueManager(redisClient, queue.Config{Consumer: "task-206-duplicate-" + uuid.NewString(), ReadBlock: 10 * time.Millisecond})
	if _, err := redisClient.XAdd(context.Background(), &redis.XAddArgs{Stream: queue.DefaultStream, Values: map[string]any{"job_id": nominalJobID.String()}}).Result(); err != nil {
		t.Fatalf("duplicate XADD error = %v", err)
	}
	duplicateDelivery, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("reserve duplicate delivery: %v", err)
	}
	store := worker.NewRedisOptimizationJobStore(redisClient)
	duplicateProcessorCalled := false
	if err := manager.Process(context.Background(), duplicateDelivery, func(ctx context.Context, delivery queue.Job) error {
		duplicateProcessorCalled = true
		_, loadErr := store.Load(ctx, uuid.MustParse(delivery.ID))
		return loadErr
	}); err != nil {
		t.Fatalf("process duplicate delivery: %v", err)
	}
	if duplicateProcessorCalled {
		t.Fatal("duplicate delivery invoked the processor after terminal publication")
	}
	afterDuplicate := waitTask206Job(t, server, nominalJobID, userCookies, workerDone)
	if got := task206AlternativesJSON(t, afterDuplicate.Envelope); got != beforeDuplicate {
		t.Fatalf("duplicate delivery changed alternatives: before=%s after=%s", beforeDuplicate, got)
	}

	ctx, cancel = context.WithCancel(context.Background())
	workerDone = startTask206Worker(t, ctx, cfg, db, redisClient)

	// Two independent submissions are accepted concurrently and both complete.
	concurrentIDs := submitTask206Concurrent(t, server, nominalBody, userCookies, csrfToken, 2)
	for _, jobID := range concurrentIDs {
		result := waitTask206Job(t, server, jobID, userCookies, workerDone)
		assertTask206Alternatives(t, result.Envelope, mealIDs, 20, 30, 10, nil)
	}

	// Excluding every feasible meal produces a solver-reported infeasible job,
	// surfaced as a stable safe failure instead of an API panic or hang.
	infeasibleJobID := submitTask206Job(t, server, task206OptimizationBody(dietID, optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, mealIDs), userCookies, csrfToken, "task-206-infeasible-"+uuid.NewString())
	infeasible := waitTask206Job(t, server, infeasibleJobID, userCookies, workerDone)
	assertTask206Failure(t, infeasible, optimization.FailureCodeSolverInfeasible)

	// Redis failure is reported before any synchronous solver fallback.
	outageRedis := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:63999",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
		MaxRetries:   0,
	})
	t.Cleanup(func() { _ = outageRedis.Close() })
	outageServer, err := NewProduction(cfg, db, outageRedis, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() outage error = %v", err)
	}
	outageCSRF, outageCookies := fetchLiveDailyDietCSRF(t, outageServer, userCookies)
	outageResponse := liveDailyDietRequest(t, outageServer, fiber.MethodPost, "/api/v1/optimization/jobs", nominalBody, outageCookies, "task-206-outage-"+uuid.NewString(), outageCSRF)
	var outage httpapi.Envelope
	if err := json.NewDecoder(outageResponse.Body).Decode(&outage); err != nil {
		t.Fatalf("decode Redis outage response: %v", err)
	}
	outageResponse.Body.Close()
	if outageResponse.StatusCode != fiber.StatusServiceUnavailable || outage.Error == nil || outage.Error.Code != "queue_unavailable" {
		t.Fatalf("Redis outage response = %d %+v, want 503 queue_unavailable", outageResponse.StatusCode, outage.Error)
	}

	cancel()
	if err := task206WorkerResult(workerDone); err != nil {
		t.Fatalf("worker shutdown error = %v", err)
	}
}

// TestTask206TimeoutAndOwnershipGate verifies IT-ARCH-004-005, ARCH-004,
// DESIGN-004, and SW-REQ-021/SW-REQ-022 through the real solver-wrapper
// timeout boundary. The
// production wrapper remains capped at optimization.SolverDeadline (30s); the
// injected runner shortens only this integration fixture's wait.
func TestTask206TimeoutAndOwnershipGate(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	resetTask206Redis(t, redisClient)
	t.Cleanup(func() { resetTask206Redis(t, redisClient) })

	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	mealIDs := createTask206Meals(t, db)
	cookies, userID := registerLiveDailyDietUser(t, server, cfg, "task-206-timeout-"+uuid.NewString()+"@example.test")
	grantTask206Trial(t, db, userID)
	var csrfToken string
	csrfToken, cookies = fetchLiveDailyDietCSRF(t, server, cookies)
	dietID := createTask206SavedDiet(t, server, cookies, csrfToken, mealIDs[0])
	jobID := submitTask206Job(t, server, task206OptimizationBody(dietID, optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, nil), cookies, csrfToken, "task-206-timeout-"+uuid.NewString())

	manager := queue.NewJobQueueManager(redisClient, queue.Config{Consumer: "task-206-timeout-" + uuid.NewString(), ReadBlock: 10 * time.Millisecond})
	delivery, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("reserve timeout delivery: %v", err)
	}
	mealRepository := repository.NewPostgresMealRepository(db)
	dietRepository := repository.NewPostgresSavedDataRepository(db)
	inputs := worker.NewRepositoryOptimizationInputLoader(optimization.NewConstraintBuilder(mealRepository, dietRepository))
	solver := optimization.NewLPSolverWrapper(optimization.CLPConfig{
		Executable: "clp", ExpectedVersion: optimization.SupportedCLPVersion, Timeout: 10 * time.Millisecond,
		Runner: task206TimeoutRunner,
	})
	processor := worker.NewOptimizationProcessor(worker.NewRedisOptimizationJobStore(redisClient), inputs, solver)
	if err := manager.Process(context.Background(), delivery, processor.Process); err != nil {
		t.Fatalf("process timeout delivery: %v", err)
	}
	failed := pollTask206Job(t, server, jobID, cookies)
	assertTask206Failure(t, failed, optimization.FailureCodeSolverTimeout)
}

// task206TimeoutRunner waits for the wrapper's bounded context rather than
// sleeping for the production 30-second deadline.
func task206TimeoutRunner(ctx context.Context, _ string, _ []string, _ io.Writer, _ io.Writer) error {
	<-ctx.Done()
	return ctx.Err()
}

func task206CLP(t *testing.T) (string, string) {
	t.Helper()
	configured := os.Getenv("MEALSWAPP_CLP_EXECUTABLE")
	path := configured
	if path == "" {
		var err error
		path, err = exec.LookPath(optimization.DefaultCLPExecutable)
		if err != nil {
			t.Fatalf("Task 206 setup failure: required CLP executable %q is unavailable; set MEALSWAPP_CLP_EXECUTABLE or run the CLP-capable worker environment: %v", optimization.DefaultCLPExecutable, err)
		}
	}
	version := os.Getenv("MEALSWAPP_CLP_VERSION")
	if version == "" {
		version = optimization.SupportedCLPVersion
	}
	if err := optimization.NewLPSolverWrapper(optimization.CLPConfig{Executable: path, ExpectedVersion: version}).StartupCheck(context.Background()); err != nil {
		t.Fatalf("Task 206 setup failure: required CLP executable %q (expected version %s) failed startup check: %v", path, version, err)
	}
	return path, version
}

func openTask206Redis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Skipf("Redis integration URL is invalid: %v", err)
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Redis integration service unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func resetTask206Redis(t *testing.T, client *redis.Client) {
	t.Helper()
	ctx := context.Background()
	patterns := []string{queue.DefaultStream, "mealswapp:optimization:job:v1:*", "mealswapp:optimization:expired:v1:*", "mealswapp:optimization:attempt:v1:*", "mealswapp:optimization:lock:v1:*"}
	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				t.Fatalf("scan Redis cleanup %q: %v", pattern, err)
			}
			if len(keys) > 0 {
				if err := client.Del(ctx, keys...).Err(); err != nil {
					t.Fatalf("delete Redis cleanup %q: %v", pattern, err)
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
}

func createTask206Meals(t *testing.T, db *pgxpool.Pool) []uuid.UUID {
	t.Helper()
	mealRepository := repository.NewPostgresMealRepository(db)
	ids := make([]uuid.UUID, 0, 5)
	for index := 0; index < 5; index++ {
		id, err := mealRepository.Create(context.Background(), repository.MealEntity{
			Type: repository.MealTypeSingle, Name: fmt.Sprintf("Task 206 meal %d", index), PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10},
		})
		if err != nil {
			t.Fatalf("create fixture meal %d: %v", index, err)
		}
		ids = append(ids, id)
	}
	return ids
}

func grantTask206Trial(t *testing.T, db *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	expires := time.Now().UTC().Add(24 * time.Hour)
	err := repository.NewPostgresEntitlementRepository(db).AppendEntitlement(context.Background(), repository.Entitlement{
		UserID: userID, Tier: "trial", Status: "active", AllowedModes: []string{"daily_diet_alternative"}, ExpiresAt: &expires,
	})
	if err != nil {
		t.Fatalf("grant trial entitlement: %v", err)
	}
}

func createTask206SavedDiet(t *testing.T, server *fiber.App, cookies []*http.Cookie, csrfToken string, mealID uuid.UUID) uuid.UUID {
	t.Helper()
	body := fmt.Sprintf(`{"name":"Task 206 saved diet","entries":[{"mealId":%q,"quantity":100,"unit":"g","position":0}]}`, mealID.String())
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", body, cookies, "task-206-diet-"+uuid.NewString(), csrfToken)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create saved diet status = %d body = %+v", response.StatusCode, envelope)
	}
	return liveUUIDFromData(t, envelope.Data, "id")
}

func task206OptimizationBody(dietID uuid.UUID, target optimization.MacroTarget, excluded []uuid.UUID) string {
	ids := make([]string, 0, len(excluded))
	for _, id := range excluded {
		ids = append(ids, fmt.Sprintf("%q", id.String()))
	}
	return fmt.Sprintf(`{"dailyDietId":%q,"targetMacros":{"protein":%v,"carbohydrates":%v,"fat":%v},"tolerancePercent":0,"excludedMealIds":[%s]}`, dietID.String(), target.Protein, target.Carbohydrates, target.Fat, strings.Join(ids, ","))
}

func submitTask206Job(t *testing.T, server *fiber.App, body string, cookies []*http.Cookie, csrfToken, key string) uuid.UUID {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/optimization/jobs", body, cookies, key, csrfToken)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusAccepted {
		t.Fatalf("submit optimization status = %d body = %+v", response.StatusCode, envelope)
	}
	return liveUUIDFromData(t, envelope.Data, "jobId")
}

func submitTask206Concurrent(t *testing.T, server *fiber.App, body string, cookies []*http.Cookie, csrfToken string, count int) []uuid.UUID {
	t.Helper()
	start := make(chan struct{})
	responses := make(chan *http.Response, count)
	errorsCh := make(chan error, count)
	var wait sync.WaitGroup
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			request := httptest.NewRequest(fiber.MethodPost, "/api/v1/optimization/jobs", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", fmt.Sprintf("task-206-concurrent-%d-%s", index, uuid.NewString()))
			request.Header.Set("X-CSRF-Token", csrfToken)
			for _, cookie := range cookies {
				request.AddCookie(cookie)
			}
			response, err := server.Test(request)
			if err != nil {
				errorsCh <- err
				return
			}
			responses <- response
		}(index)
	}
	close(start)
	wait.Wait()
	close(responses)
	close(errorsCh)
	jobIDs := make([]uuid.UUID, 0, count)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent submission: %v", err)
		}
	}
	for response := range responses {
		envelope := decodeLiveDailyDietEnvelope(t, response)
		response.Body.Close()
		if response.StatusCode != fiber.StatusAccepted {
			t.Fatalf("concurrent submission status = %d body = %+v", response.StatusCode, envelope)
		}
		jobIDs = append(jobIDs, liveUUIDFromData(t, envelope.Data, "jobId"))
	}
	if len(jobIDs) != count {
		t.Fatalf("concurrent jobs = %d, want %d", len(jobIDs), count)
	}
	return jobIDs
}

func startTask206Worker(t *testing.T, ctx context.Context, cfg config.Config, db *pgxpool.Pool, redisClient *redis.Client) <-chan error {
	t.Helper()
	mealRepository := repository.NewPostgresMealRepository(db)
	dietRepository := repository.NewPostgresSavedDataRepository(db)
	store := worker.NewRedisOptimizationJobStore(redisClient)
	inputs := worker.NewRepositoryOptimizationInputLoader(optimization.NewConstraintBuilder(mealRepository, dietRepository))
	solver := optimization.NewLPSolverWrapper(optimization.CLPConfig{Executable: cfg.CLPExecutable, ExpectedVersion: cfg.CLPVersion})
	processor := worker.NewOptimizationProcessor(store, inputs, solver)
	done := make(chan error, 1)
	go func() {
		done <- worker.RunWithProcessor(ctx, cfg, redisClient, processor.Process, processor.Terminal)
	}()
	return done
}

func waitTask206Job(t *testing.T, server *fiber.App, jobID uuid.UUID, cookies []*http.Cookie, workerDone <-chan error) task206PollResult {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-workerDone:
			t.Fatalf("worker stopped before job completion: %v", err)
		default:
		}
		envelope := pollTask206Job(t, server, jobID, cookies)
		if envelope.StatusCode == fiber.StatusOK && envelope.Envelope.Data != nil {
			status, _ := envelope.Envelope.Data["status"].(string)
			if status == string(worker.OptimizationJobCompleted) || status == string(worker.OptimizationJobFailed) {
				return envelope
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach a terminal state", jobID)
	return task206PollResult{}
}

type task206PollResult struct {
	StatusCode int
	Envelope   httpapi.Envelope
}

func pollTask206Job(t *testing.T, server *fiber.App, jobID uuid.UUID, cookies []*http.Cookie) task206PollResult {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/optimization/jobs/"+jobID.String(), "", cookies, "", "")
	var envelope httpapi.Envelope
	err := json.NewDecoder(response.Body).Decode(&envelope)
	response.Body.Close()
	if err != nil {
		t.Fatalf("decode optimization poll: %v", err)
	}
	return task206PollResult{StatusCode: response.StatusCode, Envelope: envelope}
}

func assertTask206Alternatives(t *testing.T, envelope httpapi.Envelope, mealIDs []uuid.UUID, protein, carbohydrates, fat float64, excluded []uuid.UUID) {
	t.Helper()
	if envelope.Data["status"] != string(worker.OptimizationJobCompleted) {
		t.Fatalf("optimization status = %v, want completed", envelope.Data["status"])
	}
	alternatives, ok := envelope.Data["alternatives"].([]any)
	if !ok || len(alternatives) == 0 || len(alternatives) > optimization.MaxAlternativeCount {
		t.Fatalf("alternatives = %#v, want one to three", envelope.Data["alternatives"])
	}
	if len(alternatives) < 2 {
		t.Fatalf("alternatives = %d, want at least two distinct alternatives", len(alternatives))
	}
	allowedSet := make(map[string]struct{}, len(mealIDs))
	for _, id := range mealIDs {
		allowedSet[id.String()] = struct{}{}
	}
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, id := range excluded {
		excludedSet[id.String()] = struct{}{}
	}
	previousCalories := -math.MaxFloat64
	seenSets := make(map[string]struct{}, len(alternatives))
	for index, raw := range alternatives {
		alternative, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("alternative %d has type %T", index, raw)
		}
		macros := alternative["macros"].(map[string]any)
		if !task206CloseEnough(macros["protein"].(float64), protein) || !task206CloseEnough(macros["carbohydrates"].(float64), carbohydrates) || !task206CloseEnough(macros["fat"].(float64), fat) {
			t.Fatalf("alternative %d macros = %+v, want %.2f/%.2f/%.2f", index, macros, protein, carbohydrates, fat)
		}
		calories := alternative["calories"].(float64)
		if calories < previousCalories-1e-7 {
			t.Fatalf("calories are not ordered at alternative %d: previous=%v current=%v", index, previousCalories, calories)
		}
		previousCalories = calories
		meals := alternative["meals"].([]any)
		ids := make([]string, 0, len(meals))
		for _, rawMeal := range meals {
			meal := rawMeal.(map[string]any)
			id := meal["mealId"].(string)
			if _, found := allowedSet[id]; !found {
				t.Fatalf("alternative %d contains a meal outside the fixture: %s", index, id)
			}
			if _, found := excludedSet[id]; found {
				t.Fatalf("alternative %d contains excluded meal %s", index, id)
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		key := strings.Join(ids, ",")
		if _, duplicate := seenSets[key]; duplicate {
			t.Fatalf("alternative %d repeats selected meal set %q", index, key)
		}
		seenSets[key] = struct{}{}
	}
	_ = mealIDs
}

func assertTask206Failure(t *testing.T, result task206PollResult, code optimization.OptimizationFailureCode) {
	t.Helper()
	if result.StatusCode != fiber.StatusOK || result.Envelope.Data["status"] != string(worker.OptimizationJobFailed) {
		t.Fatalf("failure poll = %d %+v", result.StatusCode, result.Envelope.Data)
	}
	failure, ok := result.Envelope.Data["failure"].(map[string]any)
	if !ok || failure["code"] != string(code) {
		t.Fatalf("failure = %#v, want %q", result.Envelope.Data["failure"], code)
	}
}

func task206AlternativesJSON(t *testing.T, envelope httpapi.Envelope) string {
	t.Helper()
	data, err := json.Marshal(envelope.Data["alternatives"])
	if err != nil {
		t.Fatalf("marshal alternatives: %v", err)
	}
	return string(data)
}

func task206CloseEnough(got, want float64) bool {
	return math.Abs(got-want) <= 1e-6
}

func task206WorkerResult(done <-chan error) error {
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		return errors.New("worker did not stop")
	}
}
