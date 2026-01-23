# SubscriptionController

**Traceability:** ARCH-007

## 1. Data Structures & Types

```go
package subscription

import (
	"time"
)

// SubscriptionTier represents the subscription tier levels
type SubscriptionTier int

const (
	TierFree SubscriptionTier = iota
	TierTrial
	TierBasic
	TierPremium
)

// SubscriptionStatus represents the current status of a subscription
type SubscriptionStatus int

const (
	StatusInactive SubscriptionStatus = iota
	StatusActive
	StatusPastDue
	StatusCanceled
	StatusTrialing
)

// UserEntitlement represents a user's subscription entitlement
type UserEntitlement struct {
	UserID             string             `json:"user_id" db:"user_id"`
	Tier               SubscriptionTier   `json:"tier" db:"tier"`
	Status             SubscriptionStatus `json:"status" db:"status"`
	StripeCustomerID   string             `json:"stripe_customer_id" db:"stripe_customer_id"`
	StripeSubscription string             `json:"stripe_subscription_id" db:"stripe_subscription_id"`
	TrialEndsAt        *time.Time         `json:"trial_ends_at" db:"trial_ends_at"`
	CurrentPeriodStart time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end" db:"current_period_end"`
	SearchesUsed       int                `json:"searches_used" db:"searches_used"`
	SearchesResetAt    time.Time          `json:"searches_reset_at" db:"searches_reset_at"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
}

// SubscriptionRequest represents a request to create or modify a subscription
type	UserID      string           SubscriptionRequest struct {
 `json:"user_id"`
	Tier        SubscriptionTier `json:"tier"`
	PaymentMethodID string       `json:"payment_method_id"`
}

// SubscriptionResponse represents the response for subscription operations
type SubscriptionResponse struct {
	Success       bool               `json:"success"`
	Entitlement   *UserEntitlement   `json:"entitlement,omitempty"`
	CheckoutURL   string             `json:"checkout_url,omitempty"`
	ErrorMessage  string             `json:"error_message,omitempty"`
}

// UsageCheckResult represents the result of a usage entitlement check
type UsageCheckResult struct {
	Allowed          bool   `json:"allowed"`
	Remaining        int    `json:"remaining"`
	ResetAt          *time.Time `json:"reset_at,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
}

// StripeWebhookEvent represents an incoming Stripe webhook event
type StripeWebhookEvent struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	CustomerID      string      `json:"customer_id"`
	SubscriptionID  string      `json:"subscription_id"`
	PaymentIntentID string      `json:"payment_intent_id"`
	Amount          int64       `json:"amount"`
	Currency        string      `json:"currency"`
	Status          string      `json:"status"`
	Created         int64       `json:"created"`
	RawPayload      []byte      `json:"-"`
}

// TrialActivationRequest represents a request to activate a trial
type TrialActivationRequest struct {
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"` // "google", "github", etc.
	ProviderID   string    `json:"provider_id"`
	Email        string    `json:"email"`
}

// TrialEligibilityCheck represents the result of checking trial eligibility
type TrialEligibilityCheck struct {
	Eligible     bool   `json:"eligible"`
	Reason       string `json:"reason,omitempty"`
	AlreadyTrial bool   `json:"already_trial"`
	HasTrialed   bool   `json:"has_trialed"`
}

// SearchUsage represents search usage data for rate limiting
type SearchUsage struct {
	UserID       string    `json:"user_id"`
	Count        int       `json:"count"`
	ResetAt      time.Time `json:"reset_at"`
}

// StripeCheckoutSession represents a Stripe checkout session
type StripeCheckoutSession struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	CustomerID    string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
	Status        string `json:"status"`
}

// ProcessedEvent represents a processed webhook event for idempotency
type ProcessedEvent struct {
	EventID     string    `db:"event_id"`
	ProcessedAt time.Time `db:"processed_at"`
	EventType   string    `db:"event_type"`
	Result      string    `db:"result"`
}

