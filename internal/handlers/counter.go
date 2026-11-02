package handlers

import (
	"net/http"

	"example.com/gothcc/internal/components"
)

// CounterInc increments the in-memory counter and returns the bare fragment
// that HTMX swaps into #counter-value.
func (a *App) CounterInc(w http.ResponseWriter, r *http.Request) {
	render(w, r, components.CounterValue(a.counter.Add(1)))
}

// CounterDec decrements the in-memory counter and returns the bare fragment.
func (a *App) CounterDec(w http.ResponseWriter, r *http.Request) {
	render(w, r, components.CounterValue(a.counter.Add(-1)))
}
