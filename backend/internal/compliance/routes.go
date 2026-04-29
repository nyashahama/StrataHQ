package compliance

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/portfolio", h.PortfolioDashboard)
	r.Post("/{schemeId}/assess", h.Assess)
	r.Post("/{schemeId}/items", h.CreateItem)
	r.Get("/{schemeId}", h.Dashboard)
	r.Put("/{schemeId}/items/{itemId}", h.UpdateItem)
	r.Delete("/{schemeId}/items/{itemId}", h.DeleteItem)
	return r
}
