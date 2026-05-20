package handlers

import (
	"context"

	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
)

type OAuthService interface {
	StartOAuth(ctx context.Context, provider string, state string) (string, error)
	CompleteOAuth(ctx context.Context, provider string, state string, code string) (any, error)
}

type OAuthHandler struct {
	service OAuthService
}

type oauthStartRequest struct {
	State string `json:"state"`
}

type oauthCallbackRequest struct {
	State string `json:"state"`
	Code  string `json:"code"`
}

func NewOAuthHandler(service OAuthService) OAuthHandler {
	return OAuthHandler{service: service}
}

func (handler OAuthHandler) Start(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[oauthStartRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("state", payload.State)); err != nil {
		return err
	}

	authURL, err := handler.service.StartOAuth(ctx.Context(), ctx.Params("provider"), payload.State)
	if err != nil {
		return err
	}

	return ctx.JSON(responses.Success(map[string]string{"authUrl": authURL}, requestID(ctx)))
}

func (handler OAuthHandler) Callback(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[oauthCallbackRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("state", payload.State), validation.RequiredString("code", payload.Code)); err != nil {
		return err
	}

	result, err := handler.service.CompleteOAuth(ctx.Context(), ctx.Params("provider"), payload.State, payload.Code)
	if err != nil {
		return err
	}

	return ctx.JSON(responses.Success(result, requestID(ctx)))
}
