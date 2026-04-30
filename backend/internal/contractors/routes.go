package contractors

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/marketplace", h.Marketplace)
	r.Patch("/{contractorId}", h.Update)
	r.Post("/{contractorId}/reviews", h.CreateReview)
	return r
}
