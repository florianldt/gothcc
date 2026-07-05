package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration, loaded from the environment (and .env).
type Config struct {
	DatabaseURL string
	LogLevel    slog.Level
	Addr        string
}

// Load reads configuration from .env (if present) and the environment.
// DATABASE_URL is required; the process exits if it is missing.
func Load() Config {
	_ = godotenv.Load() // no-op if .env is absent

	var level slog.Level
	if err := level.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		level = slog.LevelInfo
	}

	return Config{
		DatabaseURL: mustEnv("DATABASE_URL"),
		LogLevel:    level,
		Addr:        getenv("ADDR", "0.0.0.0:8000"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable is not set", "key", key)
		os.Exit(1)
	}
	return v
}
