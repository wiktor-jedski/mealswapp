package handlers

import (
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/responses"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	config config.Config
}

type HealthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
}

func NewHealthHandler(cfg config.Config) HealthHandler {
	return HealthHandler{config: cfg}
}

func (h HealthHandler) Health(ctx *fiber.Ctx) error {
	return ctx.JSON(responses.Success(HealthResponse{
		Status:      "ok",
		Environment: h.config.Environment,
	}, requestID(ctx)))
}

func (h HealthHandler) Ready(ctx *fiber.Ctx) error {
	return ctx.JSON(responses.Success(HealthResponse{
		Status:      "ready",
		Environment: h.config.Environment,
	}, requestID(ctx)))
}

func requestID(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}

	return ctx.GetRespHeader("X-Request-ID")
}
