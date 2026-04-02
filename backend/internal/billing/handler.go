package billing

import (
	"io"
	"net/http"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	session, err := h.service.CreateCheckoutSession(r.Context(), identity)
	if err != nil {
		writeBillingError(w, err, "failed to create checkout session")
		return
	}

	response.JSON(w, http.StatusOK, session)
}

func (h *Handler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	session, err := h.service.CreatePortalSession(r.Context(), identity)
	if err != nil {
		writeBillingError(w, err, "failed to create customer portal session")
		return
	}

	response.JSON(w, http.StatusOK, session)
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	subscription, err := h.service.GetSubscription(r.Context(), identity)
	if err != nil {
		writeBillingError(w, err, "failed to load subscription")
		return
	}

	response.JSON(w, http.StatusOK, subscription)
}

func (h *Handler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid webhook body")
		return
	}

	if err := h.service.HandleWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature")); err != nil {
		writeBillingError(w, err, "failed to process webhook")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func writeBillingError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "billing resource not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	case ErrNotConfigured:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "billing is not configured")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
