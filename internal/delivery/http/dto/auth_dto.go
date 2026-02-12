package dto

// tipe data untuk request register
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Major    string `json:"major" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=STUDENT TEACHER ADMIN"`
}

// tipe data untuk request login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// tipe data untuk response login
type AuthResponse struct {
	AccessToken string   `json:"access_token"`
	User        UserInfo `json:"user"`
}

// tipe data untuk response user info
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Major string `json:"major"`
	Role  string `json:"role"`
}
