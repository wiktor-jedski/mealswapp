package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

// ExportService defines account export behavior for HTTP handlers.
// Implements DESIGN-008 DataExporter.
type ExportService interface {
	BuildExport(context.Context, uuid.UUID, string) (userdata.ExportPayload, error)
}

// ExportController owns authenticated account export routes.
// Implements DESIGN-008 DataExporter.
type ExportController struct {
	service ExportService
}

// NewExportController creates account export handlers.
// Implements DESIGN-008 DataExporter.
func NewExportController(service ExportService) *ExportController {
	return &ExportController{service: service}
}

// Routes returns authenticated export routes.
// Implements DESIGN-008 DataExporter.
func (c *ExportController) Routes() []RouteDefinition {
	return []RouteDefinition{{Method: fiber.MethodGet, Path: "/account/export", RequiresAuth: true, Validate: ValidateQuery(validateExportQuery), Handler: c.ExportData}}
}

// ExportData returns JSON or CSV account export data.
// Implements DESIGN-008 DataExporter.
func (c *ExportController) ExportData(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	format := ctx.Query("format", "json")
	payload, err := c.service.BuildExport(ctx.UserContext(), user.UserID, format)
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
	}
	ctx.Set("Content-Type", payload.ContentType)
	ctx.Set("Content-Disposition", `attachment; filename="`+payload.Filename+`"`)
	return ctx.Send(payload.Body)
}

// validateExportQuery validates account export format before service dispatch.
// Implements DESIGN-010 RequestValidator.
func validateExportQuery(values map[string]string) error {
	format := values["format"]
	if format == "" {
		format = "json"
	}
	if _, err := security.NormalizeInput(security.InputFieldExportFormat, format); err != nil {
		return errors.New("export format is invalid")
	}
	return nil
}
