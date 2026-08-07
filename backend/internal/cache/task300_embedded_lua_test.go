package cache

// Implements DESIGN-011 RedisCache Task 300 embedded guarded-write verification.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestTask300ClassificationGenerationScriptIsEmbeddedAndCacheGoHasNoInlineLua(t *testing.T) {
	if strings.TrimSpace(setIfClassificationGenerationLua) == "" {
		t.Fatal("embedded classification generation script is empty")
	}
	const trace = "Implements DESIGN-011 RedisCache guarded cache-miss persistence"
	if !strings.Contains(setIfClassificationGenerationLua, trace) {
		t.Fatalf("embedded classification generation script lacks traceability %q", trace)
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
		t.Fatal("built test binary does not contain the embedded classification generation script")
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate cache package")
	}
	cacheDir := filepath.Dir(currentFile)
	err = filepath.WalkDir(cacheDir, func(path string, entry fs.DirEntry, walkErr error) error {
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
		if bytes.Contains(source, []byte("redis.call")) {
			t.Errorf("non-trivial inline Lua remains in %s", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan cache Go source: %v", err)
	}

	binding, err := os.ReadFile(filepath.Join(cacheDir, "classification_generation_scripts.go"))
	if err != nil {
		t.Fatalf("read classification generation script binding: %v", err)
	}
	if !bytes.Contains(binding, []byte("lua/set_if_classification_generation.lua")) ||
		!bytes.Contains(binding, []byte("DESIGN-011 RedisCache")) {
		t.Fatal("Go embedding binding does not identify the Lua file's DESIGN-011 behavior")
	}
}

func TestTask300ClassificationGenerationUsesEvalShaAndPreservesResultsAndErrors(t *testing.T) {
	const (
		key   = "task300:cache"
		value = `{"result":"exact"}`
	)
	ttl := 1500 * time.Millisecond

	cached := &task300ScriptClient{evalShaValue: 1}
	stored, err := (ClassificationGeneration{client: cached}).SetIfCurrent(context.Background(), 42, key, value, ttl)
	if err != nil || !stored {
		t.Fatalf("SetIfCurrent(cached script) = %v, %v, want true/nil", stored, err)
	}
	cached.assertCalls(t, "evalsha")
	call := cached.calls[0]
	if call.script != setIfClassificationGenerationScript.Hash() {
		t.Fatalf("EVALSHA hash = %q, want %q", call.script, setIfClassificationGenerationScript.Hash())
	}
	if len(call.keys) != 2 || call.keys[0] != classificationGenerationKey || call.keys[1] != key {
		t.Fatalf("EVALSHA keys = %v", call.keys)
	}
	if len(call.args) != 3 || call.args[0] != "42" || call.args[1] != value || call.args[2] != "1500" {
		t.Fatalf("EVALSHA args = %v", call.args)
	}

	uncached := &task300ScriptClient{
		evalShaErr: task300RedisError("NOSCRIPT No matching script. Please use EVAL."),
		evalValue:  1,
	}
	stored, err = (ClassificationGeneration{client: uncached}).SetIfCurrent(context.Background(), 0, key, value, ttl)
	if err != nil || !stored {
		t.Fatalf("SetIfCurrent(NOSCRIPT fallback) = %v, %v, want true/nil", stored, err)
	}
	uncached.assertCalls(t, "evalsha", "eval")
	if uncached.calls[1].script != strings.TrimSpace(setIfClassificationGenerationLua) {
		t.Fatal("EVAL fallback did not receive the embedded Lua source")
	}

	fallbackErr := errors.New("EVAL failed")
	fallbackFailure := &task300ScriptClient{
		evalShaErr: task300RedisError("NOSCRIPT missing"),
		evalErr:    fallbackErr,
	}
	stored, err = (ClassificationGeneration{client: fallbackFailure}).SetIfCurrent(context.Background(), 0, key, value, ttl)
	if stored || !errors.Is(err, fallbackErr) {
		t.Fatalf("SetIfCurrent(EVAL failure) = %v, %v, want false/fallback error", stored, err)
	}
	fallbackFailure.assertCalls(t, "evalsha", "eval")

	dependencyErr := errors.New("Redis unavailable")
	failed := &task300ScriptClient{evalShaErr: dependencyErr}
	stored, err = (ClassificationGeneration{client: failed}).SetIfCurrent(context.Background(), 0, key, value, ttl)
	if stored || !errors.Is(err, dependencyErr) {
		t.Fatalf("SetIfCurrent(Redis failure) = %v, %v, want false/dependency error", stored, err)
	}
	failed.assertCalls(t, "evalsha")

	cancelled := &task300ScriptClient{evalShaErr: context.Canceled}
	stored, err = (ClassificationGeneration{client: cancelled}).SetIfCurrent(context.Background(), 0, key, value, ttl)
	if stored || !errors.Is(err, context.Canceled) {
		t.Fatalf("SetIfCurrent(cancellation) = %v, %v, want false/context.Canceled", stored, err)
	}
	cancelled.assertCalls(t, "evalsha")

	rejected := &task300ScriptClient{evalShaValue: 0}
	stored, err = (ClassificationGeneration{client: rejected}).SetIfCurrent(context.Background(), 0, key, value, ttl)
	if stored || err != nil {
		t.Fatalf("SetIfCurrent(return 0) = %v, %v, want false/nil", stored, err)
	}

	if stored, err = (ClassificationGeneration{}).SetIfCurrent(context.Background(), 0, key, value, ttl); stored || err != nil {
		t.Fatalf("SetIfCurrent(nil dependency) = %v, %v, want false/nil", stored, err)
	}
	ignored := &task300ScriptClient{}
	if stored, err = (ClassificationGeneration{client: ignored}).SetIfCurrent(context.Background(), 0, key, value, 0); stored || err != nil {
		t.Fatalf("SetIfCurrent(nonpositive TTL) = %v, %v, want false/nil", stored, err)
	}
	ignored.assertCalls(t)
}

func TestTask300EmbeddedClassificationGenerationPreservesLiveRedisSemantics(t *testing.T) {
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13"
	}
	client, err := Open(redisURL)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer client.Close()
	peer, err := Open(redisURL)
	if err != nil {
		t.Fatalf("open peer Redis: %v", err)
	}
	defer peer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	const (
		zeroKey     = "task300:zero"
		matchingKey = "task300:matching"
		staleKey    = "task300:stale"
	)
	if err := client.Del(ctx, classificationGenerationKey, zeroKey, matchingKey, staleKey).Err(); err != nil {
		t.Fatalf("reset Redis fixtures: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Del(context.Background(), classificationGenerationKey, zeroKey, matchingKey, staleKey).Err()
	})
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("ScriptFlush() error = %v", err)
	}
	hook := &task300CommandHook{}
	client.AddHook(hook)

	generation := NewClassificationGeneration(client)
	const zeroValue = "zero\x00value"
	zeroTTL := 1500 * time.Millisecond
	stored, err := generation.SetIfCurrent(ctx, 0, zeroKey, zeroValue, zeroTTL)
	if err != nil || !stored {
		t.Fatalf("generation-zero SetIfCurrent() = %v, %v, want true/nil", stored, err)
	}
	hook.assertPrefix(t, "evalsha", "eval")
	if got := client.Get(ctx, zeroKey).Val(); got != zeroValue {
		t.Fatalf("generation-zero value = %q, want %q", got, zeroValue)
	}
	task300AssertTTL(t, client.PTTL(ctx, zeroKey).Val(), zeroTTL)

	advanced, err := generation.Advance(ctx)
	if err != nil || advanced != 1 {
		t.Fatalf("Advance() = %d, %v, want 1/nil", advanced, err)
	}
	if current, err := NewClassificationGeneration(peer).Current(ctx); err != nil || current != 1 {
		t.Fatalf("peer Current() = %d, %v, want 1/nil", current, err)
	}

	const matchingValue = `{"matching":true}`
	matchingTTL := 1375 * time.Millisecond
	stored, err = NewClassificationGeneration(peer).SetIfCurrent(ctx, 1, matchingKey, matchingValue, matchingTTL)
	if err != nil || !stored {
		t.Fatalf("matching-generation SetIfCurrent() = %v, %v, want true/nil", stored, err)
	}
	if got := client.Get(ctx, matchingKey).Val(); got != matchingValue {
		t.Fatalf("matching-generation value = %q, want %q", got, matchingValue)
	}
	task300AssertTTL(t, client.PTTL(ctx, matchingKey).Val(), matchingTTL)

	stored, err = generation.SetIfCurrent(ctx, 0, staleKey, "must-not-store", time.Minute)
	if err != nil || stored {
		t.Fatalf("stale-generation SetIfCurrent() = %v, %v, want false/nil", stored, err)
	}
	if exists := client.Exists(ctx, staleKey).Val(); exists != 0 {
		t.Fatalf("stale-generation key exists = %d, want 0", exists)
	}

	cancelledContext, cancelNow := context.WithCancel(ctx)
	cancelNow()
	if stored, err = generation.SetIfCurrent(cancelledContext, 1, staleKey, "cancelled", time.Minute); stored || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SetIfCurrent() = %v, %v, want false/context.Canceled", stored, err)
	}

	for i := 0; i < 20; i++ {
		raceKey := "task300:race:" + string(rune('a'+i))
		if err := client.Del(ctx, classificationGenerationKey, raceKey).Err(); err != nil {
			t.Fatalf("reset concurrent fixture %d: %v", i, err)
		}
		var (
			writeStored bool
			writeErr    error
			advanceErr  error
		)
		start := make(chan struct{})
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			writeStored, writeErr = generation.SetIfCurrent(ctx, 0, raceKey, "linearizable", time.Minute)
		}()
		go func() {
			defer wait.Done()
			<-start
			_, advanceErr = NewClassificationGeneration(peer).Advance(ctx)
		}()
		close(start)
		wait.Wait()
		if writeErr != nil || advanceErr != nil {
			t.Fatalf("concurrent iteration %d errors: write=%v advance=%v", i, writeErr, advanceErr)
		}
		if current, err := generation.Current(ctx); err != nil || current != 1 {
			t.Fatalf("concurrent iteration %d Current() = %d, %v, want 1/nil", i, current, err)
		}
		exists := client.Exists(ctx, raceKey).Val() == 1
		if exists != writeStored {
			t.Fatalf("concurrent iteration %d key existence=%v, return=%v", i, exists, writeStored)
		}
		_ = client.Del(ctx, raceKey).Err()
	}
}

