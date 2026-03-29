// backend/internal/invitation/routes.go
package invitation

import "github.com/go-chi/chi/v5"

// PublicRoutes registers endpoints that require no authentication.
func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{token}", h.Verify)
	r.Post("/{token}/accept", h.Accept)
	return r
}

// ProtectedRoutes registers endpoints that require a valid JWT.
func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/{id}/resend", h.Resend)
	r.Delete("/{id}", h.Revoke)
	return r
}
