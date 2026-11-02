package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"

	"example.com/gothcc/internal/components"
	"example.com/gothcc/internal/view"
)

// page builds the head data for a full page at path (e.g. "/messages").
func (a *App) page(path, title, description string) view.Page {
	p := view.Page{Title: title, Description: description}
	if a.baseURL != "" {
		p.URL = a.baseURL + path
	}
	return p
}

// render writes a templ component as an HTML response.
func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	renderStatus(w, r, http.StatusOK, c)
}

// renderStatus writes a templ component with an explicit status code.
func renderStatus(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("render component", "error", err)
	}
}

// NotFound renders the 404 page.
func (a *App) NotFound(w http.ResponseWriter, r *http.Request) {
	renderStatus(w, r, http.StatusNotFound, components.NotFound(a.page("", "Page not found", "")))
}

// fail renders 404 for a missing row and 500 for anything else. what names
// the failed step in the log ("list messages").
func (a *App) fail(w http.ResponseWriter, r *http.Request, what string, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		a.NotFound(w, r)
		return
	}
	slog.Error(what, "error", err)
	renderStatus(w, r, http.StatusInternalServerError, components.ServerError(a.page("", "Something went wrong", "")))
}

// redirect navigates to url: HX-Redirect for htmx requests (a 3xx would be
// followed inside the swap), a 303 otherwise.
func redirect(w http.ResponseWriter, r *http.Request, url string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// noIndex keeps a non-production deployment out of search results.
func noIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

// Health answers 200 when the database is reachable, 503 otherwise.
func (a *App) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	w.Header().Set("Cache-Control", "no-store")
	if err := a.pool.Ping(ctx); err != nil {
		slog.Error("health check", "error", err)
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

// Robots serves robots.txt: everything allowed in production, everything
// disallowed when the deployment is noindex (staging).
func (a *App) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if a.noIndex {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
		return
	}
	_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
}

// staticFiles serves /static. URLs carrying a content hash (?v=, see
// components.asset) never change, so browsers and CDNs may keep them for a
// year; anything else is revalidated.
func staticFiles(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fs.ServeHTTP(w, r)
	})
}
