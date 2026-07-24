package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/tagmanager"
)

// ClassificationRepositoryFactory scopes classification mutations to the gateway transaction.
// Implements DESIGN-009 TagManager and AdminController.
type ClassificationRepositoryFactory func(repository.AdminMutationExecutor) repository.ClassificationAdminRepository

// ClassificationCacheInvalidator discards classification-derived filter options after commit.
// Implements DESIGN-009 TagManager.
type ClassificationCacheInvalidator interface {
	Invalidate()
}

// ClassificationAdminController exposes audited global classification CRUD definitions.
// Implements DESIGN-009 TagManager.
type ClassificationAdminController struct {
	service     *tagmanager.Service
	factory     ClassificationRepositoryFactory
	validator   *CurationRequestValidator
	invalidator ClassificationCacheInvalidator
}

// NewClassificationAdminController creates the TagManager HTTP adapter.
// Implements DESIGN-009 TagManager.
func NewClassificationAdminController(service *tagmanager.Service, factory ClassificationRepositoryFactory, validator *CurationRequestValidator, invalidator ClassificationCacheInvalidator) *ClassificationAdminController {
	return &ClassificationAdminController{service: service, factory: factory, validator: validator, invalidator: invalidator}
}

// AdminRoutes returns the allowlisted admin CRUD definitions.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) AdminRoutes() []AdminRouteDefinition {
	readLimit := &RateLimitRule{Scope: "user", MaxRequests: 120, WindowSeconds: 60}
	mutationLimit := &RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60}
	return []AdminRouteDefinition{
		{Method: fiber.MethodGet, Path: "/classifications", Handler: c.List, RateLimit: readLimit},
		{Method: fiber.MethodPost, Path: "/classifications/:kind", Mutation: c.Create, Validate: c.validateCreate, RateLimit: mutationLimit, AuditAction: "classification.create", EntityType: "classification"},
		{Method: fiber.MethodPut, Path: "/classifications/:classificationId", Mutation: c.Update, Validate: c.validateUpdate, RateLimit: mutationLimit, AuditAction: "classification.update", EntityType: "classification"},
		{Method: fiber.MethodDelete, Path: "/classifications/:classificationId", Mutation: c.Delete, Validate: validateClassificationID, RateLimit: mutationLimit, AuditAction: "classification.delete", EntityType: "classification"},
	}
}

// validateCreate checks the closed kind and normalized body before rate limiting.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) validateCreate(ctx *fiber.Ctx) error {
	if _, err := classificationKind(ctx.Params("kind")); err != nil {
		return err
	}
	return c.validator.ValidateClassificationBody(ctx)
}

// validateUpdate checks identity and normalized body before rate limiting.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) validateUpdate(ctx *fiber.Ctx) error {
	if _, err := classificationID(ctx); err != nil {
		return err
	}
	return c.validator.ValidateClassificationBody(ctx)
}

// List returns a deterministic hierarchy for one supported kind.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) List(ctx *fiber.Ctx) error {
	kind, err := classificationKind(ctx.Query("kind"))
	if err != nil {
		return err
	}
	items, err := c.service.List(ctx.UserContext(), kind)
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"classifications": items}})
}

// Create persists one normalized classification inside the audit transaction.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) Create(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	kind, err := classificationKind(ctx.Params("kind"))
	if err != nil {
		return AdminMutationResult{}, err
	}
	req, ok := NormalizedCurationClassificationRequest(ctx)
	if !ok {
		return AdminMutationResult{}, curationValidationError()
	}
	created, err := c.service.Create(ctx.UserContext(), c.factory(tx), repository.ClassificationEntity{Name: req.Name, Kind: kind, ParentID: req.ParentID})
	if err != nil {
		return AdminMutationResult{}, err
	}
	after, err := classificationAuditJSON(created, true, false)
	if err != nil {
		return AdminMutationResult{}, err
	}
	return AdminMutationResult{HTTPStatus: fiber.StatusCreated, Data: map[string]any{"classification": created}, Audit: repository.AdminAuditChanges{EntityID: &created.ID, After: after}, AfterCommit: c.invalidate}, nil
}

// Update atomically persists and audits a normalized rename or reparent operation.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) Update(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	id, err := classificationID(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	req, ok := NormalizedCurationClassificationRequest(ctx)
	if !ok {
		return AdminMutationResult{}, curationValidationError()
	}
	before, after, err := c.service.Update(ctx.UserContext(), c.factory(tx), id, req.Name, req.ParentID)
	if err != nil {
		return AdminMutationResult{}, err
	}
	beforeJSON, err := classificationAuditJSON(before, true, false)
	if err != nil {
		return AdminMutationResult{}, err
	}
	afterJSON, err := classificationAuditJSON(after, true, false)
	if err != nil {
		return AdminMutationResult{}, err
	}
	return AdminMutationResult{Data: map[string]any{"classification": after}, Audit: repository.AdminAuditChanges{EntityID: &after.ID, Before: beforeJSON, After: afterJSON}, AfterCommit: c.invalidate}, nil
}

// Delete atomically soft-deletes and audits one unused classification.
// Implements DESIGN-009 TagManager.
func (c *ClassificationAdminController) Delete(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	id, err := classificationID(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	before, err := c.service.Delete(ctx.UserContext(), c.factory(tx), id)
	if err != nil {
		return AdminMutationResult{}, err
	}
	beforeJSON, err := classificationAuditJSON(before, true, false)
	if err != nil {
		return AdminMutationResult{}, err
	}
	afterJSON, err := classificationAuditJSON(before, false, true)
	if err != nil {
		return AdminMutationResult{}, err
	}
	return AdminMutationResult{HTTPStatus: fiber.StatusNoContent, Audit: repository.AdminAuditChanges{EntityID: &id, Before: beforeJSON, After: afterJSON}, AfterCommit: c.invalidate}, nil
}

// classificationAuditJSON records bounded change evidence without persisting administrator-authored labels.
// Implements DESIGN-009 TagManager privacy-safe before/after audit snapshots.
func classificationAuditJSON(classification repository.ClassificationEntity, active bool, deleted bool) ([]byte, error) {
	digest := sha256.Sum256([]byte(classification.Name))
	snapshot := map[string]any{"kind": classification.Kind, "nameDigest": fmt.Sprintf("%x", digest), "active": active, "deleted": deleted}
	if classification.ParentID != nil {
		snapshot["parentId"] = classification.ParentID.String()
	}
	return json.Marshal(snapshot)
}

// validateClassificationID rejects malformed path identities before rate limiting and mutation.
// Implements DESIGN-009 TagManager.
func validateClassificationID(ctx *fiber.Ctx) error {
	if _, err := classificationID(ctx); err != nil {
		return err
	}
	return ctx.Next()
}

// classificationID parses the required route identity.
// Implements DESIGN-009 TagManager.
func classificationID(ctx *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Params("classificationId"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, curationValidationError()
	}
	return id, nil
}

// classificationKind closes administration over Food Categories and Culinary Roles.
// Implements DESIGN-009 TagManager.
func classificationKind(value string) (repository.ClassificationKind, error) {
	kind := repository.ClassificationKind(value)
	if kind != repository.ClassificationKindFoodCategory && kind != repository.ClassificationKindCulinaryRole {
		return "", curationValidationError()
	}
	return kind, nil
}

// invalidate runs only after mutation and audit commit.
// Implements DESIGN-009 TagManager cache invalidation.
func (c *ClassificationAdminController) invalidate() {
	if c.invalidator != nil {
		c.invalidator.Invalidate()
	}
}