// TierConfig represents configuration for each subscription tier
type TierConfig struct {
	Tier           SubscriptionTier `json:"tier"`
	Name           string           `json:"name"`
	SearchLimit    int              `json:"search_limit"`
	Period         time.Duration    `json:"period"`
	Price          int64            `json:"price"` // in cents
	Currency       string           `json:"currency"`
	StripePriceID  string           `json:"stripe_price_id"`
	MaxItemsPerSearch int           `json:"max_items_per_search"`
	Features       []string         `json:"features"`
}

// SubscriptionController handles all subscription-related operations
type SubscriptionController struct {
	entitlementRepo EntitlementRepository
	stripeClient    StripeClient
	redisClient     *redis.Client
	logger          *log.Logger
	tierConfigs     map[SubscriptionTier]TierConfig
}

// EntitlementRepository defines the interface for entitlement data access
type EntitlementRepository interface {
	GetEntitlement(ctx context.Context, userID string) (*UserEntitlement, error)
	CreateEntitlement(ctx context.Context, entitlement *UserEntitlement) error
	UpdateEntitlement(ctx context.Context, entitlement *UserEntitlement) error
	GetTrialHistory(ctx context.Context, userID string) ([]TrialRecord, error)
	RecordTrialStart(ctx context.Context, userID string, endsAt time.Time) error
}

