package financials

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.Dashboard)
	r.Put("/{schemeId}/budget-lines", h.UpsertBudgetLine)
	r.Put("/{schemeId}/reserve-fund", h.UpdateReserveFund)
	return r
}
