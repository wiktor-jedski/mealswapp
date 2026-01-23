# FILE: StripeWebhookHandler.md
**Traceability:** ARCH-007

## 1. Data Structures & Types

```go
package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type WebhookEventType string

const (
	EventPaymentIntentSucceeded WebhookEventType = "payment_intent.succeeded"
	EventPaymentIntentFailed    WebhookEventType = "payment_intent.failed"
	EventCustomerSubscriptionCreated WebhookEventType = "customer.subscription.created"
	EventCustomerSubscriptionUpdated WebhookEventType = "customer.subscription.updated"
	EventCustomerSubscriptionDeleted WebhookEventType = "customer.subscription.deleted"
)

type StripeWebhookPayload struct {
	ID              string              `json:"id"`
	Type            WebhookEventType    `json:"type"`
	Created         int64               `json:"created"`
	Data            StripeEventData     `json:"data"`
	Livemode        bool                `json:"livemode"`
	PendingWebhooks int                 `json:"pending_webhooks"`
	Request         StripeRequestObject `json:"request"`
}

type StripeEventData struct {
	Object  map[string]interface{} `json:"object"`
	PreviousAttributes map[string]interface{} `json:"previous_attributes,omitempty"`
}

type StripeRequestObject struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type PaymentIntentObject struct {
	ID               string  `json:"id"`
	Amount           int     `json:"amount"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	CustomerID       string  `json:"customer"`
	Metadata         map[string]string `json:"metadata"`
	Created          int64   `json:"created"`
	PaymentMethodID  string  `json:"payment_method"`
}

type SubscriptionObject struct {
	ID               string    `json:"id"`
	CustomerID       string    `json:"customer"`
	Status           string    `json:"status"`
	CurrentPeriodStart int64   `json:"current_period_start"`
	CurrentPeriodEnd   int64   `json:"current_period_end"`
	TrialStart       *int64    `json:"trial_start,omitempty"`
	TrialEnd         *int64    `json:"trial_end,omitempty"`
	Metadata         map[string]string `json:"metadata"`
	CancelAtPeriodEnd bool     `json:"cancel_at_period_end"`
	CanceledAt       *int64    `json:"canceled_at,omitempty"`
	EndedAt          *int64    `json:"ended_at,omitempty"`
}

type WebhookHandlerConfig struct {
	WebhookSecret        string
	StripeClient         StripeClientInterface
	EventRepository      EventRepositoryInterface
	EntitlementService   EntitlementServiceInterface
	Logger               LoggerInterface
	DeadLetterQueue      DeadLetterQueueInterface
}

type StripeClientInterface interface {
	VerifySignature(payload []byte, signature string, secret string) error
	RetrieveEvent(eventID string) (*StripeWebhookPayload, error)
	RetrievePaymentIntent(paymentIntentID string) (*PaymentIntentObject, error)
	RetrieveSubscription(subscriptionID string) (*SubscriptionObject, error)
}

type EventRepositoryInterface interface {
	IsEventProcessed(eventID string) (bool, error)
	MarkEventProcessed(eventID string, eventType WebhookEventType, payload []byte) error
	UpdateEntitlement(userID string, tier string, periodStart time.Time, periodEnd time.Time) error
	GetUserByStripeCustomerID(customerID string) (string, error)
}

type EntitlementServiceInterface interface {
	GrantTrialAccess(userID string, duration time.Duration) error
	UpgradeToPaid(userID string, subscriptionID string) error
	DowngradeToFree(userID string) error
	CancelSubscription(userID string) error
}

type LoggerInterface interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

type DeadLetterQueueInterface interface {
	Enqueue(payload []byte, errorReason string, metadata map[string]string) error
}

