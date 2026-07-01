package subscription

import "errors"

// ErrUsageLimitExceeded indicates that the user has exceeded their usage limit for a feature.
// Implements DESIGN-007 UsageLimiter.
var ErrUsageLimitExceeded = errors.New("usage limit exceeded for the rolling 24-hour window")

// ErrFeatureNotAllowed indicates that the user tier does not permit the requested feature.
// Implements DESIGN-007 UsageLimiter.
var ErrFeatureNotAllowed = errors.New("feature not allowed for this tier")
