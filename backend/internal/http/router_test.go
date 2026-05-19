package http

import (
	"bytes"
	"encoding/json"
	"io"
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestHealthRoute(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config: config.Config{Environment: "test"},
	})

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var payload responses.Envelope
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data envelope, got %#v", payload)
	}

	if data["status"] != "ok" {
		t.Fatalf("expected ok status, got %#v", data["status"])
	}

	if !payload.Success {
		t.Fatal("expected success envelope")
	}
}

func TestVersionedRoutesAndEnvelopes(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config: config.Config{Environment: "test"},
	})

	cases := []struct {
		path       string
		wantStatus string
	}{
		{path: "/health", wantStatus: "ok"},
		{path: "/ready", wantStatus: "ready"},
		{path: "/api/v1/health", wantStatus: "ok"},
		{path: "/api/v1/ready", wantStatus: "ready"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			res := performRequest(t, app, http.MethodGet, tc.path)
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", res.StatusCode)
			}

			payload := decodeEnvelope(t, res)
			if !payload.Success || payload.Error != nil {
				t.Fatalf("expected success envelope, got %#v", payload)
			}
			if payload.Meta == nil || payload.Meta.RequestID == "" {
				t.Fatalf("expected request id metadata, got %#v", payload.Meta)
			}

			data, ok := payload.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected data object, got %#v", payload.Data)
			}
			if data["status"] != tc.wantStatus {
				t.Fatalf("expected status %q, got %#v", tc.wantStatus, data["status"])
			}
		})
	}
}

func TestNotFoundEnvelope(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config: config.Config{Environment: "test"},
	})

	res := performRequest(t, app, http.MethodGet, "/api/v1/missing")
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}

	payload := decodeEnvelope(t, res)
	if payload.Success {
		t.Fatal("expected failure envelope")
	}
	if payload.Error == nil || payload.Error.Code != "not_found" {
		t.Fatalf("expected not_found error, got %#v", payload.Error)
	}
	if payload.Meta == nil || payload.Meta.RequestID == "" || payload.Error.RequestID == "" {
		t.Fatalf("expected request IDs in error envelope, got %#v", payload)
	}
}

func TestGatewayContextIsAttached(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(gatewayContextMiddleware(10 * time.Second))
	app.Get("/context-check", func(ctx *fiber.Ctx) error {
		gatewayContext := ExtractGatewayContext(ctx)
		if gatewayContext.RequestID == "" || gatewayContext.StartedAt.IsZero() || gatewayContext.Deadline.IsZero() {
			return fiber.NewError(fiber.StatusInternalServerError, "missing gateway context")
		}
		return ctx.JSON(responses.Success(map[string]string{"status": "ok"}, gatewayContext.RequestID))
	})

	res := performRequest(t, app, http.MethodGet, "/context-check")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
}

func TestValidationErrorsUseHTTPEnvelope(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Use(requestid.New())
	app.Post("/items/:id", func(ctx *fiber.Ctx) error {
		if _, err := validation.UUIDParam(ctx, "id"); err != nil {
			return err
		}
		payload, err := validation.DecodeJSON[struct {
			Name string `json:"name"`
		}](ctx)
		if err != nil {
			return err
		}
		if err := validation.Merge(validation.RequiredString("name", payload.Name)); err != nil {
			return err
		}
		if _, err := validation.PaginationFromQuery(ctx); err != nil {
			return err
		}
		return ctx.JSON(responses.Success(map[string]string{"status": "ok"}, requestID(ctx)))
	})

	cases := []struct {
		name string
		path string
		body []byte
		code string
	}{
		{name: "invalid path param", path: "/items/not-a-uuid", body: []byte(`{"name":"ok"}`), code: "invalid_uuid"},
		{name: "malformed json", path: "/items/00000000-0000-4000-8000-000000000001", body: []byte(`{"name":`), code: "malformed_json"},
		{name: "missing required field", path: "/items/00000000-0000-4000-8000-000000000001", body: []byte(`{"name":" "}`), code: "required"},
		{name: "pagination bounds", path: "/items/00000000-0000-4000-8000-000000000001?pageSize=11", body: []byte(`{"name":"ok"}`), code: "too_large"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.path, bytes.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)

			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", res.StatusCode)
			}

			payload := decodeEnvelope(t, res)
			if payload.Error == nil || payload.Error.Code != "validation_error" {
				t.Fatalf("expected validation_error envelope, got %#v", payload)
			}
			fields, ok := payload.Error.Fields.([]any)
			if !ok || len(fields) == 0 {
				t.Fatalf("expected field errors, got %#v", payload.Error.Fields)
			}
			first, ok := fields[0].(map[string]any)
			if !ok || first["code"] != tc.code {
				t.Fatalf("expected field code %q, got %#v", tc.code, fields[0])
			}
		})
	}
}

func performRequest(t *testing.T, app *fiber.App, method string, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

func decodeEnvelope(t *testing.T, res *http.Response) responses.Envelope {
	t.Helper()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var payload responses.Envelope
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response %q: %v", string(body), err)
	}

	return payload
}
