package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL                 string
	RedisAddr                   string
	RedisPassword               string
	APIAddr                     string
	CORSOrigin                  string
	UploadDir                   string
	MaxUploadMB                 int
	OllamaBaseURL               string
	OllamaModel                 string
	ClassificationMinConfidence float64
	ClassifyConcurrency         int
	LogLevel                    string
	Environment                 string
	MigrationsPath              string
}

func Load() Config {
	maxMB, _ := strconv.Atoi(getenv("MAX_UPLOAD_MB", "20"))
	minConf, err := strconv.ParseFloat(getenv("CLASSIFICATION_MIN_CONFIDENCE", "0.7"), 64)
	if err != nil {
		minConf = 0.7
	}
	classifyConc, err := strconv.Atoi(getenv("CLASSIFY_CONCURRENCY", "2"))
	if err != nil || classifyConc < 1 {
		classifyConc = 2
	}
	if classifyConc > 16 {
		classifyConc = 16
	}
	return Config{
		DatabaseURL:                 getenv("DATABASE_URL", "postgres://rupee:rupee_secret@localhost:5432/rupee_os?sslmode=disable"),
		RedisAddr:                   getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:               os.Getenv("REDIS_PASSWORD"),
		APIAddr:                     getenv("API_ADDR", ":8080"),
		CORSOrigin:                  getenv("API_CORS_ORIGIN", "http://localhost:3000"),
		UploadDir:                   getenv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB:                 maxMB,
		OllamaBaseURL:               getenv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:                 getenv("OLLAMA_MODEL", "qwen2.5:3b"),
		ClassificationMinConfidence: minConf,
		ClassifyConcurrency:         classifyConc,
		LogLevel:                    getenv("LOG_LEVEL", "info"),
		Environment:                 getenv("ENVIRONMENT", "development"),
		MigrationsPath:              getenv("MIGRATIONS_PATH", "migrations"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
