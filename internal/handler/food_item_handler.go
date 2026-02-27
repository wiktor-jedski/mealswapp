// Phase: phase-01 | Task: 8 | Architecture: ARCH-005 | Design: FoodItemEntity
package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"mealswapp/internal/models"
	"mealswapp/internal/service"
)

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type ListResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type FoodItemHandler interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	Scale(c *fiber.Ctx) error
}

type foodItemHandler struct {
	service service.FoodItemService
}

func NewFoodItemHandler(service service.FoodItemService) FoodItemHandler {
	return &foodItemHandler{
		service: service,
	}
}

func (h *foodItemHandler) Create(c *fiber.Ctx) error {
	var input models.FoodItemCreate
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_request_body",
			Message: "Failed to parse request body",
		})
	}

	item, err := h.service.CreateFoodItem(c.Context(), input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(201).JSON(models.FoodItemResponse{
		ID:                item.ID,
		Name:              item.Name,
		PhysicalState:     item.PhysicalState,
		PrepTime:          item.PrepTime,
		AverageUnitWeight: item.AverageUnitWeight,
		Macros:            item.Macros,
		Micros:            item.Micros,
		CategoryTags:      item.CategoryTags,
		FunctionalityTags: item.FunctionalityTags,
		ImageURL:          item.ImageURL,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

func (h *foodItemHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	unitPref := models.UnitPreference(c.Query("units", "metric"))
	item, err := h.service.GetFoodItem(c.Context(), id, unitPref)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(item)
}

func (h *foodItemHandler) List(c *fiber.Ctx) error {
	query := models.FoodItemQuery{
		Page:      c.QueryInt("page", 1),
		PageSize:  c.QueryInt("page_size", 20),
		SortBy:    c.Query("sort_by", "name"),
		SortOrder: c.Query("sort_order", "asc"),
	}

	items, total, err := h.service.ListFoodItems(c.Context(), query)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(ListResponse{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: (int(total) + query.PageSize - 1) / query.PageSize,
	})
}

func (h *foodItemHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	var input models.FoodItemUpdate
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_request_body",
			Message: "Failed to parse request body",
		})
	}

	item, err := h.service.UpdateFoodItem(c.Context(), id, input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(item)
}

func (h *foodItemHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	if err := h.service.DeleteFoodItem(c.Context(), id); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(204).JSON(nil)
}

func (h *foodItemHandler) Scale(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	quantity := c.QueryFloat("quantity", 100)
	unitPref := models.UnitPreference(c.Query("units", "metric"))

	scaled, err := h.service.ScaleFoodItem(c.Context(), id, quantity, unitPref)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(scaled)
}

func (h *foodItemHandler) handleServiceError(c *fiber.Ctx, err error) error {
	errMsg := err.Error()

	switch errMsg {
	case service.ErrFoodItemNotFound.Error():
		return c.Status(404).JSON(ErrorResponse{
			Code:    "food_item_not_found",
			Message: "Food item not found",
		})
	case service.ErrInvalidCategoryTagID.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_category_tag_id",
			Message: "One or more category tag IDs are invalid",
		})
	case service.ErrInvalidFunctionalityTagID.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_functionality_tag_id",
			Message: "One or more functionality tag IDs are invalid",
		})
	case service.ErrFoodItemInUse.Error():
		return c.Status(409).JSON(ErrorResponse{
			Code:    "food_item_in_use",
			Message: "Food item is in use by recipes",
		})
	case service.ErrInvalidUnitPreference.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_unit_preference",
			Message: "Invalid unit preference",
		})
	case service.ErrInvalidPhysicalState.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_physical_state",
			Message: "Invalid physical state",
		})
	case service.ErrNegativeMacroValue.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "negative_macro_value",
			Message: "Negative macro values are not allowed",
		})
	case service.ErrQuantityOutOfRange.Error():
		return c.Status(400).JSON(ErrorResponse{
			Code:    "quantity_out_of_range",
			Message: "Quantity must be greater than 0",
		})
	default:
		if errMsg == "validation failed" {
			return c.Status(400).JSON(ErrorResponse{
				Code:    "validation_failed",
				Message: "Validation failed",
			})
		}
		return c.Status(500).JSON(ErrorResponse{
			Code:    "internal_error",
			Message: fmt.Sprintf("Internal server error: %v", err),
		})
	}
}
