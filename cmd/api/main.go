package main

import (
	"fmt"
	"log"

	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/pkg/config"
	"rag-api/pkg/database"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.Load()

	// connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

	// initialize repository
	userRepo := postgres.NewUserRepository(db)

	// initialize usecase
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)

	// initialize handler
	authHandler := handler.NewAuthHandler(authUsecase)

	// initialize fiber app
	app := fiber.New()

	// Public Routes
	api := app.Group("/api")
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	// Protected Routes
	protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
	protected.Get("/auth/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"userID": c.Locals("userID"),
			"email":  c.Locals("email"),
			"role":   c.Locals("role"),
			"major":  c.Locals("major"),
		})
	})

	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
