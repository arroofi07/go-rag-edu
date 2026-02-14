package dto

// tipe data untuk request register
type RegisterRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Major    string `json:"major"  example:"Computer Science"`
	Role     string `json:"role" example:"STUDENT" enums:"STUDENT,TEACHER,ADMIN"`
}

// tipe data untuk request login
type LoginRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// tipe data untuk response login
type AuthResponse struct {
	AccessToken string   `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User        UserInfo `json:"user"`
}

// tipe data untuk response user info
type UserInfo struct {
	ID    string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email string `json:"email" example:"user@example.com"`
	Name  string `json:"name" example:"John Doe"`
	Major string `json:"major" example:"Computer Science"`
	Role  string `json:"role" example:"STUDENT"`
}

// generic response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// error response
type ErrorResponse struct {
	Error string `json:"error" example:"Something went wrong"`
}

// register success response
type RegisterSuccessResponse struct {
	Message string   `json:"message" example:"User registered successfully"`
	User    UserInfo `json:"user"`
}

// login success response
type LoginSuccessResponse struct {
	Message string   `json:"message" example:"User logged in successfully"`
	Token   string   `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User    UserInfo `json:"user"`
}

// me response
type MeResponse struct {
	UserID string `json:"userID" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email  string `json:"email" example:"user@example.com"`
	Role   string `json:"role" example:"STUDENT"`
	Major  string `json:"major" example:"Computer Science"`
}
