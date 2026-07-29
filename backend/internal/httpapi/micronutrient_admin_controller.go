package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/micronutrient"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-005 MicronutrientVocabulary typed validated request handoff.
const micronutrientRequestLocal = "admin.micronutrient.request"

// micronutrientRequest is the closed mutation request projection.
// Implements DESIGN-005 MicronutrientVocabulary HTTP contract.
type micronutrientRequest struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Unit        string `json:"unit"`
}

// MicronutrientRepositoryFactory scopes vocabulary mutations to the gateway audit transaction.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 AdminController.
type MicronutrientRepositoryFactory func(repository.AdminMutationExecutor) repository.MicronutrientVocabularyAdminRepository

// MicronutrientAdminController exposes the canonical vocabulary lifecycle without hard delete.
// Implements DESIGN-005 MicronutrientVocabulary administrator management.
type MicronutrientAdminController struct {
	service *micronutrient.Service
	factory MicronutrientRepositoryFactory
}

// NewMicronutrientAdminController creates the vocabulary HTTP adapter.
// Implements DESIGN-005 MicronutrientVocabulary administrator management.
func NewMicronutrientAdminController(service *micronutrient.Service, factory MicronutrientRepositoryFactory) *MicronutrientAdminController {
	return &MicronutrientAdminController{service: service, factory: factory}
}

// AdminRoutes returns the explicit vocabulary route allowlist.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 AdminController.
func (c *MicronutrientAdminController) AdminRoutes() []AdminRouteDefinition {
	readLimit := &RateLimitRule{Scope: "user", MaxRequests: 120, WindowSeconds: 60}
	mutationLimit := &RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60}
	return []AdminRouteDefinition{
		{Method: fiber.MethodGet, Path: "/micronutrients", Handler: c.List, RateLimit: readLimit},
		{Method: fiber.MethodPost, Path: "/micronutrients", Mutation: c.Create, Validate: validateMicronutrientCreate, RateLimit: mutationLimit, AuditAction: "micronutrient.create", EntityType: "micronutrient_vocabulary"},
		{Method: fiber.MethodPut, Path: "/micronutrients/:key/display-name", Mutation: c.UpdateDisplayName, Validate: validateMicronutrientDisplayName, RateLimit: mutationLimit, AuditAction: "micronutrient.display_name.update", EntityType: "micronutrient_vocabulary"},
		{Method: fiber.MethodPut, Path: "/micronutrients/:key/unit", Mutation: c.UpdateUnit, Validate: validateMicronutrientUnit, RateLimit: mutationLimit, AuditAction: "micronutrient.unit.update", EntityType: "micronutrient_vocabulary"},
		{Method: fiber.MethodPost, Path: "/micronutrients/:key/deactivate", Mutation: c.Deactivate, Validate: validateMicronutrientKey, RateLimit: mutationLimit, AuditAction: "micronutrient.deactivate", EntityType: "micronutrient_vocabulary"},
		{Method: fiber.MethodPost, Path: "/micronutrients/:key/reactivate", Mutation: c.Reactivate, Validate: validateMicronutrientKey, RateLimit: mutationLimit, AuditAction: "micronutrient.reactivate", EntityType: "micronutrient_vocabulary"},
	}
}

// List returns deterministic active and inactive authoritative state.
// Implements DESIGN-005 MicronutrientVocabulary administrator listing.
func (c *MicronutrientAdminController) List(ctx *fiber.Ctx) error {
	entries, err := c.service.List(ctx.UserContext())
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"micronutrients": entries}})
}

// Create inserts one canonical entry.
// Implements DESIGN-005 MicronutrientVocabulary retry-safe creation.
func (c *MicronutrientAdminController) Create(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	req, ok := ctx.Locals(micronutrientRequestLocal).(micronutrientRequest)
	if !ok {
		return AdminMutationResult{}, curationValidationError()
	}
	entry, err := c.service.Create(ctx.UserContext(), c.factory(tx), repository.MicronutrientVocabularyEntry{Key: req.Key, DisplayName: req.DisplayName, Unit: req.Unit})
	if err != nil {
		return AdminMutationResult{}, err
	}
	after, err := micronutrientAuditJSON(entry)
	if err != nil {
		return AdminMutationResult{}, err
	}
	return AdminMutationResult{HTTPStatus: fiber.StatusCreated, Data: map[string]any{"micronutrient": entry}, Audit: repository.AdminAuditChanges{After: after}}, nil
}

// UpdateDisplayName atomically changes and audits only the human-readable label.
// Implements DESIGN-005 MicronutrientVocabulary immutable key update.
func (c *MicronutrientAdminController) UpdateDisplayName(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	req := ctx.Locals(micronutrientRequestLocal).(micronutrientRequest)
	return c.update(ctx, tx, func(repo repository.MicronutrientVocabularyAdminRepository) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
		return c.service.UpdateDisplayName(ctx.UserContext(), repo, ctx.Params("key"), req.DisplayName)
	})
}

