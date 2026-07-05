# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`gothcc` is a starter template for Go web apps. Stack: **G**o · **H**TMX · **T**empl · sql**C** · **c**hi. Single-binary server, server-rendered HTMX-driven UI, type-safe compile-time-checked templates (templ), type-safe SQL (sqlc + pgx), Postgres, standalone Tailwind CLI (no Node). It is the Go port of the Rust `hasta` template.

It is intentionally minimal — no auth. The only demo features are an in-memory HTMX counter and a tiny Postgres-backed messages list. Clone it, rename the module, build on it.

## Commands

```bash
make setup        # one-time: download Tailwind CLI + install sqlc
make generate     # templ generate + sqlc generate (run after editing .templ / .sql)
make templ        # templ generate only
make sqlc         # sqlc generate only
make css          # build Tailwind CSS once (minified)
make css-watch    # Tailwind watch mode
make dev          # full dev loop: css-watch + templ watch/proxy/rebuild (browse :7331)
make run          # go run .
make build        # generate + css + static binary at bin/server
make check        # fail if generated code is stale (generate + git diff) + go vet
go test ./...     # tests
```

## Running locally

```bash
cp .env.example .env
docker compose up -d   # start Postgres
make setup             # one-time
make generate          # codegen (also done automatically under `make dev`)
make css               # build CSS
make run               # server on http://localhost:8000 (migrations run on boot)
```

Binds `ADDR` (default `0.0.0.0:8000`). Log level via `LOG_LEVEL` (default `info`). Migrations in `migrations/` run on boot.

## Environment variables

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | Postgres connection string, e.g. `postgres://gothcc:gothcc@localhost/gothcc` (required) |
| `LOG_LEVEL` | Optional slog level: `debug`/`info`/`warn`/`error` (default `info`) |
| `ADDR` | Optional bind address (default `0.0.0.0:8000`) |

## Architecture

Single-binary Go web server.

- **[chi](https://github.com/go-chi/chi)** v5 router + middleware (RealIP, Logger, Recoverer) + [go-chi/cors](https://github.com/go-chi/cors).
- **[templ](https://templ.guide/)** — typed `.templ` components compiled to Go; compile-time checked. Pinned as a Go `tool` directive (`go tool templ`).
- **[sqlc](https://sqlc.dev/)** + **[pgx](https://github.com/jackc/pgx)** v5 — write SQL in `queries/`, generate type-safe Go into `internal/database/db/`. `sqlc.yaml` uses `migrations/` as the schema source.
- **[goose](https://github.com/pressly/goose)** — migrations embedded via `//go:embed` (in `main.go`) and run on boot over the pgx pool (stdlib bridge).
- **log/slog** — JSON structured logging.
- **[godotenv](https://github.com/joho/godotenv)** — `.env` loading.
- **Tailwind CSS** (standalone CLI, no Node) — `styles/app.css` scans `internal/components/**/*.templ`.
- **HTMX** v2.0.4 via CDN in `internal/components/layout.templ`.

### Source layout

- `main.go` — bootstrap: config, slog, pool, migrate, router, graceful SIGINT/SIGTERM shutdown. Holds the `//go:embed migrations/*.sql` directive (embed can't reach subdirs of other packages).
- `internal/config/` — `Config` loaded from env/.env.
- `internal/database/` — `NewPool` (tuned pgxpool) + `Migrate` (goose). `db/` subpackage is sqlc-generated (committed).
- `internal/handlers/` — HTTP logic. `App` struct (pool + `*db.Queries` + `atomic.Int64`) with handler methods; `Routes()` wires chi.
- `internal/components/` — templ components: `layout.templ` (base wrapper with `{ children... }`), `partials.templ` (Header/Footer), page + fragment components. `*_templ.go` are generated and committed.

### Generated code is committed

`internal/components/*_templ.go` and `internal/database/db/*.go` are committed so `go build`, IDEs, and the Docker build work with zero extra tools. Regenerate with `make generate` after editing a `.templ` or `.sql`; `make check` gates staleness. The Dockerfile does **not** run codegen — it trusts the committed output.

### Adding a new page

1. Component in `internal/components/foo.templ` (typically wrapped in `@Layout(...) { ... }`; bare components for HTMX fragments).
2. Handler method in `internal/handlers/foo.go` on `*App`, rendering via `components.Foo(...).Render(r.Context(), w)`.
3. Wire the route in `internal/handlers/handlers.go` `Routes()`.
4. `make templ` (or it regenerates under `make dev`).

### HTMX pattern

Full pages return a component wrapped in `@Layout`. HTMX endpoints return a **bare fragment** component (no layout) — e.g. `CounterValue`, `MessageList` — swapped via `hx-target` + `hx-swap="outerHTML"`. Under `make dev`, templ's `--proxy` injects its live-reload script only before `</body>`, so fragment responses are untouched.

### Adding a query

Write SQL in `queries/query.sql` with a `-- name: X :one|:many|:exec` annotation, run `make sqlc`, then call `a.queries.X(ctx, ...)`. The schema comes from `migrations/`.

### Adding a migration

Drop `migrations/0000N_name.sql` with `-- +goose Up` / `-- +goose Down` blocks. Runs on next boot and is used as sqlc's schema.

### Tailwind

`styles/app.css` uses `source(none)` + `@source "../internal/components/**/*.templ"`. Keep class names as complete literal strings so the scanner sees them; for dynamically-built classes, write full literals or add `@source inline("...")`.

### Adding auth

Not included. Pick what fits (sessions + cookies, OAuth, passwordless email codes) and hang it off the `App` struct + a chi middleware.

## Deploy

`.github/workflows/deploy.yml` builds a multi-stage Docker image on push to `main` (Tailwind CSS → Go build → distroless non-root runtime), pushes to GHCR, and pings a Dokploy webhook (`DOKPLOY_WEBHOOK_URL` secret). Guarded by `if: false` until adopted.
