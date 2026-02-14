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

	// open ai
	OpenAIKey            string
	OpenAIEmbeddingModel string
	OpenAIChatModel      string

	// rag config
	ChunkSize           int
	ChunkOverlap        int
	TopKResults         int
	SimilarityThreshold float64
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

		// OpenAI
		OpenAIKey:            getEnv("OPENAI_API_KEY", ""),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAIChatModel:      getEnv("OPENAI_CHAT_MODEL", "gpt-4o-mini"),

		// RAG Config
		ChunkSize:           getEnvInt("CHUNK_SIZE", 1000),
		ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 200),
		TopKResults:         getEnvInt("TOP_K_RESULTS", 6),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.5),
	}

}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}
