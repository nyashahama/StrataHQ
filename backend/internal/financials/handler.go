package financials

import (
	"encoding/json"
	"net/http"
	"strings"

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

//nolint:govet // Keep request DTO fields grouped by API meaning rather than field packing.
type upsertBudgetLineRequest struct {
	BudgetedCents int64  `json:"budgeted_cents"`
	ActualCents   int64  `json:"actual_cents"`
	Category      string `json:"category"`
	PeriodLabel   string `json:"period_label"`
}

type updateReserveFundRequest struct {
	BalanceCents int64 `json:"balance_cents"`
	TargetCents  int64 `json:"target_cents"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"), strings.TrimSpace(r.URL.Query().Get("period")))
	if err != nil {
		writeFinancialsError(w, err, "failed to load financial dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) UpsertBudgetLine(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req upsertBudgetLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	line, err := h.service.UpsertBudgetLine(r.Context(), identity, chi.URLParam(r, "schemeId"), UpsertBudgetLineInput{
		Category:      strings.TrimSpace(req.Category),
		PeriodLabel:   strings.TrimSpace(req.PeriodLabel),
		BudgetedCents: req.BudgetedCents,
		ActualCents:   req.ActualCents,
	})
	if err != nil {
		writeFinancialsError(w, err, "failed to save budget line")
		return
	}

	response.JSON(w, http.StatusOK, line)
}

func (h *Handler) UpdateReserveFund(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req updateReserveFundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	reserve, err := h.service.UpdateReserveFund(r.Context(), identity, chi.URLParam(r, "schemeId"), UpdateReserveFundInput{
		BalanceCents: req.BalanceCents,
		TargetCents:  req.TargetCents,
	})
	if err != nil {
		writeFinancialsError(w, err, "failed to update reserve fund")
		return
	}

	response.JSON(w, http.StatusOK, reserve)
}

func writeFinancialsError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "financial resource not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
