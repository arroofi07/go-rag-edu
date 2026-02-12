package main

import (
	"log"
	"strconv"

	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/pkg/config"
	"rag-api/pkg/database"

	"github.com/gin-gonic/gin"
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

	// initialize router
	r := gin.Default()

	// Public Router
	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
	}

	// Protected Router
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"userID": c.GetString("userID"),
				"email":  c.GetString("email"),
				"role":   c.GetString("role"),
				"major":  c.GetString("major"),
			})
		})
	}

	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
