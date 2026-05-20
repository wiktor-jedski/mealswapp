package handlers

import (
	"context"
	"strings"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, command RegisterCommand) (AuthResult, error)
	Login(ctx context.Context, command LoginCommand) (AuthResult, error)
	Logout(ctx context.Context, command LogoutCommand) error
	Refresh(ctx context.Context, command RefreshCommand) (SessionTokens, error)
	CurrentUser(ctx context.Context, command CurrentUserCommand) (AuthUser, error)
}

type AuthHandler struct {
	service       AuthService
	cookieManager AuthCookieManager
}

type RegisterCommand struct {
	Email                      string
	Password                   string
	DisplayName                string
	AcceptPrivacyPolicy        bool
	AcceptTerms                bool
	AcceptNutritionDisclaimer  bool
	PrivacyPolicyVersion       string
	TermsVersion               string
	NutritionDisclaimerVersion string
	IPAddress                  string
	UserAgent                  string
}

type LoginCommand struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LogoutCommand struct {
	AccessToken  string
	RefreshToken string
}

type RefreshCommand struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type CurrentUserCommand struct {
	AccessToken string
}

type AuthResult struct {
	User   AuthUser      `json:"user"`
	Tokens SessionTokens `json:"tokens"`
}

type AuthUser struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"displayName"`
	EmailVerified bool      `json:"emailVerified"`
	Role          string    `json:"role"`
}

type SessionTokens struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

type AuthCookieManager interface {
	SetAuthCookies(ctx *fiber.Ctx, accessToken string, refreshToken string, accessExpiresAt time.Time, refreshExpiresAt time.Time)
	ClearAuthCookies(ctx *fiber.Ctx)
}

type registerRequest struct {
	Email                      string `json:"email"`
	Password                   string `json:"password"`
	DisplayName                string `json:"displayName"`
	AcceptPrivacyPolicy        bool   `json:"acceptPrivacyPolicy"`
	AcceptTerms                bool   `json:"acceptTerms"`
	AcceptNutritionDisclaimer  bool   `json:"acceptNutritionDisclaimer"`
	PrivacyPolicyVersion       string `json:"privacyPolicyVersion"`
	TermsVersion               string `json:"termsVersion"`
	NutritionDisclaimerVersion string `json:"nutritionDisclaimerVersion"`
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func NewAuthHandler(service AuthService, cookieManager AuthCookieManager) AuthHandler {
	return AuthHandler{service: service, cookieManager: cookieManager}
}

func (handler AuthHandler) Register(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[registerRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(
		validation.RequiredString("email", payload.Email),
		validation.RequiredString("password", payload.Password),
		validation.RequiredString("privacyPolicyVersion", payload.PrivacyPolicyVersion),
		validation.RequiredString("termsVersion", payload.TermsVersion),
		validation.RequiredString("nutritionDisclaimerVersion", payload.NutritionDisclaimerVersion),
	); err != nil {
		return err
	}

	result, err := handler.service.Register(ctx.Context(), RegisterCommand{
		Email:                      strings.TrimSpace(payload.Email),
		Password:                   payload.Password,
		DisplayName:                strings.TrimSpace(payload.DisplayName),
		AcceptPrivacyPolicy:        payload.AcceptPrivacyPolicy,
		AcceptTerms:                payload.AcceptTerms,
		AcceptNutritionDisclaimer:  payload.AcceptNutritionDisclaimer,
		PrivacyPolicyVersion:       payload.PrivacyPolicyVersion,
		TermsVersion:               payload.TermsVersion,
		NutritionDisclaimerVersion: payload.NutritionDisclaimerVersion,
		IPAddress:                  ctx.IP(),
		UserAgent:                  ctx.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		return err
	}
	handler.setAuthCookies(ctx, result.Tokens)

	return ctx.Status(fiber.StatusCreated).JSON(responses.Success(result, requestID(ctx)))
}

func (handler AuthHandler) Login(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[credentialsRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("email", payload.Email), validation.RequiredString("password", payload.Password)); err != nil {
		return err
	}

	result, err := handler.service.Login(ctx.Context(), LoginCommand{
		Email:     strings.TrimSpace(payload.Email),
		Password:  payload.Password,
		IPAddress: ctx.IP(),
		UserAgent: ctx.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		return err
	}
	handler.setAuthCookies(ctx, result.Tokens)

	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AuthHandler) Logout(ctx *fiber.Ctx) error {
	payload, _ := validation.DecodeJSON[refreshRequest](ctx)
	if err := handler.service.Logout(ctx.Context(), LogoutCommand{
		AccessToken:  bearerToken(ctx),
		RefreshToken: payload.RefreshToken,
	}); err != nil {
		return err
	}
	if handler.cookieManager != nil {
		handler.cookieManager.ClearAuthCookies(ctx)
	}

	return ctx.JSON(responses.Success(map[string]string{"status": "logged_out"}, requestID(ctx)))
}

func (handler AuthHandler) Refresh(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[refreshRequest](ctx)
	if err != nil {
		return err
	}
	if err := validation.Merge(validation.RequiredString("refreshToken", payload.RefreshToken)); err != nil {
		return err
	}

	tokens, err := handler.service.Refresh(ctx.Context(), RefreshCommand{
		RefreshToken: payload.RefreshToken,
		IPAddress:    ctx.IP(),
		UserAgent:    ctx.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		return err
	}
	handler.setAuthCookies(ctx, tokens)

	return ctx.JSON(responses.Success(tokens, requestID(ctx)))
}

func (handler AuthHandler) CurrentUser(ctx *fiber.Ctx) error {
	token := bearerToken(ctx)
	if token == "" {
		return apperrors.Unauthorized("Unauthorized")
	}

	user, err := handler.service.CurrentUser(ctx.Context(), CurrentUserCommand{AccessToken: token})
	if err != nil {
		return err
	}

	return ctx.JSON(responses.Success(user, requestID(ctx)))
}

func bearerToken(ctx *fiber.Ctx) string {
	authHeader := strings.TrimSpace(ctx.Get(fiber.HeaderAuthorization))
	if authHeader == "" {
		return ""
	}

	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}

func (handler AuthHandler) setAuthCookies(ctx *fiber.Ctx, tokens SessionTokens) {
	if handler.cookieManager == nil {
		return
	}
	handler.cookieManager.SetAuthCookies(ctx, tokens.AccessToken, tokens.RefreshToken, tokens.AccessExpiresAt, tokens.RefreshExpiresAt)
}
