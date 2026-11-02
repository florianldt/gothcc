# gothcc

A minimal Go web starter: **G**o · **H**TMX · **T**empl · sql**C** · **c**hi. Server-rendered HTML with type-safe, compile-time-checked templates, HTMX for interactivity, Tailwind for styling, and sqlc + pgx + Postgres for persistence. Single binary, migrations embedded. Clone, rename, build.

The Go sibling of [`hasta`](https://github.com/) (the Rust HTMX·Askama·SQLx·Tailwind·Axum starter).

## Prerequisites

- Docker. That's it for `make dev`: Go, templ, sqlc, Tailwind and air live in the dev container, at pinned versions.

## Quickstart

```bash
make dev        # Postgres + the app with live rebuilds on http://localhost:8000
```

Edit on the host; air rebuilds inside the container on every `.go`, `.templ` or `.sql` change (templ generation is part of the build), and Tailwind watches the CSS. Migrations run on boot.

With `make dev` running, in another terminal:

```bash
make generate   # templ + sqlc codegen (after editing .templ or .sql)
make test       # go test ./..., including the migration tests against the db service
make check      # what CI runs: generated code fresh, gofmt, vet, tests
make sh         # a shell in the dev container
make down       # stop; `make clean` also drops the database and caches
```

### On the host instead

Needs Go 1.26+, sqlc v1.31.1 and a Tailwind v4 standalone binary in the repo root (`./tailwindcss`):

```bash
cp .env.example .env
docker compose up -d db
./tailwindcss -i styles/app.css -o static/css/app.css --watch &
go tool templ generate && go run .
```

## Layout

```
main.go                     # bootstrap: config, pool, migrate-on-boot, router, graceful shutdown
sqlc.yaml                   # sqlc config (schema = migrations, out = internal/database/db)
migrations/                 # goose SQL migrations, embedded (migrations.FS); also sqlc's schema
queries/                    # sqlc query source (.sql)
internal/
├── config/                 # env/.env loading (APP_ENV, BASE_URL, NO_INDEX, …)
├── database/               # pgxpool setup + locked goose migrate; db/ = sqlc generated (committed)
├── handlers/               # App + Deps, chi routes, middleware, render/fail/redirect helpers
├── components/             # templ components (layout, pages, fragments, error pages); *_templ.go committed
└── view/                   # plain structs passed to components (Page: title + SEO)
styles/app.css              # Tailwind entry (scans internal/components/**/*.templ)
static/                     # served at /static; css/app.css is the Tailwind build output (gitignored)
dev/                        # dev container (Dockerfile + entrypoint); .air.toml configures rebuilds
```

Generated code (`internal/components/*_templ.go`, `internal/database/db/*.go`) is **committed** so `go build` and the Docker image need no codegen tools. Run `make generate` after editing a `.templ` or `.sql`; `make check` and CI gate freshness.

## What's built in

- **Response helpers** (`handlers/respond.go`): `render`/`renderStatus` set the content type and log render errors; `fail` turns `pgx.ErrNoRows` into the 404 page and anything else into the 500 page; `redirect` sends `HX-Redirect` to htmx requests and a 303 otherwise.
- **Middleware**: gzip, `CleanPath`, HEAD answered by GET routes, CSRF protection via Go's `http.CrossOriginProtection` (no tokens), and a client IP middleware that trusts `CF-Connecting-IP` only from Cloudflare's ranges (it assumes Cloudflare, then Traefik/Dokploy; adapt it for another proxy).
- **`/healthz`**: 200 when the database answers within 2 s, 503 otherwise. Point the container platform's health check at it.
- **Asset versioning**: every file under `static/` is hashed at startup; `asset("/static/…")` in templates adds `?v=<hash>`, and hashed URLs are served with a one-year immutable cache.
- **SEO head**: handlers build a `view.Page` with `a.page(path, title, description)`; the layout renders description, Open Graph, Twitter, canonical and JSON-LD tags.
- **Noindex outside production**: with `APP_ENV` not `production` (or `NO_INDEX=true`), every response carries `X-Robots-Tag: noindex` and `robots.txt` disallows everything, so staging never shows up in search.
- **Migrations**: goose with a Postgres session lock, so two instances booting together never migrate at once. Tests cover a fresh build, concurrent boots, and a full down/up round trip.

## Extending

- **Add a page**: component in `internal/components/foo.templ` wrapped in `@Layout(p) { ... }`, handler method in `internal/handlers/foo.go` that calls `render(w, r, components.Foo(a.page("/foo", "Title", "Description"), ...))`, route in `handlers.go` `Routes()`.
- **Add a query**: write SQL in `queries/` with a `-- name: X :one|:many|:exec` annotation, run `make sqlc`, call `a.queries.X(ctx, ...)`. Nullable columns come out as pointers.
- **Add a migration**: drop `migrations/0000N_what_changed.sql` with `-- +goose Up` / `-- +goose Down`. Runs on next boot; also picked up as sqlc schema.
- **Add a dependency** (mailer, storage, auth client): a field on `Deps` and `App`, built in `main.go`.
- **Add auth**: not included. Pick what fits (sessions + cookies, OAuth, passwordless codes) and hang it off `App` plus a chi middleware.

## Environment

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | Postgres connection string (required; set by docker-compose under `make dev`) |
| `APP_ENV` | `development` (default), `staging` or `production` |
| `BASE_URL` | Public origin for canonical links (default `http://localhost:8000`) |
| `NO_INDEX` | `true`/`false`; defaults to `true` unless `APP_ENV=production` |
| `LOG_LEVEL` | slog level: `debug`/`info`/`warn`/`error` (default `info`) |
| `ADDR` | Bind address (default `0.0.0.0:8000`) |

## CI and deploy

`.github/workflows/ci.yml` runs on every PR and master push: generated code is fresh, gofmt, vet, tests (with a Postgres service for the migration tests), and the production image builds.

`.github/workflows/deploy.yml` builds the multi-stage image (Tailwind, Go build, distroless non-root runtime) on push to `master`, pushes it to GHCR, and hits the Dokploy webhook in the `DOKPLOY_WEBHOOK_URL` secret (skipped with a notice while the secret is unset). Guarded by `if: false` until you adopt it. In Dokploy, set the health check to `GET /healthz` and `APP_ENV=production` plus `BASE_URL` on the production app.

## License

MIT (or whatever you change it to).
