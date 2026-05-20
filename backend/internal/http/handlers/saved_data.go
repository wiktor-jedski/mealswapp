package handlers

import (
	"context"
	"encoding/json"

	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
)

type SavedDataService interface {
	ListSavedData(ctx context.Context, accessToken string, kind string) (any, error)
	CreateSavedData(ctx context.Context, accessToken string, input SavedDataInput) (any, error)
	UpdateSavedData(ctx context.Context, accessToken string, id string, input SavedDataInput) (any, error)
	DeleteSavedData(ctx context.Context, accessToken string, id string) error
}

type SavedDataHandler struct {
	service SavedDataService
}

type SavedDataInput struct {
	Kind    string          `json:"kind"`
	Label   string          `json:"label"`
	Payload json.RawMessage `json:"payload"`
}

func NewSavedDataHandler(service SavedDataService) SavedDataHandler {
	return SavedDataHandler{service: service}
}

func (handler SavedDataHandler) List(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	items, err := handler.service.ListSavedData(ctx.Context(), token, ctx.Query("kind"))
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(items, requestID(ctx)))
}

func (handler SavedDataHandler) Create(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	input, err := decodeSavedDataInput(ctx)
	if err != nil {
		return err
	}
	item, err := handler.service.CreateSavedData(ctx.Context(), token, input)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusCreated).JSON(responses.Success(item, requestID(ctx)))
}

func (handler SavedDataHandler) Update(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	input, err := decodeSavedDataInput(ctx)
	if err != nil {
		return err
	}
	item, err := handler.service.UpdateSavedData(ctx.Context(), token, ctx.Params("id"), input)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(item, requestID(ctx)))
}

func (handler SavedDataHandler) Delete(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	if err := handler.service.DeleteSavedData(ctx.Context(), token, ctx.Params("id")); err != nil {
		return err
	}
	return ctx.JSON(responses.Success(map[string]string{"status": "deleted"}, requestID(ctx)))
}

func decodeSavedDataInput(ctx *fiber.Ctx) (SavedDataInput, error) {
	input, err := validation.DecodeJSON[SavedDataInput](ctx)
	if err != nil {
		return SavedDataInput{}, err
	}
	if err := validation.Merge(validation.RequiredString("kind", input.Kind), validation.RequiredString("label", input.Label)); err != nil {
		return SavedDataInput{}, err
	}
	if len(input.Payload) == 0 {
		input.Payload = []byte(`{}`)
	}
	return input, nil
}
