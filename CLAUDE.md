# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`gothcc` is a starter template for Go web apps. Stack: **G**o · **H**TMX · **T**empl · sql**C** · **c**hi. Single-binary server, server-rendered HTMX-driven UI, type-safe compile-time-checked templates (templ), type-safe SQL (sqlc + pgx), Postgres, standalone Tailwind CLI (no Node). It is the Go port of the Rust `hasta` template.

It is intentionally minimal: no auth. The only demo features are an in-memory HTMX counter and a tiny Postgres-backed messages list. Clone it, rename the module, build on it.

## Commands

Development runs in Docker (`dev/Dockerfile`: Go, templ, sqlc, Tailwind and air at pinned versions). Every helper but `dev`/`down`/`clean` runs inside the running app container.

```bash
make dev          # docker compose up --build: Postgres + app with air rebuilds, http://localhost:8000
make down         # stop the stack
make clean        # also drop the database volume, module caches and the dev image
make sh           # shell in the app container
make generate     # templ generate + sqlc generate (run after editing .templ / .sql)
make templ        # templ generate only
make sqlc         # sqlc generate only
make test         # go test ./... (migration tests use TEST_DATABASE_URL, set by compose)
make check        # generate + fail on git diff + gofmt + go vet + go test (same as CI)
```

On the host without Docker for the app: `docker compose up -d db`, `./tailwindcss -i styles/app.css -o static/css/app.css --watch &`, `go tool templ generate && go run .` (needs Go 1.26, sqlc v1.31.1, a Tailwind v4 binary).

## Environment variables

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | Postgres connection string (required). Compose sets it to the `db` service. |
| `APP_ENV` | `development` (default), `staging` or `production` |
| `BASE_URL` | Public origin for canonical/og:url links (default `http://localhost:8000`) |
| `NO_INDEX` | `true`/`false`; defaults to `true` unless `APP_ENV=production` |
| `LOG_LEVEL` | slog level: `debug`/`info`/`warn`/`error` (default `info`) |
| `ADDR` | Bind address (default `0.0.0.0:8000`) |
| `TEST_DATABASE_URL` | Tests only: a Postgres server the migration tests create throwaway databases on |

## Architecture

Single-binary Go web server.

