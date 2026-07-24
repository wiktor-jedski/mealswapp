package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dataimporter"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// CuratedImportService defines transactional curated-draft confirmation.
// Implements DESIGN-009 DataImporter admin route.
type CuratedImportService interface {
	Confirm(context.Context, repository.AdminMutationExecutor, uuid.UUID, string, dataimporter.Request) (dataimporter.Result, error)
}

// curatedImportOutcomeRecorder delays successful telemetry until audit commit.
// Implements DESIGN-014 MetricsCollector and DESIGN-009 DataImporter.
type curatedImportOutcomeRecorder interface {
	RecordCommittedOutcome(context.Context, string, dataimporter.Result)
}

// CuratedImportInvalidator makes committed catalog changes visible across cached search instances.
// Implements DESIGN-009 DataImporter immediate catalog/substitution visibility.
type CuratedImportInvalidator interface{ Invalidate() }

// CuratedImportController handles one curated import mutation.
// Implements DESIGN-009 DataImporter.
type CuratedImportController struct {
	service     CuratedImportService
	invalidator CuratedImportInvalidator
}

// NewCuratedImportAdminController composes import confirmation with the secure audited gateway.
// Implements DESIGN-009 AdminController and DataImporter.
func NewCuratedImportAdminController(audit repository.AdminMutationAuditRepository, service CuratedImportService, invalidators ...CuratedImportInvalidator) *AdminController {
	controller := &CuratedImportController{service: service}
	if len(invalidators) > 0 {
		controller.invalidator = invalidators[0]
	}
	limit := RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60}
	return NewAdminController(audit, AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/imports", Mutation: controller.Confirm, Validate: validateCuratedImport,
		RateLimit: &limit, AuditAction: "import_food", EntityType: "food_item",
	})
}

// Confirm persists or replays one validated curated draft and defers response until audit commit.
// Implements DESIGN-009 DataImporter ImportItem.
func (c *CuratedImportController) Confirm(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	admin, err := RequireAdmin(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	if c == nil || c.service == nil {
		return AdminMutationResult{}, curatedImportDependencyError()
	}
	req, ok := ctx.Locals("curatedImportRequest").(dataimporter.Request)
	if !ok {
		return AdminMutationResult{}, curationValidationError()
	}
	result, err := c.service.Confirm(ctx.UserContext(), tx, admin.UserID, ctx.Get("Idempotency-Key"), req)
	if err != nil {
		return AdminMutationResult{}, curatedImportError(err)
	}
	id := result.FoodItemID
	afterCommit := func() {
		if recorder, ok := c.service.(curatedImportOutcomeRecorder); ok {
			recorder.RecordCommittedOutcome(ctx.UserContext(), req.SourceProvider, result)
		}
		if !result.Replayed && c.invalidator != nil {
			c.invalidator.Invalidate()
		}
	}
	return AdminMutationResult{
		HTTPStatus: fiber.StatusCreated,
		Data:       map[string]any{"importId": result.ImportID, "foodItemId": result.FoodItemID, "name": result.Name, "physicalState": result.PhysicalState, "merged": result.Merged, "replayed": result.Replayed},
		Audit: func() repository.AdminAuditChanges {
			if result.Replayed {
				return repository.AdminAuditChanges{Replayed: true}
			}
			return repository.AdminAuditChanges{EntityID: &id, After: curatedImportAuditSnapshot(result)}
		}(),
		AfterCommit: afterCommit,
	}, nil
}

// validateCuratedImport strictly decodes required editable fields before transactional dispatch.
// Implements DESIGN-009 DataImporter editable-draft validation and DESIGN-010 RequestValidator.
func validateCuratedImport(ctx *fiber.Ctx) error {
	if err := rejectDuplicateJSONKeys(ctx.Body()); err != nil {
		return curationValidationError()
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(ctx.Body(), &raw); err != nil {
		return curationValidationError()
	}
	for _, field := range []string{"name", "physicalState", "macrosPer100", "micros", "foodCategoryIds", "culinaryRoleIds"} {
		value, present := raw[field]
		if !present || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return curationValidationError()
		}
	}
	if err := validateRequiredMacros(ctx.Body()); err != nil {
		return curationValidationError()
	}
	var req dataimporter.Request
	if err := decodeStrictBody(ctx.Body(), &req); err != nil {
		return curationValidationError()
	}
	normalized, err := dataimporter.NormalizeRequest(ctx.UserContext(), req)
	if err != nil {
		return curationValidationError()
	}
	ctx.Locals("curatedImportRequest", normalized)
	return ctx.Next()
}

// curatedImportAuditSnapshot emits only bounded import state, never provider/body data.
// Implements DESIGN-009 DataImporter privacy-safe audit persistence.
func curatedImportAuditSnapshot(result dataimporter.Result) []byte {
	payload, _ := json.Marshal(map[string]any{"physicalState": result.PhysicalState, "status": "imported"})
	return payload
}

// curatedImportError maps internal conflict classes into safe explicit confirmation responses.
// Implements DESIGN-009 DataImporter conflict handling and DESIGN-017 ErrorMessageMapper.
func curatedImportError(err error) error {
	switch {
	case errors.Is(err, dataimporter.ErrMissingIdempotencyKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required when provider identity is absent"}
	case errors.Is(err, dataimporter.ErrIdempotencyConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "idempotency_key_conflict", Message: "Idempotency-Key was already used with a different request body"}
	case errors.Is(err, dataimporter.ErrProviderConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "provider_identity_conflict", Message: "provider item was already imported with different curated data"}
	case errors.Is(err, dataimporter.ErrNameConfirmation):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "name_conflict_confirmation_required", Message: "an existing item with this name requires explicit confirmation"}
	case repository.IsKind(err, repository.ErrorKindValidation), repository.IsKind(err, repository.ErrorKindInvalidMicronutrientKey):
		return curationValidationError()
	default:
		return err
	}
}

// curatedImportDependencyError fails closed when import persistence is unavailable.
// Implements DESIGN-009 DataImporter fail-closed behavior.
func curatedImportDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "curated_import_unavailable", Message: "curated import service is unavailable", Retryable: true}
}
