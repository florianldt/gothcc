package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration, loaded from the environment (and .env).
type Config struct {
	DatabaseURL string
	LogLevel    slog.Level
	Addr        string
	// AppEnv is "development", "staging" or "production".
	AppEnv string
	// BaseURL is the public origin used for canonical and og:url links.
	BaseURL string
	// NoIndex asks search engines not to index the site (X-Robots-Tag and
	// robots.txt). On by default outside production, so a staging deploy
	// never competes with the real site in search results.
	NoIndex bool
}

// Load reads configuration from .env (if present) and the environment.
// DATABASE_URL is required; the process exits if it is missing.
func Load() Config {
	_ = godotenv.Load() // no-op if .env is absent

	var level slog.Level
	if err := level.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		level = slog.LevelInfo
	}

	appEnv := getenv("APP_ENV", "development")

	return Config{
		DatabaseURL: mustEnv("DATABASE_URL"),
		LogLevel:    level,
		Addr:        getenv("ADDR", "0.0.0.0:8000"),
		AppEnv:      appEnv,
		BaseURL:     strings.TrimRight(getenv("BASE_URL", "http://localhost:8000"), "/"),
		NoIndex:     getenv("NO_INDEX", fmt.Sprint(appEnv != "production")) == "true",
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
