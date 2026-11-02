package handlers

import (
	"net/http"
	"strings"

	"example.com/gothcc/internal/components"
)

// ListMessages returns the message-list fragment (HTMX target #message-list).
func (a *App) ListMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := a.queries.ListMessages(r.Context())
	if err != nil {
		a.fail(w, r, "list messages", err)
		return
	}
	render(w, r, components.MessageList(msgs))
}

// CreateMessage persists a message via sqlc, then returns the refreshed
// message-list fragment for HTMX to swap in.
func (a *App) CreateMessage(w http.ResponseWriter, r *http.Request) {
	body := strings.TrimSpace(r.FormValue("body"))
	if body != "" {
		if _, err := a.queries.CreateMessage(r.Context(), body); err != nil {
			a.fail(w, r, "create message", err)
			return
		}
	}

	msgs, err := a.queries.ListMessages(r.Context())
	if err != nil {
		a.fail(w, r, "list messages", err)
		return
	}
	render(w, r, components.MessageList(msgs))
}
