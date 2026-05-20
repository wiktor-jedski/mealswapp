package handlers

import (
	"context"

	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
)

type AccountFlowService interface {
	RequestPasswordReset(ctx context.Context, email string) error
	ConfirmPasswordReset(ctx context.Context, token string, newPassword string) error
	RequestEmailVerification(ctx context.Context, accessToken string) error
	ConfirmEmailVerification(ctx context.Context, token string) error
}

type AccountFlowHandler struct {
	service AccountFlowService
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type emailVerificationConfirmRequest struct {
	Token string `json:"token"`
}

func NewAccountFlowHandler(service AccountFlowService) AccountFlowHandler {
	return AccountFlowHandler{service: service}
}

func (handler AccountFlowHandler) RequestPasswordReset(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[passwordResetRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("email", payload.Email)); err != nil {
		return err
	}
	if err := handler.service.RequestPasswordReset(ctx.Context(), payload.Email); err != nil {
		return err
	}
	return ctx.JSON(responses.Success(map[string]string{"status": "accepted"}, requestID(ctx)))
}

func (handler AccountFlowHandler) ConfirmPasswordReset(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[passwordResetConfirmRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("token", payload.Token), validation.RequiredString("newPassword", payload.NewPassword)); err != nil {
		return err
	}
	if err := handler.service.ConfirmPasswordReset(ctx.Context(), payload.Token, payload.NewPassword); err != nil {
		return err
	}
	return ctx.JSON(responses.Success(map[string]string{"status": "password_reset"}, requestID(ctx)))
}

func (handler AccountFlowHandler) RequestEmailVerification(ctx *fiber.Ctx) error {
	if err := handler.service.RequestEmailVerification(ctx.Context(), bearerToken(ctx)); err != nil {
		return err
	}
	return ctx.JSON(responses.Success(map[string]string{"status": "accepted"}, requestID(ctx)))
}

func (handler AccountFlowHandler) ConfirmEmailVerification(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[emailVerificationConfirmRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("token", payload.Token)); err != nil {
		return err
	}
	if err := handler.service.ConfirmEmailVerification(ctx.Context(), payload.Token); err != nil {
		return err
	}
	return ctx.JSON(responses.Success(map[string]string{"status": "email_verified"}, requestID(ctx)))
}
