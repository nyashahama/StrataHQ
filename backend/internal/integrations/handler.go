package integrations

import (
	_ "embed"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

//go:embed openapi.json
var openAPIDocument []byte

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) OpenAPIDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPIDocument)
}

type createAPIClientRequest struct {
	Name      string   `json:"name"`
	SchemeIDs []string `json:"scheme_ids"`
	Scopes    []string `json:"scopes"`
}

func (h *Handler) CreateAPIClient(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	var req createAPIClientRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	input := CreateAPIClientInput{
		Name:      strings.TrimSpace(req.Name),
		SchemeIDs: req.SchemeIDs,
		Scopes:    req.Scopes,
	}
	result, err := h.service.CreateAPIClient(r.Context(), identity, input)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) ListAPIClients(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	result, err := h.service.ListAPIClients(r.Context(), identity)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) RevokeAPIClient(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	result, err := h.service.RevokeAPIClient(r.Context(), identity, chi.URLParam(r, "clientId"))
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func writeIntegrationsError(w http.ResponseWriter, err error) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}

func integrationIdentity(w http.ResponseWriter, r *http.Request) (Identity, bool) {
	identity, ok := IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing integration identity")
		return Identity{}, false
	}
	return identity, true
}

func requireScope(w http.ResponseWriter, r *http.Request, identity Identity, scope string) bool {
	if !identity.HasScope(scope) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "missing required integration scope")
		return false
	}
	return true
}

func requireScheme(w http.ResponseWriter, r *http.Request, identity Identity, schemeID string) bool {
	if !identity.CanAccessScheme(schemeID) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "scheme is not granted to this integration client")
		return false
	}
	return true
}

func parsePagination(r *http.Request) (limitRows int32, offsetRows int32) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage := parsePositiveInt(r.URL.Query().Get("per_page"), 50)
	if perPage > 200 {
		perPage = 200
	}
	return int32(perPage), int32((page - 1) * perPage)
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func (h *Handler) OpenListSchemes(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:schemes") {
		return
	}
	items, err := h.service.ListOpenAPISchemes(r.Context(), identity)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) OpenGetScheme(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:schemes") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	item, err := h.service.GetOpenAPIScheme(r.Context(), identity, schemeID)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) OpenListUnits(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:schemes") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	items, err := h.service.ListOpenAPIUnits(r.Context(), schemeID)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) OpenListLevyPeriods(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:schemes") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	items, err := h.service.ListOpenAPILevyPeriods(r.Context(), schemeID)
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) OpenListLevyAccounts(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:levies") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limitRows, offsetRows := parsePagination(r)
	items, total, err := h.service.ListOpenAPILevyAccounts(r.Context(), schemeID, OpenAPILevyAccountFilters{
		PeriodID:     r.URL.Query().Get("period_id"),
		Status:       r.URL.Query().Get("status"),
		UpdatedSince: r.URL.Query().Get("updated_since"),
		LimitRows:    limitRows,
		OffsetRows:   offsetRows,
	})
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSONList(w, http.StatusOK, items, response.Meta{
		Page:    page,
		PerPage: int(limitRows),
		Total:   total,
	})
}

func (h *Handler) OpenListLevyPayments(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:levies") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limitRows, offsetRows := parsePagination(r)
	items, total, err := h.service.ListOpenAPILevyPayments(r.Context(), schemeID, OpenAPILevyPaymentFilters{
		FromDate:   r.URL.Query().Get("from"),
		ToDate:     r.URL.Query().Get("to"),
		LimitRows:  limitRows,
		OffsetRows: offsetRows,
	})
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	response.JSONList(w, http.StatusOK, items, response.Meta{
		Page:    page,
		PerPage: int(limitRows),
		Total:   total,
	})
}

func (h *Handler) OpenFinancials(w http.ResponseWriter, r *http.Request) {
	identity, ok := integrationIdentity(w, r)
	if !ok || !requireScope(w, r, identity, "read:financials") {
		return
	}
	schemeID := chi.URLParam(r, "schemeId")
	if !requireScheme(w, r, identity, schemeID) {
		return
	}
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		writeIntegrationsError(w, ErrInvalidInput)
		return
	}
	periodLabel := r.URL.Query().Get("period")
	var periodFilter pgtype.Text
	if periodLabel != "" {
		periodFilter = pgtype.Text{String: periodLabel, Valid: true}
	}
	lines, err := h.service.db.Q.ListOpenAPIBudgetLinesByScheme(r.Context(), dbgen.ListOpenAPIBudgetLinesBySchemeParams{
		SchemeID:    sid,
		PeriodLabel: periodFilter,
	})
	if err != nil {
		writeIntegrationsError(w, err)
		return
	}
	reserve, _ := h.service.db.Q.GetOpenAPIReserveFundByScheme(r.Context(), sid)
	result := map[string]any{
		"budget_lines": lines,
	}
	if reserve.SchemeID != uuid.Nil {
		result["reserve_fund"] = reserve
	}
	response.JSON(w, http.StatusOK, result)
}
