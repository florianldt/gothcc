package handlers

import (
	"log/slog"
	"net/http"

	"example.com/gothcc/internal/components"
)

// ShowIndex renders the full landing page: layout + counter + message list.
func (a *App) ShowIndex(w http.ResponseWriter, r *http.Request) {
	msgs, err := a.queries.ListMessages(r.Context())
	if err != nil {
		slog.Error("list messages", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := components.Index(a.counter.Load(), msgs).Render(r.Context(), w); err != nil {
		slog.Error("render index", "error", err)
	}
}
