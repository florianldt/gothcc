package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/gothcc/internal/config"
	"example.com/gothcc/internal/database"
	"example.com/gothcc/internal/handlers"
)

// migrations are embedded into the binary so the runtime image needs nothing but
// the binary itself. go:embed cannot reach parent/sibling directories, which is
// why this directive lives in the root package alongside main.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.Info("starting gothcc server")

	// signal-aware root context: cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(pool, migrationsFS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database ready")

	app := handlers.New(pool)
	srv := &http.Server{Addr: cfg.Addr, Handler: app.Routes()}

	go func() {
		slog.Info("server started", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, exiting")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	pool.Close()
	slog.Info("pool closed, goodbye")
}
