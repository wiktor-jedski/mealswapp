package validation

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type testPayload struct {
	Name string `json:"name"`
}

func TestDecodeJSONRejectsMalformedJSON(t *testing.T) {
	app := fiber.New()
	var gotErr error
	app.Post("/", func(ctx *fiber.Ctx) error {
		_, gotErr = DecodeJSON[testPayload](ctx)
		return nil
	})

	_, err := app.Test(fiberPostRequest(t, "/", []byte(`{"name":`)))
	if err != nil {
		t.Fatal(err)
	}
	if gotErr == nil {
		t.Fatal("expected app test to return validation error")
	}
	var validationErr ValidationError
	if !errors.As(gotErr, &validationErr) {
		t.Fatalf("expected validation error, got %v", gotErr)
	}
	if validationErr.Fields[0].Code != "malformed_json" {
		t.Fatalf("expected malformed_json, got %#v", validationErr.Fields)
	}
}

func TestRequiredString(t *testing.T) {
	errs := RequiredString("name", " ")
	if len(errs) != 1 || errs[0].Code != "required" {
		t.Fatalf("expected required field error, got %#v", errs)
	}
}

func TestUUIDParam(t *testing.T) {
	app := fiber.New()
	want := uuid.New()
	app.Get("/items/:id", func(ctx *fiber.Ctx) error {
		got, err := UUIDParam(ctx, "id")
		if err != nil {
			return err
		}
		if got != want {
			t.Fatalf("expected %s, got %s", want, got)
		}
		return nil
	})

	if _, err := app.Test(fiberGetRequest(t, "/items/"+want.String())); err != nil {
		t.Fatal(err)
	}
}

func TestPaginationFromQuery(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		pagination, err := PaginationFromQuery(ctx)
		if err != nil {
			return err
		}
		if pagination.Page != 2 || pagination.PageSize != 5 || pagination.Offset != 5 || pagination.Limit != 5 {
			t.Fatalf("unexpected pagination: %#v", pagination)
		}
		return nil
	})

	if _, err := app.Test(fiberGetRequest(t, "/?page=2&pageSize=5")); err != nil {
		t.Fatal(err)
	}
}

func TestPaginationRejectsInvalidQueryParamsAndBounds(t *testing.T) {
	cases := []struct {
		path string
		code string
	}{
		{path: "/?page=nope", code: "invalid_integer"},
		{path: "/?page=0", code: "too_small"},
		{path: "/?pageSize=11", code: "too_large"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			app := fiber.New()
			var gotErr error
			app.Get("/", func(ctx *fiber.Ctx) error {
				_, gotErr = PaginationFromQuery(ctx)
				return nil
			})

			_, err := app.Test(fiberGetRequest(t, tc.path))
			if err != nil {
				t.Fatal(err)
			}
			if gotErr == nil {
				t.Fatal("expected validation error")
			}
			var validationErr ValidationError
			if !errors.As(gotErr, &validationErr) {
				t.Fatalf("expected validation error, got %v", gotErr)
			}
			if validationErr.Fields[0].Code != tc.code {
				t.Fatalf("expected %s, got %#v", tc.code, validationErr.Fields)
			}
		})
	}
}

func fiberGetRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func fiberPostRequest(t *testing.T, path string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	return req
}
