package worker

// Implements DESIGN-004 JobStatusTracker Task 301 embedded transition verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
)

func TestTask301TransitionScriptIsEmbeddedAndWorkerGoHasNoInlineLua(t *testing.T) {
	if strings.TrimSpace(optimizationStateTransitionLua) == "" {
		t.Fatal("embedded optimization transition script is empty")
	}
	const trace = "Implements DESIGN-004 JobStatusTracker atomic monotonic publication"
	if !strings.Contains(optimizationStateTransitionLua, trace) {
		t.Fatalf("embedded optimization transition script lacks traceability %q", trace)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if !bytes.Contains(binary, []byte(trace)) {
		t.Fatal("built test binary does not contain the embedded optimization transition script")
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate worker package")
	}
	workerDir := filepath.Dir(currentFile)
	err = filepath.WalkDir(workerDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(source, []byte("redis.call")) || bytes.Contains(source, []byte("cjson.decode")) {
			t.Errorf("non-trivial inline Lua remains in %s", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan worker Go source: %v", err)
	}

	binding, err := os.ReadFile(filepath.Join(workerDir, "optimization_scripts.go"))
	if err != nil {
		t.Fatalf("read optimization script binding: %v", err)
	}
	if !bytes.Contains(binding, []byte("lua/optimization_state_transition.lua")) || !bytes.Contains(binding, []byte("DESIGN-004 JobStatusTracker")) {
		t.Fatal("Go embedding binding does not identify the Lua file's DESIGN-004 behavior")
	}
}

func TestTask301TransitionUsesEvalShaAndFallsBackOnlyForNoScript(t *testing.T) {
	job := task301QueuedJob()

	cached := &task301ScriptClient{evalShaValue: 1}
	if err := NewRedisOptimizationJobStore(cached).Save(context.Background(), job); err != nil {
		t.Fatalf("Save(cached script) error = %v", err)
	}
	cached.assertCalls(t, "evalsha")
	if cached.calls[0].script != optimizationStateTransitionScript.Hash() {
		t.Fatalf("EVALSHA hash = %q, want %q", cached.calls[0].script, optimizationStateTransitionScript.Hash())
	}

	uncached := &task301ScriptClient{
		evalShaErr: task301RedisError("NOSCRIPT No matching script. Please use EVAL."),
		evalValue:  1,
	}
	if err := NewRedisOptimizationJobStore(uncached).Save(context.Background(), job); err != nil {
		t.Fatalf("Save(NOSCRIPT fallback) error = %v", err)
	}
	uncached.assertCalls(t, "evalsha", "eval")
	if uncached.calls[1].script != strings.TrimSpace(optimizationStateTransitionLua) {
		t.Fatal("EVAL fallback did not receive the embedded Lua source")
	}
	if len(uncached.calls[1].keys) != 2 || len(uncached.calls[1].args) != 5 {
		t.Fatalf("EVAL fallback keys/args = %d/%d, want 2/5", len(uncached.calls[1].keys), len(uncached.calls[1].args))
	}
	fallbackFailure := &task301ScriptClient{
		evalShaErr: task301RedisError("NOSCRIPT No matching script. Please use EVAL."),
		evalErr:    errors.New("EVAL failed"),
	}
	err := NewRedisOptimizationJobStore(fallbackFailure).Save(context.Background(), job)
	if !errors.Is(err, queue.ErrQueueUnavailable) || !strings.Contains(err.Error(), "EVAL failed") {
		t.Fatalf("Save(EVAL failure) error = %v, want mapped fallback error", err)
	}
	fallbackFailure.assertCalls(t, "evalsha", "eval")

	failed := &task301ScriptClient{evalShaErr: errors.New("Redis unavailable")}
	err = NewRedisOptimizationJobStore(failed).Save(context.Background(), job)
	if !errors.Is(err, queue.ErrQueueUnavailable) {
		t.Fatalf("Save(Redis failure) error = %v, want ErrQueueUnavailable", err)
	}
	failed.assertCalls(t, "evalsha")

	cancelled := &task301ScriptClient{evalShaErr: context.Canceled}
	err = NewRedisOptimizationJobStore(cancelled).Save(context.Background(), job)
	if !errors.Is(err, queue.ErrQueueUnavailable) || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("Save(cancellation) error = %v, want mapped cancellation", err)
	}

	missing := &task301ScriptClient{evalShaValue: -1}
	changed, err := NewRedisOptimizationJobStore(missing).transition(context.Background(), job, "save")
	if changed || !errors.Is(err, ErrOptimizationJobNotFound) {
		t.Fatalf("transition(return -1) = %v, %v, want false/ErrOptimizationJobNotFound", changed, err)
	}
	forbidden := &task301ScriptClient{evalShaValue: 0}
	changed, err = NewRedisOptimizationJobStore(forbidden).transition(context.Background(), job, "save")
	if changed || err != nil {
		t.Fatalf("transition(return 0) = %v, %v, want false/nil", changed, err)
	}
}

func TestTask301EmbeddedTransitionPreservesRedisStateMachineAndTTLs(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	ctx := context.Background()
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("ScriptFlush() error = %v", err)
	}
	hook := &task301CommandHook{}
	client.AddHook(hook)

	const jobTTL = 2 * time.Minute
	store := NewRedisOptimizationJobStoreWithTTL(client, jobTTL)
	job := task301QueuedJob()
	if err := store.Save(ctx, job); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	hook.assertPrefix(t, "evalsha", "eval")
	task301AssertTTL(t, client.PTTL(ctx, optimizationJobKey(job.JobID)).Val(), jobTTL)

	if err := store.Save(ctx, job); err != nil {
		t.Fatalf("queued self-transition Save() error = %v", err)
	}
	startedAt := time.Now().UTC()
	processing, err := store.MarkProcessing(ctx, job.JobID, startedAt)
	if err != nil {
		t.Fatalf("MarkProcessing(queued) error = %v", err)
	}
	processingAgain, err := store.MarkProcessing(ctx, job.JobID, startedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("MarkProcessing(processing) error = %v", err)
	}
	if processing.StartedAt == nil || processingAgain.StartedAt == nil || !processing.StartedAt.Equal(*processingAgain.StartedAt) {
		t.Fatalf("processing self-transition changed start time: %v then %v", processing.StartedAt, processingAgain.StartedAt)
	}

	if err := store.PublishCompleted(ctx, job.JobID, []optimization.DietAlternative{task221Alternative(0.5)}, time.Now().UTC()); err != nil {
		t.Fatalf("PublishCompleted(processing) error = %v", err)
	}
	task301AssertTTL(t, client.PTTL(ctx, optimizationJobKey(job.JobID)).Val(), jobTTL)
	task301AssertTTL(t, client.PTTL(ctx, optimizationExpiredKey(job.JobID)).Val(), time.Hour)
	if owner := client.Get(ctx, optimizationExpiredKey(job.JobID)).Val(); owner != job.UserID.String() {
		t.Fatalf("expiry marker owner = %q, want %q", owner, job.UserID)
	}
	failure := OptimizationJobFailure{Code: optimization.FailureCodeSolverTimeout, Message: safeFailureMessage(optimization.FailureCodeSolverTimeout)}
	if err := store.PublishFailed(ctx, job.JobID, nil, failure, time.Now().UTC()); err == nil {
		t.Fatal("PublishFailed(completed) error = nil, want terminal immutability conflict")
	}

	failedJob := task301QueuedJob()
	if err := store.Save(ctx, failedJob); err != nil {
		t.Fatalf("Save(failed fixture) error = %v", err)
	}
	if _, err := store.MarkProcessing(ctx, failedJob.JobID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkProcessing(failed fixture) error = %v", err)
	}
	if err := store.PublishFailed(ctx, failedJob.JobID, nil, failure, time.Now().UTC()); err != nil {
		t.Fatalf("PublishFailed(processing) error = %v", err)
	}
	task301AssertTTL(t, client.PTTL(ctx, optimizationJobKey(failedJob.JobID)).Val(), jobTTL)
	task301AssertTTL(t, client.PTTL(ctx, optimizationExpiredKey(failedJob.JobID)).Val(), time.Hour)
	if owner := client.Get(ctx, optimizationExpiredKey(failedJob.JobID)).Val(); owner != failedJob.UserID.String() {
		t.Fatalf("failed expiry marker owner = %q, want %q", owner, failedJob.UserID)
	}
	if err := store.PublishCompleted(ctx, failedJob.JobID, []optimization.DietAlternative{task221Alternative(0.5)}, time.Now().UTC()); err == nil {
		t.Fatal("PublishCompleted(failed) error = nil, want terminal immutability conflict")
	}

	missing := task301QueuedJob()
	missing.Status = OptimizationJobProcessing
	if changed, err := store.transition(ctx, missing, "processing"); changed || !errors.Is(err, ErrOptimizationJobNotFound) {
		t.Fatalf("missing processing transition = %v, %v, want false/ErrOptimizationJobNotFound", changed, err)
	}
	expired := task301QueuedJob()
	if err := client.Set(ctx, optimizationExpiredKey(expired.JobID), expired.UserID.String(), time.Hour).Err(); err != nil {
		t.Fatalf("seed expiry marker: %v", err)
	}
	if changed, err := store.transition(ctx, expired, "save"); changed || !errors.Is(err, ErrOptimizationJobNotFound) {
		t.Fatalf("expired save transition = %v, %v, want false/ErrOptimizationJobNotFound", changed, err)
	}

	cancelled := task301QueuedJob()
	cancelled.Status = OptimizationJobCancelled
	payload, err := json.Marshal(cancelled)
	if err != nil {
		t.Fatalf("marshal cancelled fixture: %v", err)
	}
	if err := client.Set(ctx, optimizationJobKey(cancelled.JobID), payload, jobTTL).Err(); err != nil {
		t.Fatalf("seed cancelled job: %v", err)
	}
	cancelled.Status = OptimizationJobProcessing
	if changed, err := store.transition(ctx, cancelled, "processing"); changed || err != nil {
		t.Fatalf("cancelled processing transition = %v, %v, want false/nil", changed, err)
	}

	cancelledContext, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.transition(cancelledContext, task301QueuedJob(), "save"); !errors.Is(err, queue.ErrQueueUnavailable) {
		t.Fatalf("canceled transition error = %v, want ErrQueueUnavailable", err)
	}
}

func task301QueuedJob() OptimizationJob {
	return OptimizationJob{
		JobID: uuid.New(), UserID: uuid.New(), DailyDietID: uuid.New(),
		Status: OptimizationJobQueued, CreatedAt: time.Now().UTC(),
	}
}

func task301AssertTTL(t *testing.T, got, want time.Duration) {
	t.Helper()
	if got > want || got < want-2*time.Second {
		t.Fatalf("Redis TTL = %v, want within two seconds of %v", got, want)
	}
}

type task301ScriptCall struct {
	name   string
	script string
	keys   []string
	args   []interface{}
}

type task301RedisError string

func (e task301RedisError) Error() string { return string(e) }

func (task301RedisError) RedisError() {}

type task301ScriptClient struct {
	redis.UniversalClient
	calls        []task301ScriptCall
	evalShaErr   error
	evalShaValue int64
	evalErr      error
	evalValue    int64
}

func (c *task301ScriptClient) EvalSha(ctx context.Context, hash string, keys []string, args ...interface{}) *redis.Cmd {
	c.calls = append(c.calls, task301ScriptCall{name: "evalsha", script: hash, keys: append([]string(nil), keys...), args: append([]interface{}(nil), args...)})
	return task301RedisCmd(ctx, c.evalShaValue, c.evalShaErr)
}

func (c *task301ScriptClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	c.calls = append(c.calls, task301ScriptCall{name: "eval", script: script, keys: append([]string(nil), keys...), args: append([]interface{}(nil), args...)})
	return task301RedisCmd(ctx, c.evalValue, c.evalErr)
}

func (c *task301ScriptClient) assertCalls(t *testing.T, want ...string) {
	t.Helper()
	if len(c.calls) != len(want) {
		t.Fatalf("Redis script calls = %+v, want %v", c.calls, want)
	}
	for i, name := range want {
		if c.calls[i].name != name {
			t.Fatalf("Redis script call %d = %q, want %q", i, c.calls[i].name, name)
		}
	}
}

func task301RedisCmd(ctx context.Context, value int64, err error) *redis.Cmd {
	cmd := redis.NewCmd(ctx)
	if err != nil {
		cmd.SetErr(err)
	} else {
		cmd.SetVal(value)
	}
	return cmd
}

type task301CommandHook struct {
	mu       sync.Mutex
	commands []string
}

func (h *task301CommandHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *task301CommandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *task301CommandHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		name := strings.ToLower(cmd.Name())
		if name == "evalsha" || name == "eval" {
			h.mu.Lock()
			h.commands = append(h.commands, name)
			h.mu.Unlock()
		}
		return next(ctx, cmd)
	}
}

func (h *task301CommandHook) assertPrefix(t *testing.T, want ...string) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.commands) < len(want) {
		t.Fatalf("Redis script commands = %v, want prefix %v", h.commands, want)
	}
	for i, name := range want {
		if h.commands[i] != name {
			t.Fatalf("Redis script command %d = %q, want %q; all commands %v", i, h.commands[i], name, h.commands)
		}
	}
}
