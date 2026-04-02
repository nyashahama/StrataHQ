package communications

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.List)
	r.Post("/{schemeId}", h.Create)
	return r
}
