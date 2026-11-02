package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"example.com/gothcc/migrations"
)

// These tests need a Postgres server: TEST_DATABASE_URL points at a database
// on it (the dev container's db service, or CI's). Each test works in a
// throwaway database of its own.

func scratchDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("migrate_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	return pool
}

// A fresh database is built from the migrations; running again is a no-op.
func TestMigrateFresh(t *testing.T) {
	pool := scratchDB(t)
	ctx := context.Background()
	for range 2 {
		if err := Migrate(ctx, pool, migrations.FS); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO messages (body) VALUES ('hello')`); err != nil {
		t.Fatalf("messages table not usable: %v", err)
	}
}

// Instances booting together wait on the session lock instead of racing.
func TestMigrateConcurrent(t *testing.T) {
	pool := scratchDB(t)
	ctx := context.Background()
	errs := make(chan error, 3)
	for range 3 {
		go func() { errs <- Migrate(ctx, pool, migrations.FS) }()
	}
	for range 3 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}

// Every Down section undoes its Up: migrating all the way down and back up
// works, so a bad deploy can be rolled back.
func TestMigrateDownUp(t *testing.T) {
	pool := scratchDB(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
