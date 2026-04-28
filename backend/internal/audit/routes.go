package audit

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/schemes/{schemeId}/events", h.ListSchemeEvents)
	return r
}
