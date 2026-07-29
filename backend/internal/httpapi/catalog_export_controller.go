package httpapi

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
)

// GlobalCatalogExportService streams one canonical ownerless catalog document.
// Implements DESIGN-009 AdminController global catalog export.
type GlobalCatalogExportService interface {
	Write(context.Context, io.Writer, bool) (int, error)
}

// CatalogExportController handles the administration-only catalog export read.
// Implements DESIGN-009 AdminController global catalog export.
type CatalogExportController struct {
	service GlobalCatalogExportService
}

// NewGlobalCatalogExportAdminController composes the export with admin authorization.
// Implements DESIGN-009 AdminController global catalog export route.
func NewGlobalCatalogExportAdminController(service GlobalCatalogExportService) *AdminController {
	controller := &CatalogExportController{service: service}
	limit := RateLimitRule{Scope: "user", MaxRequests: 10, WindowSeconds: 60}
	return NewAdminController(nil, AdminRouteDefinition{
		Method: fiber.MethodGet, Path: "/catalog-export", Handler: controller.Export,
		Validate: validateCatalogExportQuery, RateLimit: &limit,
	})
}

// Export streams a deterministic read-only snapshot without an account-data envelope.
// Implements DESIGN-009 AdminController global/private export separation.
func (c *CatalogExportController) Export(ctx *fiber.Ctx) error {
	if c == nil || c.service == nil {
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true}
	}
	includeDeleted := ctx.Query("includeDeleted") == "true"
	streamContext := context.Background()
	cancel := func() {}
	if gateway, ok := ctx.Locals("gateway").(GatewayContext); ok {
		streamContext, cancel = context.WithDeadline(streamContext, gateway.Deadline)
	}
	reader, writer := io.Pipe()
	go func() {
		_, err := c.service.Write(streamContext, writer, includeDeleted)
		_ = writer.CloseWithError(err)
	}()
	ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	ctx.Set(fiber.HeaderContentDisposition, `attachment; filename="global-catalog.json"`)
	ctx.Context().Response.SetBodyStream(&cancelingReadCloser{ReadCloser: reader, cancel: cancel}, -1)
	return nil
}

// validateCatalogExportQuery accepts only the optional strict boolean selection.
// Implements DESIGN-009 AdminController global catalog export request boundary.
func validateCatalogExportQuery(ctx *fiber.Ctx) error {
	valid := true
	count := 0
	ctx.Context().QueryArgs().VisitAll(func(key []byte, value []byte) {
		count++
		valid = valid && string(key) == "includeDeleted" && (string(value) == "true" || string(value) == "false")
	})
	if !valid || count > 1 {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return ctx.Next()
}

// cancelingReadCloser cancels the detached snapshot deadline when streaming ends.
// Implements DESIGN-009 AdminController safe export cancellation.
type cancelingReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

// Close cancels snapshot work and closes the stream.
// Implements DESIGN-009 AdminController safe export cancellation.
func (r *cancelingReadCloser) Close() error {
	r.cancel()
	return r.ReadCloser.Close()
}
