package middleware

import (
	"mealswapp/backend/internal/http/apperrors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type RateLimitKeyScope string

const (
	RateLimitKeyIP       RateLimitKeyScope = "ip"
	RateLimitKeyUser     RateLimitKeyScope = "user"
	RateLimitKeyEndpoint RateLimitKeyScope = "endpoint"
)

type RateLimitRule struct {
	Name        string
	PathPrefix  string
	KeyScope    RateLimitKeyScope
	MaxRequests int
	Window      time.Duration
}

type RateLimiterConfig struct {
	Rules        []RateLimitRule
	ExemptPaths  []string
	Now          func() time.Time
	KeyLocalName string
}

type rateLimitBucket struct {
	Count   int
	ResetAt time.Time
}

type rateLimiterState struct {
	mu      sync.Mutex
	buckets map[string]rateLimitBucket
}

func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Rules: []RateLimitRule{
			{Name: "auth-login", PathPrefix: "/api/v1/auth/login", KeyScope: RateLimitKeyIP, MaxRequests: 10, Window: 10 * time.Minute},
			{Name: "search", PathPrefix: "/api/v1/search", KeyScope: RateLimitKeyIP, MaxRequests: 60, Window: time.Minute},
			{Name: "admin", PathPrefix: "/api/v1/admin", KeyScope: RateLimitKeyUser, MaxRequests: 120, Window: time.Minute},
			{Name: "webhooks", PathPrefix: "/api/v1/webhooks", KeyScope: RateLimitKeyEndpoint, MaxRequests: 300, Window: time.Minute},
			{Name: "api-default", PathPrefix: "/api/v1", KeyScope: RateLimitKeyIP, MaxRequests: 300, Window: time.Minute},
		},
		ExemptPaths: []string{"/health", "/ready", "/api/v1/health", "/api/v1/ready"},
	}
}

func RateLimiter(config RateLimiterConfig) fiber.Handler {
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.KeyLocalName == "" {
		config.KeyLocalName = "userID"
	}

	state := &rateLimiterState{buckets: make(map[string]rateLimitBucket)}

	return func(ctx *fiber.Ctx) error {
		path := ctx.Path()
		if pathIsExempt(path, config.ExemptPaths) {
			return ctx.Next()
		}

		rule, ok := matchingRateLimitRule(path, config.Rules)
		if !ok || rule.MaxRequests <= 0 || rule.Window <= 0 {
			return ctx.Next()
		}

		now := config.Now()
		key := rateLimitKey(ctx, rule, config.KeyLocalName)
		allowed, retryAfter := state.allow(key, rule, now)
		if allowed {
			return ctx.Next()
		}

		retryAfterSeconds := int(retryAfter.Seconds())
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}
		ctx.Set(fiber.HeaderRetryAfter, strconv.Itoa(retryAfterSeconds))

		return apperrors.AppError{
			Category:  apperrors.CategoryDependency,
			Code:      "rate_limited",
			Message:   "Too many requests",
			Retryable: true,
			Status:    fiber.StatusTooManyRequests,
			Fields: map[string]any{
				"limit":             rule.MaxRequests,
				"windowSeconds":     int(rule.Window.Seconds()),
				"retryAfterSeconds": retryAfterSeconds,
				"rule":              rule.Name,
			},
		}
	}
}

func (state *rateLimiterState) allow(key string, rule RateLimitRule, now time.Time) (bool, time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()

	bucket, exists := state.buckets[key]
	if !exists || !now.Before(bucket.ResetAt) {
		state.buckets[key] = rateLimitBucket{Count: 1, ResetAt: now.Add(rule.Window)}
		state.deleteExpired(now)
		return true, 0
	}

	if bucket.Count >= rule.MaxRequests {
		return false, bucket.ResetAt.Sub(now)
	}

	bucket.Count++
	state.buckets[key] = bucket
	return true, 0
}

func (state *rateLimiterState) deleteExpired(now time.Time) {
	for key, bucket := range state.buckets {
		if !now.Before(bucket.ResetAt) {
			delete(state.buckets, key)
		}
	}
}

func matchingRateLimitRule(path string, rules []RateLimitRule) (RateLimitRule, bool) {
	var matched RateLimitRule
	for _, rule := range rules {
		if strings.HasPrefix(path, rule.PathPrefix) && len(rule.PathPrefix) > len(matched.PathPrefix) {
			matched = rule
		}
	}

	if matched.PathPrefix == "" {
		return RateLimitRule{}, false
	}
	return matched, true
}

func pathIsExempt(path string, exemptPaths []string) bool {
	for _, exemptPath := range exemptPaths {
		if path == exemptPath {
			return true
		}
	}
	return false
}

func rateLimitKey(ctx *fiber.Ctx, rule RateLimitRule, keyLocalName string) string {
	switch rule.KeyScope {
	case RateLimitKeyUser:
		if userID, ok := ctx.Locals(keyLocalName).(string); ok && userID != "" {
			return rule.Name + ":user:" + userID
		}
		return rule.Name + ":ip:" + ctx.IP()
	case RateLimitKeyEndpoint:
		return rule.Name + ":endpoint:" + ctx.Method() + ":" + ctx.Path()
	default:
		return rule.Name + ":ip:" + ctx.IP()
	}
}
