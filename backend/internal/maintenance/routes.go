package maintenance

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.Dashboard)
	r.Post("/{schemeId}", h.Create)
	r.Post("/{schemeId}/{id}/assign", h.Assign)
	r.Post("/{schemeId}/{id}/resolve", h.Resolve)
	return r
}