func task300AssertTTL(t *testing.T, got, want time.Duration) {
	t.Helper()
	if got > want || got < want-300*time.Millisecond {
		t.Fatalf("Redis TTL = %v, want within 300ms of %v", got, want)
	}
}

type task300ScriptCall struct {
	name   string
	script string
	keys   []string
	args   []interface{}
}

type task300RedisError string

func (e task300RedisError) Error() string { return string(e) }

func (task300RedisError) RedisError() {}

type task300ScriptClient struct {
	redis.Scripter
	calls        []task300ScriptCall
	evalShaErr   error
	evalShaValue int64
	evalErr      error
	evalValue    int64
}

func (c *task300ScriptClient) Get(ctx context.Context, _ string) *redis.StringCmd {
	return redis.NewStringCmd(ctx)
}

func (c *task300ScriptClient) Incr(ctx context.Context, _ string) *redis.IntCmd {
	return redis.NewIntCmd(ctx)
}

func (c *task300ScriptClient) EvalSha(ctx context.Context, hash string, keys []string, args ...interface{}) *redis.Cmd {
	c.calls = append(c.calls, task300ScriptCall{name: "evalsha", script: hash, keys: append([]string(nil), keys...), args: append([]interface{}(nil), args...)})
	return task300RedisCmd(ctx, c.evalShaValue, c.evalShaErr)
}

func (c *task300ScriptClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	c.calls = append(c.calls, task300ScriptCall{name: "eval", script: script, keys: append([]string(nil), keys...), args: append([]interface{}(nil), args...)})
	return task300RedisCmd(ctx, c.evalValue, c.evalErr)
}

func (c *task300ScriptClient) assertCalls(t *testing.T, want ...string) {
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

func task300RedisCmd(ctx context.Context, value int64, err error) *redis.Cmd {
	cmd := redis.NewCmd(ctx)
	if err != nil {
		cmd.SetErr(err)
	} else {
		cmd.SetVal(value)
	}
	return cmd
}

type task300CommandHook struct {
	mu       sync.Mutex
	commands []string
}

func (h *task300CommandHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *task300CommandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *task300CommandHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
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

func (h *task300CommandHook) assertPrefix(t *testing.T, want ...string) {
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
