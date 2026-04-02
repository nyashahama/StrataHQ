package levy

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createPeriodRequest struct {
	Label       string `json:"label"`
	DueDate     string `json:"due_date"`
	AmountCents int64  `json:"amount_cents"`
}

type reconcilePaymentRequest struct {
	AccountID   string  `json:"account_id"`
	PaymentDate string  `json:"payment_date"`
	Reference   string  `json:"reference"`
	BankRef     *string `json:"bank_ref"`
	AmountCents int64   `json:"amount_cents"`
}

type reconcileRequest struct {
	Payments []reconcilePaymentRequest `json:"payments"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeLevyError(w, err, "failed to load levy dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) CreatePeriod(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	dueDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.DueDate))
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "due_date must be YYYY-MM-DD")
		return
	}

	created, err := h.service.CreatePeriod(r.Context(), identity, chi.URLParam(r, "schemeId"), CreatePeriodInput{
		Label:       strings.TrimSpace(req.Label),
		DueDate:     dueDate,
		AmountCents: req.AmountCents,
	})
	if err != nil {
		writeLevyError(w, err, "failed to create levy period")
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Reconcile(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req reconcileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	payments := make([]ReconcilePaymentInput, 0, len(req.Payments))
	for _, payment := range req.Payments {
		paymentDate, err := time.Parse("2006-01-02", strings.TrimSpace(payment.PaymentDate))
		if err != nil {
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "payment_date must be YYYY-MM-DD")
			return
		}
		payments = append(payments, ReconcilePaymentInput{
			AccountID:   strings.TrimSpace(payment.AccountID),
			PaymentDate: paymentDate,
			Reference:   strings.TrimSpace(payment.Reference),
			BankRef:     normalizeOptionalString(payment.BankRef),
			AmountCents: payment.AmountCents,
		})
	}

	result, err := h.service.Reconcile(r.Context(), identity, chi.URLParam(r, "schemeId"), payments)
	if err != nil {
		writeLevyError(w, err, "failed to reconcile levy payments")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func writeLevyError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "levy resource not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
