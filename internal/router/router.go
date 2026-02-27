// Phase: phase-01 | Task: 9 | Architecture: ARCH-005 | Design: FoodItemEntity
package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"mealswapp/internal/handler"
	"mealswapp/internal/repository"
	"mealswapp/internal/service"
)

func Setup(app *fiber.App, foodItemRepo repository.FoodItemRepository, tagRepo repository.TagRepository) {
	app.Use(recover.New())
	app.Use(logger.New())

	foodItemService := service.NewFoodItemService(foodItemRepo, tagRepo)
	foodItemHandler := handler.NewFoodItemHandler(foodItemService)

	api := app.Group("/api/v1")

	foodItems := api.Group("/food-items")
	foodItems.Post("", foodItemHandler.Create)
	foodItems.Get("", foodItemHandler.List)
	foodItems.Get("/:id", foodItemHandler.GetByID)
	foodItems.Put("/:id", foodItemHandler.Update)
	foodItems.Delete("/:id", foodItemHandler.Delete)
	foodItems.Get("/:id/scale", foodItemHandler.Scale)
}
