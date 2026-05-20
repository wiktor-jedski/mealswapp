package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestRequestLoggerWritesStructuredSuccessLog(t *testing.T) {
	var logs bytes.Buffer
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	app := newLoggingApp(&logs, func() time.Time {
		now = now.Add(25 * time.Millisecond)
		return now
	})
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	res, err := app.Test(newRequest(t, http.MethodGet, "/ok"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	entry := decodeLogEntry(t, logs.String())
	if entry["msg"] != "http_request" || entry["method"] != http.MethodGet || entry["path"] != "/ok" {
		t.Fatalf("unexpected log entry: %#v", entry)
	}
	if entry["status"] != float64(http.StatusOK) || entry["route"] != "/ok" || entry["request_id"] == "" {
		t.Fatalf("expected status, route, and request id, got %#v", entry)
	}
	if entry["latency_ms"] != float64(25) {
		t.Fatalf("expected latency_ms 25, got %#v", entry["latency_ms"])
	}
}

func TestRequestLoggerWritesSanitizedErrorMetadata(t *testing.T) {
	var logs bytes.Buffer
	app := newLoggingApp(&logs, time.Now)
	app.Get("/fail", func(ctx *fiber.Ctx) error {
		return apperrors.Internal(errors.New("database password=super-secret"))
	})

	res, err := app.Test(newRequest(t, http.MethodGet, "/fail"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	raw := logs.String()
	if strings.Contains(raw, "super-secret") || strings.Contains(raw, "password") {
		t.Fatalf("expected sensitive error details to be omitted, got %s", raw)
	}

	entry := decodeLogEntry(t, raw)
	if entry["status"] != float64(http.StatusInternalServerError) || entry["error_code"] != "internal_error" || entry["error_category"] != "server" {
		t.Fatalf("expected sanitized internal error metadata, got %#v", entry)
	}
}

func newLoggingApp(logs *bytes.Buffer, now func() time.Time) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: func(ctx *fiber.Ctx, err error) error {
		appErr, ok := apperrors.As(err)
		if !ok {
			appErr = apperrors.Internal(err)
		}

		envelope := responses.Failure(appErr.Code, appErr.Message, requestID(ctx))
		envelope.Error.Category = string(appErr.Category)
		envelope.Error.Retryable = appErr.Retryable
		envelope.Error.Fields = appErr.Fields

		return ctx.Status(appErr.Status).JSON(envelope)
	}})
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	app.Use(requestid.New())
	app.Use(RequestLogger(RequestLoggerConfig{Logger: logger, Now: now}))
	return app
}

func decodeLogEntry(t *testing.T, raw string) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &entry); err != nil {
		t.Fatalf("decode log entry: %v; raw=%s", err, raw)
	}
	return entry
}
