// backend/internal/earlyaccess/routes.go
package earlyaccess

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) PublicRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", h.Submit)
	return r
}

func (h *Handler) ProtectedRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/reject", h.Reject)
	return r
}
