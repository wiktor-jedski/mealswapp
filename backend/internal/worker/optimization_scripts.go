// Implements DESIGN-004 JobStatusTracker embedded transition program.
package worker

import (
	_ "embed"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Embedded script lua/optimization_state_transition.lua is part of the worker
// binary and uses EVALSHA with automatic EVAL fallback after NOSCRIPT.
// Implements DESIGN-004 JobStatusTracker atomic monotonic publication.
var (
	//go:embed lua/optimization_state_transition.lua
	optimizationStateTransitionLua string

	optimizationStateTransitionScript = redis.NewScript(strings.TrimSpace(optimizationStateTransitionLua))
)
