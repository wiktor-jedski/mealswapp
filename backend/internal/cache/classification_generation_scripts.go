// Implements DESIGN-011 RedisCache embedded guarded-write program.
package cache

import (
	_ "embed"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Embedded script lua/set_if_classification_generation.lua is part of the cache
// binary and uses EVALSHA with automatic EVAL fallback after NOSCRIPT.
// Implements DESIGN-011 RedisCache guarded cache-miss persistence.
var (
	//go:embed lua/set_if_classification_generation.lua
	setIfClassificationGenerationLua string

	setIfClassificationGenerationScript = redis.NewScript(strings.TrimSpace(setIfClassificationGenerationLua))
)
