package billing

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/checkout", h.CreateCheckoutSession)
	r.Get("/subscription", h.GetSubscription)
	return r
}

func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/stripe", h.HandleStripeWebhook)
	return r
}
