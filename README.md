# gothcc

A minimal Go web starter — **G**o · **H**TMX · **T**empl · sql**C** · chi (**A**). Server-rendered HTML with type-safe, compile-time-checked templates, HTMX for interactivity, Tailwind for styling, and sqlc + pgx + Postgres for persistence. Single binary, migrations embedded. Clone, rename, build.

The Go sibling of [`hasta`](https://github.com/) (the Rust HTMX·Askama·SQLx·Tailwind·Axum starter).

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- Docker (for Postgres via `docker compose`)
- `make` and `curl` (for the Tailwind CLI download)

`templ` is pinned as a Go tool directive (no install needed). `sqlc` is installed by `make setup`.

## Quickstart

```bash
cp .env.example .env
docker compose up -d        # start Postgres
make setup                  # one-time: download Tailwind CLI + install sqlc
make generate               # templ + sqlc codegen
make css                    # build CSS once
make run                    # boots on http://localhost:8000 (runs migrations)
```

For an active dev loop (Tailwind watching + templ regenerating + server restarting + live reload):

```bash
make dev                    # then open http://localhost:7331
```

## Layout

```
main.go                     # bootstrap: config, pool, migrate-on-boot, router, graceful shutdown
                            # (//go:embed migrations lives here — embed can't reach parent dirs)
sqlc.yaml                   # sqlc config (schema = migrations, out = internal/database/db)
migrations/                 # goose SQL migrations (run on boot; also sqlc's schema source)
queries/                    # sqlc query source (.sql)
internal/
├── config/                 # env/.env loading
├── database/               # pgxpool setup + goose migrate; db/ = sqlc generated (committed)
├── handlers/               # HTTP logic; App state struct + chi routes
└── components/                  # templ components (layout, pages, fragments); *_templ.go committed
styles/app.css              # Tailwind entry (scans internal/components/**/*.templ)
static/css/app.css          # Tailwind build output (gitignored)
```

Generated code (`internal/components/*_templ.go`, `internal/database/db/*.go`) is **committed** so `go build` and the Docker image need no codegen tools. Run `make generate` after editing a `.templ` or `.sql`; `make check` gates freshness.

## Extending

- **Add a page**: component in `internal/components/foo.templ`, handler method in `internal/handlers/foo.go`, wire the route in `handlers.go` `Routes()`. Run `make templ`.
- **Add a query**: write SQL in `queries/query.sql` with a `-- name: X :one|:many|:exec` annotation, run `make sqlc`, call `a.queries.X(ctx, ...)`.
- **Add a migration**: drop `migrations/0000N_xxx.sql` with `-- +goose Up` / `-- +goose Down`. Runs on next boot; also picked up as sqlc schema.
- **Add auth**: not included — pick what fits (sessions + cookies, OAuth, passwordless codes) and store it on the `App` struct.
- **JSON access logs**: `handlers.go` uses chi's text `middleware.Logger`; swap it for a small slog request-logger if you want structured logs.

## Deploy

`.github/workflows/deploy.yml` builds a multi-stage Docker image on push to `main`, pushes to GHCR, and hits a Dokploy webhook (`DOKPLOY_WEBHOOK_URL` secret). Guarded by `if: false` until you adopt it.

## License

MIT (or whatever you change it to).
