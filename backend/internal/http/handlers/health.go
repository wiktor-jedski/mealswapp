package handlers

import (
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/observability"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	config           config.Config
	readinessChecker ReadinessChecker
	metrics          *observability.MetricsCollector
}

type HealthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
}

type ReadinessResponse struct {
	Status       string             `json:"status"`
	Environment  string             `json:"environment"`
	Dependencies []DependencyStatus `json:"dependencies,omitempty"`
}

type DependencyStatus struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

type ReadinessChecker interface {
	Check() []DependencyStatus
}

func NewHealthHandler(cfg config.Config, checker ReadinessChecker, metrics *observability.MetricsCollector) HealthHandler {
	return HealthHandler{config: cfg, readinessChecker: checker, metrics: metrics}
}

func (h HealthHandler) Health(ctx *fiber.Ctx) error {
	return ctx.JSON(responses.Success(HealthResponse{
		Status:      "ok",
		Environment: h.config.Environment,
	}, requestID(ctx)))
}

func (h HealthHandler) Ready(ctx *fiber.Ctx) error {
	dependencies := h.checkDependencies()
	healthy := dependenciesHealthy(dependencies)
	h.recordReadiness(healthy, dependencies)

	if !healthy {
		err := apperrors.DependencyUnavailable("Readiness check failed")
		envelope := responses.Failure(err.Code, err.Message, requestID(ctx))
		envelope.Error.Category = string(err.Category)
		envelope.Error.Retryable = err.Retryable
		envelope.Error.Fields = map[string]any{"dependencies": dependencies}
		return ctx.Status(err.Status).JSON(envelope)
	}

	return ctx.JSON(responses.Success(ReadinessResponse{
		Status:       "ready",
		Environment:  h.config.Environment,
		Dependencies: dependencies,
	}, requestID(ctx)))
}

func requestID(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}

	return ctx.GetRespHeader("X-Request-ID")
}

func (h HealthHandler) checkDependencies() []DependencyStatus {
	if h.readinessChecker == nil {
		return nil
	}
	return h.readinessChecker.Check()
}

func (h HealthHandler) recordReadiness(healthy bool, dependencies []DependencyStatus) {
	if h.metrics == nil {
		return
	}

	dependencyMetrics := make(map[string]bool, len(dependencies))
	for _, dependency := range dependencies {
		dependencyMetrics[dependency.Name] = dependency.Healthy
	}
	h.metrics.SetReadiness(healthy, dependencyMetrics)
}

func dependenciesHealthy(dependencies []DependencyStatus) bool {
	for _, dependency := range dependencies {
		if !dependency.Healthy {
			return false
		}
	}
	return true
}
