package handler

import (
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUsecase *auth.AuthUsecase
}

func NewAuthHandler(authUsecase *auth.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

// handler register start
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.authUsecase.Register(
		c.Context(),
		req.Email,
		req.Password,
		req.Name,
		req.Major,
		entity.UserRole(req.Role),
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User registered successfully", "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// handler register end

// handler login start
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	token, user, err := h.authUsecase.Login(
		c.Context(),
		req.Email,
		req.Password,
	)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User logged in successfully", "token": token, "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// handler login end
