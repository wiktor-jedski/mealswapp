package handlers

import (
	"context"

	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"mealswapp/backend/internal/services/entitlements"

	"github.com/gofiber/fiber/v2"
)

type SubscriptionService interface {
	GetStatus(ctx context.Context, accessToken string) (SubscriptionStatus, error)
	CreateCheckout(ctx context.Context, accessToken string, request CheckoutRequest) (CheckoutSession, error)
	CreateCustomerPortal(ctx context.Context, accessToken string, returnURL string) (CustomerPortalSession, error)
	GetEntitlement(ctx context.Context, accessToken string) (entitlements.Entitlement, error)
}

type SubscriptionHandler struct {
	service SubscriptionService
}

type SubscriptionStatus struct {
	Entitlement  entitlements.Entitlement `json:"entitlement"`
	BillingState string                   `json:"billingState"`
	Plans        []entitlements.Plan      `json:"plans,omitempty"`
}

type CheckoutRequest struct {
	PriceID    string `json:"priceId"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

type CheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type CustomerPortalSession struct {
	URL string `json:"url"`
}

type customerPortalRequest struct {
	ReturnURL string `json:"returnUrl"`
}

func NewSubscriptionHandler(service SubscriptionService) SubscriptionHandler {
	return SubscriptionHandler{service: service}
}

func (handler SubscriptionHandler) Status(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	status, err := handler.service.GetStatus(ctx.Context(), token)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(status, requestID(ctx)))
}

func (handler SubscriptionHandler) CreateCheckout(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	payload, err := validation.DecodeJSON[CheckoutRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(
		validation.RequiredString("priceId", payload.PriceID),
		validation.RequiredString("successUrl", payload.SuccessURL),
		validation.RequiredString("cancelUrl", payload.CancelURL),
	); err != nil {
		return err
	}
	session, err := handler.service.CreateCheckout(ctx.Context(), token, payload)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusCreated).JSON(responses.Success(session, requestID(ctx)))
}

func (handler SubscriptionHandler) CreateCustomerPortal(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	payload, err := validation.DecodeJSON[customerPortalRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("returnUrl", payload.ReturnURL)); err != nil {
		return err
	}
	session, err := handler.service.CreateCustomerPortal(ctx.Context(), token, payload.ReturnURL)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(session, requestID(ctx)))
}

func (handler SubscriptionHandler) Entitlement(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	entitlement, err := handler.service.GetEntitlement(ctx.Context(), token)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(entitlement, requestID(ctx)))
}