type WebhookProcessingResult struct {
	Success       bool
	EventID       string
	EventType     WebhookEventType
	ErrorMessage  string
	Retryable     bool
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Main Handler Entry Point

```
FUNCTION HandleStripeWebhook(c *fiber.Ctx) -> error
    1. GET raw body from request (preserve exact bytes)
    2. GET Stripe-Signature header from request
    3. EXTRACT webhook secret from config
    4. CALL VerifyWebhookSignature(rawBody, signature, secret)
       IF error THEN
           LOG warning "Invalid webhook signature"
           RETURN fiber.NewError(fiber.StatusBadRequest, "Invalid signature")
       END IF
    5. PARSE rawBody into StripeWebhookPayload
       IF parse error THEN
           LOG error "Failed to parse webhook payload"
           RETURN fiber.NewError(fiber.StatusBadRequest, "Invalid payload")
       END IF
    6. CALL ProcessWebhookEvent(payload)
    7. RETURN appropriate HTTP status based on result
END FUNCTION
```

### 2.2 Signature Verification

```
FUNCTION VerifyWebhookSignature(payload []byte, signature string, secret string) -> error
    1. IF signature is empty THEN
           RETURN error "Missing Stripe-Signature header"
       END IF
    2. USE Stripe library to verify signature with timestamp tolerance (5 minutes)
    3. RETURN nil on success, error on failure
END FUNCTION
```

### 2.3 Event Processing Main Loop

```
FUNCTION ProcessWebhookEvent(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. SET eventID = payload.ID
    2. SET eventType = payload.Type
    3. LOG info "Processing webhook event", "event_id", eventID, "type", eventType
    
    4. CALL CheckIdempotency(eventID)
       IF event was already processed THEN
           LOG info "Duplicate event detected, skipping", "event_id", eventID
           RETURN &WebhookProcessingResult{Success: true, EventID: eventID, EventType: eventType}
       END IF
    
    5. MARK event as processed in database (BEGIN TRANSACTION)
       IF mark fails THEN
           RETURN &WebhookProcessingResult{
               Success: false,
               EventID: eventID,
               EventType: eventType,
               ErrorMessage: "Failed to mark event as processed",
               Retryable: false
           }
       END IF
    
    6. SWITCH ON eventType
       CASE EventPaymentIntentSucceeded:
           CALL HandlePaymentIntentSucceeded(payload)
       CASE EventPaymentIntentFailed:
           CALL HandlePaymentIntentFailed(payload)
       CASE EventCustomerSubscriptionCreated:
           CALL HandleSubscriptionCreated(payload)
       CASE EventCustomerSubscriptionUpdated:
           CALL HandleSubscriptionUpdated(payload)
       CASE EventCustomerSubscriptionDeleted:
           CALL HandleSubscriptionDeleted(payload)
       DEFAULT:
           LOG warn "Unhandled event type", "type", eventType
           RETURN &WebhookProcessingResult{Success: true, EventID: eventID, EventType: eventType}
    END SWITCH
    
    7. RETURN result from handler
END FUNCTION
```

### 2.4 Idempotency Check

```
FUNCTION CheckIdempotency(eventID string) -> (bool, error)
    1. CALL EventRepository.IsEventProcessed(eventID)
    2. RETURN processed, error
END FUNCTION
```

### 2.5 Payment Intent Succeeded Handler

```
FUNCTION HandlePaymentIntentSucceeded(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. EXTRACT paymentIntent from payload.Data.Object
    2. GET customerID from paymentIntent.Customer
    3. CALL RetrieveUserByCustomerID(customerID)
       IF user not found THEN
           RETURN &WebhookProcessingResult{
               Success: false,
               EventID: payload.ID,
               EventType: payload.Type,
               ErrorMessage: "User not found for customer",
               Retryable: false
           }
       END IF
    4. BEGIN DATABASE TRANSACTION
    5. CALL UpdateEntitlement(userID, "paid", now(), now().Add(30 days))
       IF update fails THEN
           ROLLBACK TRANSACTION
           CALL DeadLetterQueue.Enqueue(payload, error, {"handler": "payment_intent_succeeded"})
           RETURN &WebhookProcessingResult{
               Success: false,
               EventID: payload.ID,
               EventType: payload.Type,
               ErrorMessage: error.Error(),
               Retryable: true
           }
       END IF
    6. COMMIT TRANSACTION
    7. LOG info "Payment successful, entitlement upgraded", "user_id", userID
    8. RETURN success result
END FUNCTION
```

### 2.6 Payment Intent Failed Handler

```
FUNCTION HandlePaymentIntentFailed(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. EXTRACT paymentIntent from payload.Data.Object
    2. GET customerID from paymentIntent.Customer
    3. GET failureMessage from paymentIntent.LastPaymentError.Message (if present)
    4. CALL RetrieveUserByCustomerID(customerID)
       IF user found THEN
           LOG warn "Payment failed for user", "user_id", userID, "error", failureMessage
           OPTIONALLY: Send notification email to user
       END IF
    5. RETURN success result (Stripe should not retry failed payments)
END FUNCTION
```

### 2.7 Subscription Created Handler

```
FUNCTION HandleSubscriptionCreated(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. EXTRACT subscription from payload.Data.Object
    2. GET customerID from subscription.Customer
    3. GET userID by customerID
       IF user not found THEN
           RETURN non-retryable error
       END IF
    4. PARSE periodStart and periodEnd as timestamps
    5. BEGIN TRANSACTION
    6. CALL UpdateEntitlement(userID, "paid", periodStart, periodEnd)
       IF update fails THEN
           ROLLBACK
           CALL DeadLetterQueue.Enqueue(payload, error, {"handler": "subscription_created"})
           RETURN retryable error
       END IF
    7. IF subscription has trial period THEN
           SET trialEnd = subscription.TrialEnd
           CALL UpdateUserTrialStatus(userID, trialEnd)
       END IF
    8. COMMIT TRANSACTION
    9. LOG info "Subscription created", "user_id", userID, "subscription_id", subscription.ID
    10. RETURN success
END FUNCTION
```

### 2.8 Subscription Updated Handler

```
FUNCTION HandleSubscriptionUpdated(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. EXTRACT subscription from payload.Data.Object
    2. GET customerID from subscription.Customer
    3. GET userID by customerID
    4. GET newStatus = subscription.Status
    5. SWITCH ON newStatus
       CASE "active":
           CALL HandleActiveSubscription(userID, subscription)
       CASE "past_due":
           CALL HandlePastDueSubscription(userID, subscription)
       CASE "canceled":
           CALL HandleCanceledSubscription(userID, subscription)
       CASE "unpaid":
           CALL HandleUnpaidSubscription(userID, subscription)
    END SWITCH
    10. RETURN result
END FUNCTION
```

### 2.9 Subscription Deleted Handler

```
FUNCTION HandleSubscriptionDeleted(payload *StripeWebhookPayload) -> *WebhookProcessingResult
    1. EXTRACT subscription from payload.Data.Object
    2. GET customerID from subscription.Customer
    3. GET userID by customerID
    4. BEGIN TRANSACTION
    5. IF subscription.CanceledAt is set THEN
           CALL ScheduleDowngrade(userID, subscription.CurrentPeriodEnd)
           Update entitlement to reflect scheduled downgrade
       ELSE
           CALL DowngradeToFree(userID)
       END IF
    6. COMMIT TRANSACTION
    7. LOG info "Subscription deleted, downgraded to free", "user_id", userID
    8. RETURN success
END FUNCTION
```

### 2.10 Active Subscription Handler

```
FUNCTION HandleActiveSubscription(userID string, subscription *SubscriptionObject) -> *WebhookProcessingResult
    1. BEGIN TRANSACTION
    2. UPDATE entitlement with new period dates
    3. IF user was in trial THEN
           Mark trial as completed
       END IF
    4. COMMIT TRANSACTION
    5. RETURN success
END FUNCTION
```

### 2.11 Past Due Subscription Handler

```
FUNCTION HandlePastDueSubscription(userID string, subscription *SubscriptionObject) -> *WebhookProcessingResult
    1. LOG warn "Subscription is past due", "user_id", userID
    2. SEND notification to user about payment issue
    3. Keep current entitlement (grace period)
    4. RETURN success (don't revoke access yet)
END FUNCTION
```

### 2.12 Canceled Subscription Handler

```
FUNCTION HandleCanceledSubscription(userID string, subscription *SubscriptionObject) -> *WebhookProcessingResult
    1. IF subscription.CancelAtPeriodEnd THEN
           Schedule downgrade at CurrentPeriodEnd
       ELSE
           CALL DowngradeToFree(userID) immediately
       END IF
    2. RETURN success
END FUNCTION
```

### 2.13 Unpaid Subscription Handler

```
FUNCTION HandleUnpaidSubscription(userID string, subscription *SubscriptionObject) -> *WebhookProcessingResult
    1. LOG warn "Subscription is unpaid, revoking access", "user_id", userID
    2. CALL DowngradeToFree(userID)
    3. RETURN success
END FUNCTION
```

### 2.14 Dead Letter Queue Handler

```
FUNCTION EnqueueToDeadLetter(payload *StripeWebhookPayload, err error, handlerName string)
    1. SERIALIZE full payload to JSON
    2. CREATE metadata map with:
       - event_id: payload.ID
       - event_type: payload.Type
       - handler_name: handlerName
       - error_message: err.Error()
       - timestamp: now()
    3. CALL DeadLetterQueue.Enqueue(serializedPayload, err.Error(), metadata)
    4. LOG error "Event sent to dead letter queue", "event_id", payload.ID
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Cause | Handling Strategy |
|-------------|-------|-------------------|
| Invalid Signature | Spoofed request, wrong secret | Return 400, do not retry |
| Missing Signature | Client didn't send header | Return 400, do not retry |
| Parse Error | Malformed JSON payload | Return 400, do not retry |
| Duplicate Event | Stripe retry or replay attack | Return 200, skip processing |
| Database Connection Failed | DB unavailable | Return 500, trigger Stripe retry |
| Transaction Rollback | Constraint violation, deadlock | Return 500, trigger Stripe retry |
| User Not Found | Stale customer reference | Return 200, log for investigation |
| Stripe API Error | Network issue, rate limit | Return 500, trigger Stripe retry |
| Dead Letter Queue Full | Queue capacity exceeded | Log and continue |
| Entitlement Update Failed | Data corruption, invalid state | Return 500, trigger Stripe retry |

### 3.2 State Transitions

```
[Initial] --valid signature--> [SignatureVerified] --valid payload--> [PayloadParsed]
                                      |                                        |
                                      |                                        v
                               [InvalidSignature]                    [IdempotencyCheck]
                                                                     /              \
                                                            [AlreadyProcessed] [NotProcessed]
                                                                                     |
                                                                                     v
                                                                          [EventProcessing]
                                                                                     |
                                      +--------------------------------------------------------+
                                      |                                                        |
                   [PaymentSucceededHandler]         [SubscriptionCreatedHandler]        ...
                                      |                                                        |
                                      v                                                        v
                           [DB Transaction Begin]                              [DB Transaction Begin]
                                      |                                                        |
                           [Entitlement Update]                                [Entitlement Update]
                                      |                                                        |
                     +----------------------------+                            +----------------------------+
                     |                            |                            |                            |
              [Update Success]             [Update Failure]               [Update Success]           [Update Failure]
                     |                            |                            |                            |
                     v                            v                            v                            v
               [Commit Txn]              [Rollback Txn]                  [Commit Txn]            [Rollback Txn]
                     |                            |                            |                            |
                     v                            v                            v                            v
              [Return 200 OK]          [Enqueue to DLQ]                 [Return 200 OK]        [Enqueue to DLQ]
                                      [Return 500]                                              [Return 500]
```

### 3.3 Retry Policy Awareness

- Stripe retries webhooks for 3 days with exponential backoff (25s, 5m, 30m, 3h, 10h, 20h)
- Handler must be fully idempotent to handle retries safely
- Return 2xx only after successful processing
- Return 4xx for client errors (Stripe won't retry)
- Return 5xx for server errors (Stripe will retry)

### 3.4 Idempotency Guarantees

- Every event is marked as processed BEFORE any side effects
- Database transaction ensures atomicity of event tracking + entitlement update
- If transaction fails, event is NOT marked as processed (will be retried)
- If transaction succeeds, event is marked processed (won't be reprocessed)

## 4. Component Interfaces

### 4.1 Main Handler Function

```go
// HandleStripeWebhook processes incoming Stripe webhook requests
// Returns fiber error with appropriate HTTP status code
func (h *StripeWebhookHandler) HandleStripeWebhook(c *fiber.Ctx) error
```

### 4.2 Signature Verification

```go
// VerifyWebhookSignature validates the Stripe-Signature header
// Returns error if signature is invalid, expired, or missing
func (h *StripeWebhookHandler) VerifyWebhookSignature(payload []byte, signature string) error
```

### 4.3 Event Processing

```go
// ProcessWebhookEvent orchestrates the processing of a single webhook event
// Returns WebhookProcessingResult indicating success/failure and retryability
func (h *StripeWebhookHandler) ProcessWebhookEvent(payload *StripeWebhookPayload) *WebhookProcessingResult
```

### 4.4 Payment Intent Handlers

```go
// HandlePaymentIntentSucceeded processes successful payment events
func (h *StripeWebhookHandler) HandlePaymentIntentSucceeded(payload *StripeWebhookPayload) *WebhookProcessingResult

// HandlePaymentIntentFailed processes failed payment events
func (h *StripeWebhookHandler) HandlePaymentIntentFailed(payload *StripeWebhookPayload) *WebhookProcessingResult
```

### 4.5 Subscription Handlers

```go
// HandleSubscriptionCreated processes new subscription creation events
func (h *StripeWebhookHandler) HandleSubscriptionCreated(payload *StripeWebhookPayload) *WebhookProcessingResult

// HandleSubscriptionUpdated processes subscription update events
func (h *StripeWebhookHandler) HandleSubscriptionUpdated(payload *StripeWebhookPayload) *WebhookProcessingResult

// HandleSubscriptionDeleted processes subscription deletion/cancellation events
func (h *StripeWebhookHandler) HandleSubscriptionDeleted(payload *StripeWebhookPayload) *WebhookProcessingResult
```

### 4.6 Helper Functions

```go
// CheckIdempotency checks if an event has already been processed
func (h *StripeWebhookHandler) CheckIdempotency(eventID string) (bool, error)

// GetUserByStripeCustomerID retrieves the local user ID from Stripe customer ID
func (h *StripeWebhookHandler) GetUserByStripeCustomerID(customerID string) (string, error)

// UpdateEntitlement wraps the entitlement update in a transaction
func (h *StripeWebhookHandler) UpdateEntitlement(userID string, tier string, periodStart, periodEnd time.Time) error

// EnqueueToDeadLetter sends failed events to the dead letter queue
func (h *StripeWebhookHandler) EnqueueToDeadLetter(payload *StripeWebhookPayload, err error, handlerName string)

// HandleActiveSubscription processes active subscription status
func (h *StripeWebhookHandler) HandleActiveSubscription(userID string, sub *SubscriptionObject) *WebhookProcessingResult

// HandlePastDueSubscription processes past_due subscription status
func (h *StripeWebhookHandler) HandlePastDueSubscription(userID string, sub *SubscriptionObject) *WebhookProcessingResult

// HandleCanceledSubscription processes canceled subscription status
func (h *StripeWebhookHandler) HandleCanceledSubscription(userID string, sub *SubscriptionObject) *WebhookProcessingResult

// HandleUnpaidSubscription processes unpaid subscription status
func (h *StripeWebhookHandler) HandleUnpaidSubscription(userID string, sub *SubscriptionObject) *WebhookProcessingResult
```

### 4.7 Configuration

```go
// NewStripeWebhookHandler creates a new handler with the given configuration
func NewStripeWebhookHandler(config WebhookHandlerConfig) *StripeWebhookHandler

// WebhookHandlerConfig holds all dependencies for the webhook handler
type WebhookHandlerConfig struct {
	WebhookSecret       string
	StripeClient        StripeClientInterface
	EventRepository     EventRepositoryInterface
	EntitlementService  EntitlementServiceInterface
	Logger              LoggerInterface
	DeadLetterQueue     DeadLetterQueueInterface
}
```

### 4.8 Fiber Route Registration

```go
// RegisterRoutes registers the webhook handler on the Fiber app
func (h *StripeWebhookHandler) RegisterRoutes(app *fiber.App, path string) {
	app.Post(path, h.HandleStripeWebhook)
}
```
