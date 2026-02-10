package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration
	Port          int
}

func Load() *Config {
	godotenv.Load()

	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "168h"))

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: jwtExp,
		Port:          port,
	}

}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
