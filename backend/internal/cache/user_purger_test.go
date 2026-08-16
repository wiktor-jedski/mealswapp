package cache

// Implements DESIGN-008 AccountDeleter cache-prefix erasure verification.

import (
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type userPurgeScanPage struct {
	keys   []string
	cursor uint64
	err    error
}

type userPurgeClientStub struct {
	pages     []userPurgeScanPage
	scanCalls int
	matches   []string
	counts    []int64
	deleted   [][]string
	delErr    error
	cancelDel context.CancelFunc
}

func (s *userPurgeClientStub) Scan(ctx context.Context, _ uint64, match string, count int64) *redis.ScanCmd {
	cmd := redis.NewScanCmd(ctx, nil)
	s.matches = append(s.matches, match)
	s.counts = append(s.counts, count)
	if err := ctx.Err(); err != nil {
		cmd.SetErr(err)
		return cmd
	}
	page := s.pages[s.scanCalls]
	s.scanCalls++
	if page.err != nil {
		cmd.SetErr(page.err)
		return cmd
	}
	cmd.SetVal(page.keys, page.cursor)
	return cmd
}

func (s *userPurgeClientStub) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	s.deleted = append(s.deleted, slices.Clone(keys))
	if s.cancelDel != nil {
		s.cancelDel()
	}
	if err := ctx.Err(); err != nil {
		cmd.SetErr(err)
	} else if s.delErr != nil {
		cmd.SetErr(s.delErr)
	} else {
		cmd.SetVal(int64(len(keys)))
	}
	return cmd
}

func TestUserPurgerNilAndEmptyNamespace(t *testing.T) {
	ownerID := uuid.New()
	if err := NewUserPurger(nil).PurgeUser(context.Background(), ownerID); err != nil {
		t.Fatalf("nil client purge error = %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer redisClient.Close()
	if NewUserPurger(redisClient).client == nil {
		t.Fatal("configured Redis client was discarded")
	}
	client := &userPurgeClientStub{pages: []userPurgeScanPage{{}}}
	if err := (UserPurger{client: client}).PurgeUser(context.Background(), ownerID); err != nil {
		t.Fatalf("empty namespace purge error = %v", err)
	}
	if client.scanCalls != 1 || len(client.deleted) != 0 {
		t.Fatalf("empty namespace scans=%d deletes=%v", client.scanCalls, client.deleted)
	}
}

func TestUserPurgerPaginatesAndPreservesOtherNamespaces(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	prefix := "user:" + ownerID.String()
	first := make([]string, 100)
	for i := range first {
		first[i] = prefix + ":custom-items:first:" + strconv.Itoa(i)
	}
	second := make([]string, 50)
	for i := range second {
		second[i] = prefix + ":custom-items:second:" + strconv.Itoa(i)
	}
	otherKey := "user:" + otherID.String() + ":custom-items"
	client := &userPurgeClientStub{pages: []userPurgeScanPage{{keys: first, cursor: 7}, {keys: second}}}
	if err := (UserPurger{client: client}).PurgeUser(context.Background(), ownerID); err != nil {
		t.Fatalf("paginated purge error = %v", err)
	}
	if client.scanCalls != 2 || len(client.deleted) != 2 || len(client.deleted[0])+len(client.deleted[1]) != 150 {
		t.Fatalf("paginated scans=%d deletes=%v", client.scanCalls, client.deleted)
	}
	for i, match := range client.matches {
		if match != prefix+"*" || client.counts[i] != 100 {
			t.Fatalf("scan %d match=%q count=%d", i, match, client.counts[i])
		}
	}
	for _, keys := range client.deleted {
		if slices.Contains(keys, otherKey) {
			t.Fatalf("cross-user key deleted: %q", otherKey)
		}
	}
}

func TestUserPurgerPropagatesScanAndDeleteErrors(t *testing.T) {
	wantErr := errors.New("redis unavailable")
	ownerID := uuid.New()
	for name, client := range map[string]*userPurgeClientStub{
		"scan": {pages: []userPurgeScanPage{{err: wantErr}}},
		"delete": {
			pages:  []userPurgeScanPage{{keys: []string{"user:" + ownerID.String() + ":custom-items"}}},
			delErr: wantErr,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := (UserPurger{client: client}).PurgeUser(context.Background(), ownerID); !errors.Is(err, wantErr) {
				t.Fatalf("PurgeUser() error=%v, want %v", err, wantErr)
			}
		})
	}
}

func TestUserPurgerHonorsCancellationDuringScanAndDelete(t *testing.T) {
	ownerID := uuid.New()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	scanClient := &userPurgeClientStub{pages: []userPurgeScanPage{{}}}
	if err := (UserPurger{client: scanClient}).PurgeUser(canceled, ownerID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled scan error = %v", err)
	}

	deleteCtx, cancelDelete := context.WithCancel(context.Background())
	deleteClient := &userPurgeClientStub{
		pages:     []userPurgeScanPage{{keys: []string{"user:" + ownerID.String() + ":custom-items"}}},
		cancelDel: cancelDelete,
	}
	if err := (UserPurger{client: deleteClient}).PurgeUser(deleteCtx, ownerID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delete error = %v", err)
	}
}

func TestUserPurgerLiveRedisPaginationAndIsolation(t *testing.T) {
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13"
	}
	client, err := Open(redisURL)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	ownerID := uuid.New()
	otherID := uuid.New()
	ownerPrefix := "user:" + ownerID.String()
	ownerKeys := make([]string, 150)
	pipe := client.Pipeline()
	for i := range ownerKeys {
		ownerKeys[i] = ownerPrefix + ":custom-items:" + strconv.Itoa(i)
		pipe.Set(ctx, ownerKeys[i], "private", time.Minute)
	}
	otherKey := "user:" + otherID.String() + ":custom-items"
	pipe.Set(ctx, otherKey, "other", time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		t.Fatalf("seed Redis namespace: %v", err)
	}
	t.Cleanup(func() {
		keys := append(slices.Clone(ownerKeys), otherKey)
		_ = client.Del(context.Background(), keys...).Err()
	})

	if err := NewUserPurger(client).PurgeUser(ctx, ownerID); err != nil {
		t.Fatalf("live paginated purge: %v", err)
	}
	if got := client.Exists(ctx, ownerKeys...).Val(); got != 0 {
		t.Fatalf("live purge retained %d owner keys", got)
	}
	if got := client.Get(ctx, otherKey).Val(); got != "other" {
		t.Fatalf("live purge changed other namespace: %q", got)
	}
}
