package handlers

import (
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/gothcc/internal/database/db"
)

// App holds shared server state, injected into handlers via method receivers —
// the idiomatic Go analogue of axum's State extractor. It is passed by pointer
// because atomic.Int64 must not be copied.
type App struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	counter atomic.Int64
}

// New builds the application state from a database pool.
func New(pool *pgxpool.Pool) *App {
	return &App{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Routes wires the router, middleware, and static file serving.
func (a *App) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/", a.ShowIndex)
	r.Post("/counter/inc", a.CounterInc)
	r.Post("/counter/dec", a.CounterDec)
	r.Get("/messages", a.ListMessages)
	r.Post("/messages", a.CreateMessage)

	fs := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	return r
}
