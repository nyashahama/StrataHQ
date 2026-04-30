package integrations

import "github.com/go-chi/chi/v5"

func (h *Handler) AdminRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListAPIClients)
	r.Post("/", h.CreateAPIClient)
	r.Delete("/{clientId}", h.RevokeAPIClient)
	return r
}

func (h *Handler) OpenRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.service.APIKeyAuth)
	r.Get("/schemes", h.OpenListSchemes)
	r.Get("/schemes/{schemeId}", h.OpenGetScheme)
	r.Get("/schemes/{schemeId}/units", h.OpenListUnits)
	r.Get("/schemes/{schemeId}/levy-periods", h.OpenListLevyPeriods)
	r.Get("/schemes/{schemeId}/levy-accounts", h.OpenListLevyAccounts)
	r.Get("/schemes/{schemeId}/levy-payments", h.OpenListLevyPayments)
	r.Get("/schemes/{schemeId}/financials", h.OpenFinancials)
	return r
}
