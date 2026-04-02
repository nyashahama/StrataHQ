package maintenance

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

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

//nolint:govet // Keep request DTO fields grouped by API meaning rather than field packing.
type assignRequest struct {
	ContractorPhone *string `json:"contractor_phone"`
	ContractorName  string  `json:"contractor_name"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeMaintenanceError(w, err, "failed to load maintenance dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	created, err := h.service.Create(r.Context(), identity, chi.URLParam(r, "schemeId"), CreateInput{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Category:    strings.TrimSpace(req.Category),
	})
	if err != nil {
		writeMaintenanceError(w, err, "failed to create maintenance request")
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	updated, err := h.service.Assign(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "id"), AssignInput{
		ContractorName:  strings.TrimSpace(req.ContractorName),
		ContractorPhone: normalizeOptionalString(req.ContractorPhone),
	})
	if err != nil {
		writeMaintenanceError(w, err, "failed to assign maintenance contractor")
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	updated, err := h.service.Resolve(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "id"))
	if err != nil {
		writeMaintenanceError(w, err, "failed to resolve maintenance request")
		return
	}

	response.JSON(w, http.StatusOK, updated)
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

func writeMaintenanceError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "maintenance request not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
