// Phase: phase-01 | Task: 9 | Architecture: ARCH-005 | Design: FoodItemEntity
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"mealswapp/internal/database"
	"mealswapp/internal/repository"
	"mealswapp/internal/router"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "mealswapp",
		ServerHeader: "mealswapp/1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	ctx := context.Background()
	cfg := database.DefaultConfig()

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool.Config())
	defer db.Close()

	foodItemRepo := repository.NewFoodItemRepository(pool)
	tagRepo := repository.NewTagRepository(db)

	router.Setup(app, foodItemRepo, tagRepo)

	go func() {
		addr := fmt.Sprintf(":%s", getEnv("PORT", "3000"))
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
