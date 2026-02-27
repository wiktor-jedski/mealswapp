// Phase: phase-01 | Task: 8 | Architecture: ARCH-005 | Design: FoodItemEntity

package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"mealswapp/internal/models"
	"mealswapp/internal/service"
)

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

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	items, total, err := h.service.ListFoodItems(c.Context(), query)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	totalPages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		totalPages++
	}

	return c.JSON(ListResponse{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
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

	return c.JSON(models.FoodItemResponse{
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

	return c.SendStatus(204)
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
	switch err {
	case service.ErrFoodItemNotFound:
		return c.Status(404).JSON(ErrorResponse{
			Code:    "food_item_not_found",
			Message: "Food item not found",
		})
	case service.ErrInvalidCategoryTagID:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_category_tag_id",
			Message: "One or more category tag IDs are invalid",
		})
	case service.ErrInvalidFunctionalityTagID:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_functionality_tag_id",
			Message: "One or more functionality tag IDs are invalid",
		})
	case service.ErrFoodItemInUse:
		return c.Status(409).JSON(ErrorResponse{
			Code:    "food_item_in_use",
			Message: "Food item is in use by recipes",
		})
	case service.ErrInvalidUnitPreference:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_unit_preference",
			Message: "Unit preference must be 'metric' or 'imperial'",
		})
	case service.ErrInvalidPhysicalState:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_physical_state",
			Message: "Physical state must be 'solid' or 'liquid'",
		})
	case service.ErrNegativeMacroValue:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "negative_macro_value",
			Message: "Macro values cannot be negative",
		})
	case service.ErrQuantityOutOfRange:
		return c.Status(400).JSON(ErrorResponse{
			Code:    "quantity_out_of_range",
			Message: "Quantity must be greater than 0",
		})
	default:
		return c.Status(500).JSON(ErrorResponse{
			Code:    "internal_error",
			Message: "An internal error occurred",
		})
	}
}