- **[chi](https://github.com/go-chi/chi)** v5 router. Middleware order in `Routes()`: `clientIP`, `noIndex` (when set), Logger, Recoverer, Compress (gzip), CleanPath, GetHead, `http.CrossOriginProtection` (CSRF).
- **[templ](https://templ.guide/)**: typed `.templ` components compiled to Go. Pinned as a Go `tool` directive (`go tool templ`).
- **[sqlc](https://sqlc.dev/)** + **[pgx](https://github.com/jackc/pgx)** v5: SQL in `queries/`, generated Go in `internal/database/db/`. Schema source is `migrations/`. `emit_pointers_for_null_types` is on, so nullable columns are pointers.
- **[goose](https://github.com/pressly/goose)**: migrations embedded by `migrations/migrations.go` (`migrations.FS`) and applied on boot with goose's `Provider` plus a Postgres session lock (safe with several instances).
- **log/slog**: JSON structured logging. **godotenv**: `.env` loading (real env vars win).
- **Tailwind CSS** v4 standalone CLI (musl build in Docker): `styles/app.css` scans `internal/components/**/*.templ`. The Inter font is a `<link>` in the layout, not a CSS `@import`.
- **HTMX** v2.0.4 via CDN in `internal/components/layout.templ`.
- **air** (dev only): `.air.toml` runs `templ generate` as part of each build, so one edit means one rebuild and no templ watch-mode artifacts.

### Source layout

- `main.go`: bootstrap: config, slog, asset hashing, pool, migrate, router, `http.Server` with `ReadHeaderTimeout`, graceful SIGINT/SIGTERM shutdown. Imports `time/tzdata` (distroless has no zoneinfo).
- `migrations/`: goose SQL files + `migrations.go` (embed).
- `internal/config/`: `Config` loaded from env/.env.
- `internal/database/`: `NewPool` (tuned pgxpool), `Migrate` (locked goose), `migrate_test.go`. `db/` is sqlc-generated (committed).
- `internal/handlers/`: `Deps` (constructor input) and `App` (pool, queries, counter, baseURL, noIndex); `Routes()`; `respond.go` helpers (`page`, `render`, `renderStatus`, `fail`, `redirect`, `NotFound`, `Health`, `Robots`, `staticFiles`); `clientip.go`.
- `internal/components/`: templ components: `layout.templ` (`Layout(view.Page)` with the SEO head), `errors.templ` (NotFound/ServerError), `partials.templ` (Header/Footer), pages and fragments; `assets.go` (`VersionAssets`, `asset`). `*_templ.go` are generated and committed.
- `internal/view/`: plain structs for components; `Page` holds title, description, canonical URL, OG image, JSON-LD.

### Generated code is committed

`internal/components/*_templ.go` and `internal/database/db/*.go` are committed so `go build`, IDEs, and the Docker build work with zero extra tools. Regenerate with `make generate`; `make check` and CI gate staleness. sqlc must stay at the version pinned in `dev/Dockerfile` and `ci.yml` (v1.31.1), or the generated header changes. The production Dockerfile does **not** run codegen.

### Responding

- Full pages: `render(w, r, components.Foo(a.page("/foo", "Title", "Description"), ...))`; the component wraps `@Layout(p) { ... }`.
- Errors: `a.fail(w, r, "what failed", err)` renders 404 for `pgx.ErrNoRows`, else logs and renders 500.
- Redirects after a form post: `redirect(w, r, url)` (HX-Redirect for htmx, 303 otherwise).
- HTMX endpoints return a **bare fragment** component (no layout), e.g. `CounterValue`, `MessageList`, swapped via `hx-target` + `hx-swap="outerHTML"`. htmx does not swap 4xx/5xx responses by default.
- Static files in templates: `asset("/static/...")` adds the content hash for cache busting.

### Adding a new page

1. Component in `internal/components/foo.templ` (wrapped in `@Layout(p) { ... }`; bare components for HTMX fragments).
2. Handler method in `internal/handlers/foo.go` on `*App`, using `render` and `a.page`.
3. Wire the route in `internal/handlers/handlers.go` `Routes()`.
4. air regenerates under `make dev`; otherwise `make templ`.

### Adding a query

Write SQL in `queries/` with a `-- name: X :one|:many|:exec` annotation, run `make sqlc`, then call `a.queries.X(ctx, ...)`.

### Adding a migration

Drop `migrations/0000N_what_changed.sql` with `-- +goose Up` / `-- +goose Down` blocks. Runs on next boot (air restarts on `.sql` edits) and is sqlc's schema. `TestMigrateDownUp` checks every Down undoes its Up.

### Adding a dependency

Add a field to `handlers.Deps` and `App`, copy it in `New`, build it in `main.go`. Tests build routers with `New(Deps{})`, so keep routes that don't need a dependency working without it.

### Tests

`go test ./...`. Router tests (`routes_test.go`) use `httptest` and `New(Deps{})` with no database, so they only hit routes that run no query. Database tests skip unless `TEST_DATABASE_URL` is set (compose and CI set it).

### Tailwind

`styles/app.css` uses `source(none)` + `@source "../internal/components/**/*.templ"`. Keep class names as complete literal strings so the scanner sees them; for dynamically-built classes, write full literals or add `@source inline("...")`. Keep the Tailwind version in `Dockerfile` and `dev/Dockerfile` in sync, and use the `-musl` build on Alpine.

### Adding auth

Not included. Pick what fits (sessions + cookies, OAuth, passwordless email codes) and hang it off the `App` struct + a chi middleware. Set cookies `Secure` outside `APP_ENV=development`.

## CI and deploy

- `.github/workflows/ci.yml`: on PRs and master pushes: generated code fresh, gofmt, vet, tests with a Postgres 18 service, production image build.
- `.github/workflows/deploy.yml`: on master push (or manual run): build the multi-stage image (Tailwind musl, Go build, distroless non-root), push to GHCR, call the Dokploy webhook from the `DOKPLOY_WEBHOOK_URL` secret (skipped while unset). Guarded by `if: false` until adopted. Dokploy health check: `GET /healthz`.
- The `clientIP` middleware assumes Cloudflare in front of Traefik (Dokploy). Adapt it for another proxy.