// StripeClient defines the interface for Stripe API operations
type StripeClient interface {
	CreateCustomer(ctx context.Context, email, name string) (*StripeCustomer, error)
	CreateCheckoutSession(ctx context.Context, req *CreateCheckoutRequest) (*StripeCheckoutSession, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*StripeSubscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	RetrievePaymentIntent(ctx context.Context, paymentIntentID string) (*StripePaymentIntent, error)
	VerifyWebhookSignature(payload []byte, signature, webhookSecret string) error
}

// StripeCustomer represents a Stripe customer
type StripeCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CreateCheckoutRequest represents parameters for creating a checkout session
type CreateCheckoutRequest struct {
	CustomerID       string
	PriceID          string
	SuccessURL       string
	CancelURL        string
	Mode             string // "subscription" or "payment"
	TrialPeriodDays  *int
	PaymentMethodID  string
}

// StripeSubscription represents a Stripe subscription
type StripeSubscription struct {
	ID               string    `json:"id"`
	CustomerID       string    `json:"customer_id"`
	Status           string    `json:"status"`
	CurrentPeriodEnd int64     `json:"current_period_end"`
	TrialEnd         *int64    `json:"trial_end"`
	CancelAtPeriodEnd bool     `json:"cancel_at_period_end"`
}

// StripePaymentIntent represents a Stripe payment intent
type StripePaymentIntent struct {
	ID              string `json:"id"`
	CustomerID      string `json:"customer_id"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	PaymentMethodID string `json:"payment_method_id"`
}

// TrialRecord represents a trial period record
type TrialRecord struct {
	UserID    string    `db:"user_id"`
	StartedAt time.Time `db:"started_at"`
	EndedAt   time.Time `db:"ended_at"`
	Completed bool      `db:"completed"`
}
```

## 2. Logic & Algorithms

### 2.1 Subscribe (Create Checkout Session)

```go
func (c *SubscriptionController) Subscribe(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResponse, error) {
	// Step 1: Retrieve current entitlement for user
	entitlement, err := c.entitlementRepo.GetEntitlement(ctx, req.UserID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return &SubscriptionResponse{
			Success:      false,
			ErrorMessage: "failed to retrieve entitlement",
		}, err
	}

	// Step 2: Validate tier transition
	if err := c.validateTierTransition(entitlement, req.Tier); err != nil {
		return &SubscriptionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Step 3: Get or create Stripe customer
	customerID := entitlement.StripeCustomerID
	if customerID == "" {
		customer, err := c.stripeClient.CreateCustomer(ctx, req.UserID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
		}
		customerID = customer.ID

		// Update entitlement with customer ID
		if entitlement != nil {
			entitlement.StripeCustomerID = customerID
			if err := c.entitlementRepo.UpdateEntitlement(ctx, entitlement); err != nil {
				return nil, fmt.Errorf("failed to update customer ID: %w", err)
			}
		}
	}

	// Step 4: Get tier configuration
	tierConfig, ok := c.tierConfigs[req.Tier]
	if !ok {
		return &SubscriptionResponse{
			Success:      false,
			ErrorMessage: "invalid tier configuration",
		}, nil
	}

	// Step 5: Create Stripe checkout session
	checkoutReq := &CreateCheckoutRequest{
		CustomerID:      customerID,
		PriceID:         tierConfig.StripePriceID,
		SuccessURL:      c.config.SuccessURL,
		CancelURL:       c.config.CancelURL,
		Mode:            "subscription",
		TrialPeriodDays: nil,
		PaymentMethodID: req.PaymentMethodID,
	}

	session, err := c.stripeClient.CreateCheckoutSession(ctx, checkoutReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &SubscriptionResponse{
		Success:     true,
		CheckoutURL: session.URL,
	}, nil
}
```

### 2.2 Handle Webhook Event

```go
func (c *SubscriptionController) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	// Step 1: Verify Stripe webhook signature
	event, err := c.stripeClient.VerifyWebhookSignature(payload, signature, c.config.WebhookSecret)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Step 2: Check idempotency - return early if already processed
	processed, err := c.checkIdempotency(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if processed {
		return nil // Already processed, return 200 OK
	}

	// Step 3: Process event based on type
	var handlerErr error
	switch event.Type {
	case "payment_intent.succeeded":
		handlerErr = c.handlePaymentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		handlerErr = c.handlePaymentFailed(ctx, event)
	case "customer.subscription.created":
		handlerErr = c.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated":
		handlerErr = c.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		handlerErr = c.handleSubscriptionDeleted(ctx, event)
	case "customer.subscription.trial_will_end":
		handlerErr = c.handleTrialWillEnd(ctx, event)
	default:
		// Log unknown event type but don't error
		c.logger.Printf("unknown webhook event type: %s", event.Type)
	}

	// Step 4: Record processed event
	if err := c.recordProcessedEvent(ctx, event.ID, event.Type, handlerErr); err != nil {
		c.logger.Printf("failed to record processed event: %v", err)
	}

	// Step 5: Return error if processing failed (triggers Stripe retry)
	if handlerErr != nil {
		return handlerErr
	}

	return nil
}
```

### 2.3 Handle Payment Succeeded

```go
func (c *SubscriptionController) handlePaymentSucceeded(ctx context.Context, event *StripeWebhookEvent) error {
	// Step 1: Retrieve payment intent to get customer ID
	paymentIntent, err := c.stripeClient.RetrievePaymentIntent(ctx, event.PaymentIntentID)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment intent: %w", err)
	}

	// Step 2: Find entitlement by Stripe customer ID
	entitlement, err := c.findEntitlementByCustomerID(ctx, paymentIntent.CustomerID)
	if err != nil {
		return fmt.Errorf("entitlement not found for customer: %w", err)
	}

	// Step 3: Start database transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 4: Update entitlement status
	entitlement.Status = StatusActive
	entitlement.UpdatedAt = time.Now()

	if err := c.entitlementRepo.UpdateEntitlement(ctx, entitlement); err != nil {
		return fmt.Errorf("failed to update entitlement: %w", err)
	}

	// Step 5: Log successful payment
	c.logger.Printf("payment succeeded for user %s, amount: %d %s",
		entitlement.UserID, event.Amount, event.Currency)

	// Step 6: Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
```

### 2.4 Check Usage Entitlement

```go
func (c *SubscriptionController) CheckEntitlement(ctx context.Context, userID string, itemsCount int) (*UsageCheckResult, error) {
	// Step 1: Get user's current entitlement
	entitlement, err := c.entitlementRepo.GetEntitlement(ctx, userID)
	if err != nil {
		return &UsageCheckResult{
			Allowed:      false,
			ErrorMessage: "failed to retrieve entitlement",
		}, err
	}

	// Step 2: Check if user has active subscription or trial
	if !c.isSubscriptionActive(entitlement) {
		// Check if free tier
		if entitlement.Tier == TierFree {
			// Check if within daily search limit
			canSearch, remaining, resetAt, err := c.checkFreeTierSearchLimit(ctx, userID)
			if err != nil {
				return &UsageCheckResult{
					Allowed:      false,
					ErrorMessage: "failed to check usage",
				}, err
			}
			if !canSearch {
				return &UsageCheckResult{
					Allowed:   false,
					Remaining: remaining,
					ResetAt:   &resetAt,
					ErrorMessage: fmt.Sprintf("daily search limit reached. Resets at %s", resetAt.Format(time.RFC3339)),
				}, nil
			}
			return &UsageCheckResult{
				Allowed:   true,
				Remaining: remaining - 1,
				ResetAt:   &resetAt,
			}, nil
		}
		return &UsageCheckResult{
			Allowed:      false,
			ErrorMessage: "subscription inactive",
		}, nil
	}

	// Step 3: Check if trial has expired
	if entitlement.Tier == TierTrial && entitlement.TrialEndsAt != nil {
		if time.Now().After(*entitlement.TrialEndsAt) {
			// Auto-downgrade to free tier
			if err := c.downgradeToFreeTier(ctx, userID); err != nil {
				c.logger.Printf("failed to downgrade trial user: %v", err)
			}
			return &UsageCheckResult{
				Allowed:      false,
				ErrorMessage: "trial period expired",
			}, nil
		}
	}

	// Step 4: Check items per search limit for free tier
	tierConfig := c.tierConfigs[entitlement.Tier]
	if entitlement.Tier == TierFree && itemsCount > tierConfig.MaxItemsPerSearch {
		return &UsageCheckResult{
			Allowed:      false,
			ErrorMessage: fmt.Sprintf("free tier limited to %d items per search", tierConfig.MaxItemsPerSearch),
		}, nil
	}

	return &UsageCheckResult{
		Allowed: true,
	}, nil
}
```

### 2.5 Check Free Tier Search Limit

```go
func (c *SubscriptionController) checkFreeTierSearchLimit(ctx context.Context, userID string) (bool, int, time.Time, error) {
	// Step 1: Get current usage from Redis
	usageKey := fmt.Sprintf("usage:search:%s:%s", userID, time.Now().Format("2006-01-02"))
	currentCount, err := c.redisClient.Get(ctx, usageKey).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, 0, time.Time{}, err
	}

	// Step 2: Calculate reset time (midnight local time)
	now := time.Now()
	resetAt := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	// Step 3: Check if within limit
	maxSearches := c.tierConfigs[TierFree].SearchLimit // 3 searches per 24h
	remaining := maxSearches - currentCount

	if currentCount >= maxSearches {
		return false, remaining, resetAt, nil
	}

	// Step 4: Increment usage counter
	newCount := currentCount + 1
	if err := c.redisClient.Set(ctx, usageKey, newCount, time.Until(resetAt)).Err(); err != nil {
		return false, 0, time.Time{}, err
	}

	return true, remaining - 1, resetAt, nil
}
```

### 2.6 Activate Trial

```go
func (c *SubscriptionController) ActivateTrial(ctx context.Context, req *TrialActivationRequest) (*TrialEligibilityCheck, error) {
	// Step 1: Check if user already has active trial
	existingEntitlement, err := c.entitlementRepo.GetEntitlement(ctx, req.UserID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return &TrialEligibilityCheck{Eligible: false}, err
	}

	if existingEntitlement != nil && existingEntitlement.Tier == TierTrial {
		if existingEntitlement.TrialEndsAt != nil && time.Now().Before(*existingEntitlement.TrialEndsAt) {
			return &TrialEligibilityCheck{
				Eligible:     false,
				AlreadyTrial: true,
				Reason:       "already has active trial",
			}, nil
		}
	}

	// Step 2: Check if user has previously used a trial
	trialHistory, err := c.entitlementRepo.GetTrialHistory(ctx, req.UserID)
	if err != nil {
		return &TrialEligibilityCheck{Eligible: false}, err
	}

	if len(trialHistory) > 0 {
		return &TrialEligibilityCheck{
			Eligible:   false,
			HasTrialed: true,
			Reason:     "trial already used",
		}, nil
	}

	// Step 3: Create or update entitlement with trial
	trialDuration := 7 * 24 * time.Hour
	trialEndsAt := time.Now().Add(trialDuration)

	if existingEntitlement == nil {
		// Create new entitlement with trial
		entitlement := &UserEntitlement{
			UserID:           req.UserID,
			Tier:             TierTrial,
			Status:           StatusTrialing,
			TrialEndsAt:      &trialEndsAt,
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   trialEndsAt,
			SearchesUsed:     0,
			SearchesResetAt:  time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+1, 0, 0, 0, 0, time.Local),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := c.entitlementRepo.CreateEntitlement(ctx, entitlement); err != nil {
			return &TrialEligibilityCheck{Eligible: false}, err
		}
	} else {
		// Update existing entitlement to trial
		existingEntitlement.Tier = TierTrial
		existingEntitlement.Status = StatusTrialing
		existingEntitlement.TrialEndsAt = &trialEndsAt
		existingEntitlement.CurrentPeriodStart = time.Now()
		existingEntitlement.CurrentPeriodEnd = trialEndsAt
		existingEntitlement.UpdatedAt = time.Now()

		if err := c.entitlementRepo.UpdateEntitlement(ctx, existingEntitlement); err != nil {
			return &TrialEligibilityCheck{Eligible: false}, err
		}
	}

	// Step 4: Record trial start for history tracking
	if err := c.entitlementRepo.RecordTrialStart(ctx, req.UserID, trialEndsAt); err != nil {
		c.logger.Printf("failed to record trial start: %v", err)
	}

	return &TrialEligibilityCheck{Eligible: true}, nil
}
```

### 2.7 Downgrade to Free Tier (Scheduled Task)

```go
func (c *SubscriptionController) ProcessExpiredTrials(ctx context.Context) error {
	// Step 1: Query for expired trials
	expiredTrials, err := c.entitlementRepo.GetExpiredTrials(ctx)
	if err != nil {
		return fmt.Errorf("failed to query expired trials: %w", err)
	}

	// Step 2: Process each expired trial
	for _, entitlement := range expiredTrials {
		// Start transaction
		tx, err := c.db.BeginTx(ctx, nil)
		if err != nil {
			c.logger.Printf("failed to start transaction for user %s: %v", entitlement.UserID, err)
			continue
		}

		// Downgrade to free tier
		entitlement.Tier = TierFree
		entitlement.Status = StatusInactive
		entitlement.TrialEndsAt = nil
		entitlement.UpdatedAt = time.Now()

		if err := c.entitlementRepo.UpdateEntitlement(ctx, entitlement); err != nil {
			tx.Rollback()
			c.logger.Printf("failed to downgrade user %s: %v", entitlement.UserID, err)
			continue
		}

		// Log the downgrade
		c.logger.Printf("downgraded user %s from trial to free tier", entitlement.UserID)

		if err := tx.Commit(); err != nil {
			c.logger.Printf("failed to commit downgrade for user %s: %v", entitlement.UserID, err)
		}
	}

	return nil
}
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Cause | Handling Strategy |
|-------------|-------|-------------------|
| `StripeTimeout` | Stripe API request times out | Retry with exponential backoff up to 3 attempts |
| `StripeSignatureInvalid` | Webhook signature verification fails | Return 400, log potential attack |
| `DuplicateWebhookEvent` | Same webhook event received twice | Idempotent check returns 200 OK without reprocessing |
| `EntitlementNotFound` | User has no entitlement record | Create default free tier entitlement |
| `DatabaseTransactionFailure` | Database transaction fails | Rollback, log to dead-letter queue, return 500 |
| `RedisConnectionFailure` | Redis unavailable for usage tracking | Fallback to database-based tracking |
| `PaymentIntentNotFound` | Payment intent doesn't exist | Log error, investigate with reconciliation job |
| `TrialAlreadyUsed` | User has previously used trial | Return error, prevent duplicate trial |
| `InvalidTierTransition` | Attempting invalid tier upgrade/downgrade | Validate before processing, return error |
| `SubscriptionCanceled` | Stripe subscription canceled | Update local entitlement to inactive |

### 3.2 State Transitions

```
┌──────────────┐
│   New User   │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ Free Tier (Default)  │──── On social login + eligibility check ────> Trial (7 days)
│ searches: 3/24h      │
│ items: 1 per search  │
└──────────────────────┘
       │
       │ On successful subscription payment
       ▼
┌──────────────────────┐
│   Paid Subscription  │──── On payment failure ────> Past Due
│ searches: unlimited  │                           │
│ items: unlimited     │──── On cancel ───────────> Canceled
│                     │                           │
└──────────────────────┘                           │
       │                                          │
       │ On trial expiration ─────────────────────┘
       ▼
┌──────────────────────┐
│   Free Tier (Downgrade)
│ searches: 3/24h
│ items: 1 per search
└──────────────────────┘
```

### 3.3 Reconciliation Job

```go
func (c *SubscriptionController) RunReconciliation(ctx context.Context) error {
	// Step 1: Query local entitlements with active subscriptions
	localSubs, err := c.entitlementRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to query local subscriptions: %w", err)
	}

	// Step 2: For each local subscription, verify with Stripe
	for _, entitlement := range localSubs {
		if entitlement.StripeSubscription == "" {
			continue
		}

		stripeSub, err := c.stripeClient.GetSubscription(ctx, entitlement.StripeSubscription)
		if err != nil {
			c.logger.Printf("failed to get Stripe subscription %s: %v",
				entitlement.StripeSubscription, err)
			continue
		}

		// Step 3: Fix discrepancy if statuses don't match
		localActive := c.isSubscriptionActive(entitlement)
		remoteActive := stripeSub.Status == "active" || stripeSub.Status == "trialing"

		if localActive && !remoteActive {
			// Local says active, Stripe says inactive - fix local
			if err := c.fixEntitlementStatus(ctx, entitlement, stripeSub.Status); err != nil {
				c.logger.Printf("failed to fix entitlement for user %s: %v",
					entitlement.UserID, err)
			}
		} else if !localActive && remoteActive {
			// Local says inactive, Stripe says active - fix local
			if err := c.fixEntitlementStatus(ctx, entitlement, stripeSub.Status); err != nil {
				c.logger.Printf("failed to fix entitlement for user %s: %v",
					entitlement.UserID, err)
			}
		}
	}

	return nil
}
```

### 3.4 Dead Letter Queue Handling

```go
func (c *SubscriptionController) handleDeadLetter(ctx context.Context, event *FailedWebhookEvent) error {
	// Step 1: Log the failed event
	c.logger.Printf("processing dead letter event: %s for user %s",
		event.EventID, event.UserID)

	// Step 2: Attempt to retry processing
	retryErr := c.HandleWebhook(ctx, event.Payload, event.Signature)
	if retryErr != nil {
		// Still failing after retry - escalate
		c.logger.Printf("dead letter retry failed for event %s: %v",
			event.EventID, retryErr)
		return c.sendEscalationAlert(ctx, event)
	}

	// Step 3: Mark as resolved
	if err := c.markDeadLetterResolved(ctx, event.ID); err != nil {
		c.logger.Printf("failed to mark dead letter resolved: %v", err)
	}

	return nil
}
```

## 4. Component Interfaces

### 4.1 Public Methods

```go
// Subscribe creates a Stripe checkout session for subscription
func (c *SubscriptionController) Subscribe(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResponse, error)

// HandleWebhook processes incoming Stripe webhook events
func (c *SubscriptionController) HandleWebhook(ctx context.Context, payload []byte, signature string) error

// CheckEntitlement checks if a user can perform an action based on their subscription
func (c *SubscriptionController) CheckEntitlement(ctx context.Context, userID string, itemsCount int) (*UsageCheckResult, error)

// GetEntitlement retrieves the current entitlement for a user
func (c *SubscriptionController) GetEntitlement(ctx context.Context, userID string) (*UserEntitlement, error)

// ActivateTrial activates a trial period for a new user
func (c *SubscriptionController) ActivateTrial(ctx context.Context, req *TrialActivationRequest) (*TrialEligibilityCheck, error)

// CancelSubscription cancels a user's subscription
func (c *SubscriptionController) CancelSubscription(ctx context.Context, userID string) error

// GetSubscriptionStatus returns the current subscription status
func (c *SubscriptionController) GetSubscriptionStatus(ctx context.Context, userID string) (*SubscriptionStatusResponse, error)
```

### 4.2 Internal Helper Methods

```go
// validateTierTransition validates if a tier transition is allowed
func (c *SubscriptionController) validateTierTransition(current *UserEntitlement, newTier SubscriptionTier) error

// checkIdempotency checks if an event has already been processed
func (c *SubscriptionController) checkIdempotency(ctx context.Context, eventID string) (bool, error)

// recordProcessedEvent records a processed webhook event
func (c *SubscriptionController) recordProcessedEvent(ctx context.Context, eventID, eventType string, processingErr error) error

// findEntitlementByCustomerID finds an entitlement by Stripe customer ID
func (c *SubscriptionController) findEntitlementByCustomerID(ctx context.Context, customerID string) (*UserEntitlement, error)

// handlePaymentSucceeded handles successful payment events
func (c *SubscriptionController) handlePaymentSucceeded(ctx context.Context, event *StripeWebhookEvent) error

// handlePaymentFailed handles failed payment events
func (c *SubscriptionController) handlePaymentFailed(ctx context.Context, event *StripeWebhookEvent) error

// handleSubscriptionCreated handles subscription creation events
func (c *SubscriptionController) handleSubscriptionCreated(ctx context.Context, event *StripeWebhookEvent) error

// handleSubscriptionUpdated handles subscription update events
func (c *SubscriptionController) handleSubscriptionUpdated(ctx context.Context, event *StripeWebhookEvent) error

// handleSubscriptionDeleted handles subscription deletion events
func (c *SubscriptionController) handleSubscriptionDeleted(ctx context.Context, event *StripeWebhookEvent) error

// handleTrialWillEnd handles trial ending warning events
func (c *SubscriptionController) handleTrialWillEnd(ctx context.Context, event *StripeWebhookEvent) error

// checkFreeTierSearchLimit checks if user is within free tier search limits
func (c *SubscriptionController) checkFreeTierSearchLimit(ctx context.Context, userID string) (bool, int, time.Time, error)

// isSubscriptionActive checks if a subscription is currently active
func (c *SubscriptionController) isSubscriptionActive(entitlement *UserEntitlement) bool

// downgradeToFreeTier downgrades a user to free tier
func (c *SubscriptionController) downgradeToFreeTier(ctx context.Context, userID string) error

// ProcessExpiredTrials processes trials that have expired
func (c *SubscriptionController) ProcessExpiredTrials(ctx context.Context) error

// RunReconciliation reconciles local entitlements with Stripe
func (c *SubscriptionController) RunReconciliation(ctx context.Context) error

// fixEntitlementStatus fixes mismatched entitlement status
func (c *SubscriptionController) fixEntitlementStatus(ctx context.Context, entitlement *UserEntitlement, stripeStatus string) error
```

### 4.3 Configuration

```go
type SubscriptionConfig struct {
	// Stripe configuration
	StripeAPIKey        string
	WebhookSecret       string
	SuccessURL          string
	CancelURL           string

	// Trial configuration
	TrialDurationDays   int

	// Free tier configuration
	FreeTierSearchLimit int
	FreeTierItemsLimit  int

	// Redis configuration
	RedisAddr           string
	RedisPassword       string
	RedisDB             int

	// Database configuration
	DatabaseURL         string
}
```
