package cache

import (
	"context"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// UserPurger removes every Redis entry in one server-derived user namespace.
// Implements DESIGN-008 AccountDeleter cache-prefix erasure.
type UserPurger struct {
	client userPurgeClient
}

// userPurgeClient is the Redis command subset needed for bounded owner erasure.
// Implements DESIGN-008 AccountDeleter cache-prefix erasure.
type userPurgeClient interface {
	Scan(context.Context, uint64, string, int64) *redis.ScanCmd
	Del(context.Context, ...string) *redis.IntCmd
}

// NewUserPurger creates owner-scoped Redis erasure behavior.
// Implements DESIGN-008 AccountDeleter cache-prefix erasure.
func NewUserPurger(client *redis.Client) UserPurger {
	if client == nil {
		return UserPurger{}
	}
	return UserPurger{client: client}
}

// PurgeUser deletes the exact user key and all descendant keys in bounded batches.
// Implements DESIGN-008 AccountDeleter cache-prefix erasure.
func (p UserPurger) PurgeUser(ctx context.Context, userID uuid.UUID) error {
	if p.client == nil {
		return nil
	}
	prefix := "user:" + userID.String()
	var cursor uint64
	for {
		keys, next, err := p.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := p.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
