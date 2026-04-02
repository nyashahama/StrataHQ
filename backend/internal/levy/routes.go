package levy

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.Dashboard)
	r.Post("/{schemeId}/periods", h.CreatePeriod)
	r.Post("/{schemeId}/reconcile", h.Reconcile)
	return r
}
