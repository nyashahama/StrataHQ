package scheme

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/units", h.ListUnits)
	r.Post("/{id}/units", h.CreateUnit)
	r.Put("/{id}/units/{unitId}", h.UpdateUnit)
	r.Get("/{id}/members", h.ListMembers)
	r.Patch("/{id}/members/{userId}", h.UpdateMember)
	return r
}
