package queue

import (
	_ "embed"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Embedded scripts are part of the worker binary and use redis.Script's
// EVALSHA-first execution with automatic EVAL fallback after NOSCRIPT.
// Implements DESIGN-004 JobQueueManager binary Lua traceability.
var (
	//go:embed lua/enqueue.lua
	enqueueLua string
	//go:embed lua/finalize.lua
	finalizeLua string
	//go:embed lua/count_attempt.lua
	countAttemptLua string
	//go:embed lua/remove_delivery.lua
	removeDeliveryLua string
	//go:embed lua/release_lock.lua
	releaseLockLua string

	enqueueScript        = redis.NewScript(strings.TrimSpace(enqueueLua))
	finalizeScript       = redis.NewScript(strings.TrimSpace(finalizeLua))
	countAttemptScript   = redis.NewScript(strings.TrimSpace(countAttemptLua))
	removeDeliveryScript = redis.NewScript(strings.TrimSpace(removeDeliveryLua))
	releaseLockScript    = redis.NewScript(strings.TrimSpace(releaseLockLua))
)
