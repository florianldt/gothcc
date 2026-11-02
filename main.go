package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Embeds the time zone database: the distroless image has none, so
	// time.LoadLocation would fail there.
	_ "time/tzdata"

	"example.com/gothcc/internal/components"
	"example.com/gothcc/internal/config"
	"example.com/gothcc/internal/database"
	"example.com/gothcc/internal/handlers"
	"example.com/gothcc/migrations"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.Info("starting gothcc server", "env", cfg.AppEnv)

	// Content hashes for /static URLs (cache busting after deploys).
	components.VersionAssets("static")

	// signal-aware root context: cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database ready")

	app := handlers.New(handlers.Deps{
		Pool:    pool,
		BaseURL: cfg.BaseURL,
		NoIndex: cfg.NoIndex,
	})
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: app.Routes(),
		// Bounds how long a client may take to send its headers, so slow
		// clients can't hold connections open (Slowloris).
		ReadHeaderTimeout: 10 * time.Second,
	}

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
