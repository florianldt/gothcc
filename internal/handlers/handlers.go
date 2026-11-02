package handlers

import (
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/gothcc/internal/database/db"
)

// Deps are the dependencies injected into the app at bootstrap. A struct
// keeps New readable as the app grows (mailer, storage, auth, …), and lets
// tests build a router with only what they need: New(Deps{}).
type Deps struct {
	Pool *pgxpool.Pool
	// BaseURL is the public origin for canonical links (no trailing slash).
	BaseURL string
	// NoIndex adds X-Robots-Tag: noindex to every response and disallows
	// everything in robots.txt (staging).
	NoIndex bool
}

// App holds shared server state, injected into handlers via method receivers.
// It is passed by pointer because atomic.Int64 must not be copied.
type App struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	counter atomic.Int64
	baseURL string
	noIndex bool
}

// New builds the application state from its dependencies.
func New(d Deps) *App {
	return &App{
		pool:    d.Pool,
		queries: db.New(d.Pool),
		baseURL: d.BaseURL,
		noIndex: d.NoIndex,
	}
}

// Routes wires the router, middleware, and static file serving.
func (a *App) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(clientIP)
	if a.noIndex {
		r.Use(noIndex)
	}
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Gzip pages and assets: HTML compresses to a fifth of its size, often
	// small enough to arrive in one round trip.
	r.Use(middleware.Compress(5, "text/html", "text/css", "text/javascript", "application/javascript",
		"application/json", "application/xml", "image/svg+xml", "text/plain"))
	r.Use(middleware.CleanPath)
	// HEAD (uptime monitors, link checkers, some crawlers) is answered by
	// the GET route; net/http drops the body.
	r.Use(middleware.GetHead)
	// CSRF protection: rejects cross-origin POST/PUT/PATCH/DELETE from
	// browsers (Sec-Fetch-Site / Origin checks), no tokens needed.
	r.Use(http.NewCrossOriginProtection().Handler)

	// Health check for the container platform: the app is up and the
	// database answers.
	r.Get("/healthz", a.Health)
	r.Get("/robots.txt", a.Robots)

	r.Get("/", a.ShowIndex)
	r.Post("/counter/inc", a.CounterInc)
	r.Post("/counter/dec", a.CounterDec)
	r.Get("/messages", a.ListMessages)
	r.Post("/messages", a.CreateMessage)

	r.Handle("/static/*", http.StripPrefix("/static/", staticFiles(http.Dir("static"))))
	r.NotFound(a.NotFound)

	return r
}
