package cache

// Implements DESIGN-009 TagManager search/filter cache invalidation verification.

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/redis/go-redis/v9"
)

type filterInvalidatorStub struct{ calls int }

func (s *filterInvalidatorStub) Invalidate() { s.calls++ }

type classificationRedisStub struct {
	pages     []userPurgeScanPage
	scanCalls int
	matches   []string
	deleted   [][]string
	delErr    error
}

func (s *classificationRedisStub) Scan(ctx context.Context, _ uint64, match string, _ int64) *redis.ScanCmd {
	cmd := redis.NewScanCmd(ctx, nil)
	s.matches = append(s.matches, match)
	page := s.pages[s.scanCalls]
	s.scanCalls++
	if page.err != nil {
		cmd.SetErr(page.err)
	} else {
		cmd.SetVal(page.keys, page.cursor)
	}
	return cmd
}

func (s *classificationRedisStub) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	s.deleted = append(s.deleted, slices.Clone(keys))
	if s.delErr != nil {
		cmd.SetErr(s.delErr)
	} else {
		cmd.SetVal(int64(len(keys)))
	}
	return cmd
}

func TestClassificationInvalidatorClearsFilterAndSearchNamespaces(t *testing.T) {
	filter := &filterInvalidatorStub{}
	redisClient := &classificationRedisStub{pages: []userPurgeScanPage{{keys: []string{"search:search-response-v3:first"}, cursor: 7}, {keys: []string{"search:search-response-v3:second"}}}}
	(ClassificationInvalidator{filter: filter, redis: redisClient}).Invalidate()
	if filter.calls != 1 || redisClient.scanCalls != 2 || len(redisClient.deleted) != 2 {
		t.Fatalf("filter calls=%d scans=%d deleted=%v", filter.calls, redisClient.scanCalls, redisClient.deleted)
	}
	for _, match := range redisClient.matches {
		if match != "search:"+SearchSchemaVersion+"*:*" {
			t.Fatalf("unexpected namespace match %q", match)
		}
	}
}

func TestClassificationInvalidatorNilAndRedisFailuresAreSafe(t *testing.T) {
	ClassificationInvalidator{}.Invalidate()
	want := errors.New("redis unavailable")
	filter := &filterInvalidatorStub{}
	for _, redisClient := range []*classificationRedisStub{
		{pages: []userPurgeScanPage{{err: want}}},
		{pages: []userPurgeScanPage{{keys: []string{"search:search-response-v3:key"}}}, delErr: want},
	} {
		(ClassificationInvalidator{filter: filter, redis: redisClient}).Invalidate()
	}
	if filter.calls != 2 {
		t.Fatalf("filter invalidations = %d", filter.calls)
	}
	NewClassificationInvalidator(filter, nil).Invalidate()
	if filter.calls != 3 {
		t.Fatalf("constructor filter invalidations = %d", filter.calls)
	}
}