// UpdateUnit atomically changes and audits the unit of an unused entry.
// Implements DESIGN-005 MicronutrientVocabulary in-use safeguard.
func (c *MicronutrientAdminController) UpdateUnit(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	req := ctx.Locals(micronutrientRequestLocal).(micronutrientRequest)
	return c.update(ctx, tx, func(repo repository.MicronutrientVocabularyAdminRepository) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
		return c.service.UpdateUnit(ctx.UserContext(), repo, ctx.Params("key"), req.Unit)
	})
}

// Deactivate atomically disables and audits one unused canonical entry.
// Implements DESIGN-005 MicronutrientVocabulary deactivate lifecycle.
func (c *MicronutrientAdminController) Deactivate(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	return c.update(ctx, tx, func(repo repository.MicronutrientVocabularyAdminRepository) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
		return c.service.SetActive(ctx.UserContext(), repo, ctx.Params("key"), false)
	})
}

// Reactivate atomically restores and audits one canonical entry.
// Implements DESIGN-005 MicronutrientVocabulary reactivate lifecycle.
func (c *MicronutrientAdminController) Reactivate(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	return c.update(ctx, tx, func(repo repository.MicronutrientVocabularyAdminRepository) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
		return c.service.SetActive(ctx.UserContext(), repo, ctx.Params("key"), true)
	})
}

// update applies one vocabulary change and returns bounded audit snapshots.
// Implements DESIGN-005 MicronutrientVocabulary transactional audit behavior.
func (c *MicronutrientAdminController) update(ctx *fiber.Ctx, tx repository.AdminMutationExecutor, change func(repository.MicronutrientVocabularyAdminRepository) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error)) (AdminMutationResult, error) {
	before, after, err := change(c.factory(tx))
	if err != nil {
		return AdminMutationResult{}, err
	}
	beforeJSON, err := micronutrientAuditJSON(before)
	if err != nil {
		return AdminMutationResult{}, err
	}
	afterJSON, err := micronutrientAuditJSON(after)
	if err != nil {
		return AdminMutationResult{}, err
	}
	return AdminMutationResult{Data: map[string]any{"micronutrient": after}, Audit: repository.AdminAuditChanges{Before: beforeJSON, After: afterJSON}}, nil
}

// validateMicronutrientCreate strictly validates a create body before dispatch.
// Implements DESIGN-005 MicronutrientVocabulary HTTP validation.
func validateMicronutrientCreate(ctx *fiber.Ctx) error {
	var req micronutrientRequest
	if decodeStrictBody(ctx.Body(), &req) != nil || micronutrient.ValidateKey(req.Key) != nil || micronutrient.ValidateDisplayName(req.DisplayName) != nil || micronutrient.ValidateUnit(req.Unit) != nil {
		return curationValidationError()
	}
	ctx.Locals(micronutrientRequestLocal, req)
	return ctx.Next()
}

// validateMicronutrientDisplayName strictly validates a label-only update.
// Implements DESIGN-005 MicronutrientVocabulary HTTP validation.
func validateMicronutrientDisplayName(ctx *fiber.Ctx) error {
	var body struct {
		DisplayName string `json:"displayName"`
	}
	if decodeStrictBody(ctx.Body(), &body) != nil || micronutrient.ValidateKey(ctx.Params("key")) != nil || micronutrient.ValidateDisplayName(body.DisplayName) != nil {
		return curationValidationError()
	}
	ctx.Locals(micronutrientRequestLocal, micronutrientRequest{DisplayName: body.DisplayName})
	return ctx.Next()
}

// validateMicronutrientUnit strictly validates a unit-only update.
// Implements DESIGN-005 MicronutrientVocabulary HTTP validation.
func validateMicronutrientUnit(ctx *fiber.Ctx) error {
	var body struct {
		Unit string `json:"unit"`
	}
	if decodeStrictBody(ctx.Body(), &body) != nil || micronutrient.ValidateKey(ctx.Params("key")) != nil || micronutrient.ValidateUnit(body.Unit) != nil {
		return curationValidationError()
	}
	ctx.Locals(micronutrientRequestLocal, micronutrientRequest{Unit: body.Unit})
	return ctx.Next()
}

// validateMicronutrientKey validates lifecycle paths and requires an empty body.
// Implements DESIGN-005 MicronutrientVocabulary HTTP validation.
func validateMicronutrientKey(ctx *fiber.Ctx) error {
	if micronutrient.ValidateKey(ctx.Params("key")) != nil || len(ctx.Body()) != 0 {
		return curationValidationError()
	}
	return ctx.Next()
}

// micronutrientAuditJSON stores bounded integrity evidence without administrator-authored labels.
// Implements DESIGN-005 MicronutrientVocabulary transactional audit behavior.
func micronutrientAuditJSON(entry repository.MicronutrientVocabularyEntry) ([]byte, error) {
	digest := sha256.Sum256([]byte(entry.Key))
	return json.Marshal(map[string]any{"keyDigest": fmt.Sprintf("%x", digest), "unit": entry.Unit, "active": entry.Active})
}
