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
	_ "rag-api/docs"
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title           RAG API
// @version         1.0
// @description     API documentation for the RAG (Retrieval-Augmented Generation) service
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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

	// Swagger route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Public Routes
	api := app.Group("/api")
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	// Protected Routes
	protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
	protected.Get("/auth/me", authHandler.Me)

	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
