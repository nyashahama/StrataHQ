package whatsapp

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.Dashboard)
	r.Post("/{schemeId}/broadcasts", h.CreateBroadcast)
	r.Post("/{schemeId}/messages/{messageId}/maintenance-request", h.CreateMaintenanceFromMessage)
	r.Patch("/{schemeId}/maintenance-intakes/{intakeId}", h.DismissMaintenanceIntake)
	return r
}
