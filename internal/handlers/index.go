package handlers

import (
	"net/http"

	"example.com/gothcc/internal/components"
)

// ShowIndex renders the full landing page: layout + counter + message list.
func (a *App) ShowIndex(w http.ResponseWriter, r *http.Request) {
	msgs, err := a.queries.ListMessages(r.Context())
	if err != nil {
		a.fail(w, r, "list messages", err)
		return
	}
	p := a.page("/", "gothcc: HTMX · templ · sqlc · Tailwind · chi", "A minimal Go web starter.")
	render(w, r, components.Index(p, a.counter.Load(), msgs))
}
