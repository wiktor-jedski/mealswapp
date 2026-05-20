package handlers

import (
	"context"
	"strings"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
)

type ProfileService interface {
	GetProfile(ctx context.Context, accessToken string) (Profile, error)
	UpdateProfile(ctx context.Context, accessToken string, update ProfileUpdate) (Profile, error)
}

type ProfileHandler struct {
	service ProfileService
}

type Profile struct {
	ID              string         `json:"id"`
	Email           string         `json:"email"`
	EmailVerified   bool           `json:"emailVerified"`
	DisplayName     string         `json:"displayName"`
	DietarySettings map[string]any `json:"dietarySettings,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type ProfileUpdate struct {
	DisplayName     *string        `json:"displayName,omitempty"`
	DietarySettings map[string]any `json:"dietarySettings,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type profileUpdateRequest struct {
	DisplayName     *string        `json:"displayName"`
	DietarySettings map[string]any `json:"dietarySettings"`
	Metadata        map[string]any `json:"metadata"`
}

func NewProfileHandler(service ProfileService) ProfileHandler {
	return ProfileHandler{service: service}
}

func (handler ProfileHandler) Get(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}

	profile, err := handler.service.GetProfile(ctx.Context(), token)
	if err != nil {
		return err
	}

	return ctx.JSON(responses.Success(profile, requestID(ctx)))
}

func (handler ProfileHandler) Update(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}

	payload, err := validation.DecodeJSON[profileUpdateRequest](ctx)
	if err != nil {
		return err
	}
	if payload.DisplayName != nil {
		trimmed := strings.TrimSpace(*payload.DisplayName)
		payload.DisplayName = &trimmed
		if err := validation.Merge(validation.RequiredString("displayName", trimmed)); err != nil {
			return err
		}
	}

	profile, err := handler.service.UpdateProfile(ctx.Context(), token, ProfileUpdate{
		DisplayName:     payload.DisplayName,
		DietarySettings: payload.DietarySettings,
		Metadata:        payload.Metadata,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(responses.Success(profile, requestID(ctx)))
}

func requiredBearerToken(ctx *fiber.Ctx) (string, error) {
	token := bearerToken(ctx)
	if token == "" {
		return "", apperrors.Unauthorized("Unauthorized")
	}
	return token, nil
}
