package compliance

import (
	"net/http"
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

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeComplianceError(w, err, "failed to load compliance dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func writeComplianceError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "scheme not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}

type createItemRequest struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Requirement string `json:"requirement"`
	Detail      string `json:"detail"`
	Action      string `json:"action"`
	DueDate     string `json:"due_date,omitempty"`
}

type updateItemRequest struct {
	Status  *string `json:"status,omitempty"`
	Detail  *string `json:"detail,omitempty"`
	Action  *string `json:"action,omitempty"`
	DueDate *string `json:"due_date,omitempty"`
}

func (h *Handler) PortfolioDashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.PortfolioDashboard(r.Context(), identity)
	if err != nil {
		writeComplianceError(w, err, "failed to load portfolio compliance")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createItemRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		t, err := time.Parse("2006-01-02", req.DueDate)
		if err == nil {
			dueDate = &t
		}
	}

	item, err := h.service.CreateItem(r.Context(), identity, chi.URLParam(r, "schemeId"), CreateItemInput{
		Category:    req.Category,
		Title:       req.Title,
		Requirement: req.Requirement,
		Detail:      req.Detail,
		Action:      req.Action,
		DueDate:     dueDate,
	})
	if err != nil {
		writeComplianceError(w, err, "failed to create compliance item")
		return
	}

	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req updateItemRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	input := UpdateItemInput{
		Status:  req.Status,
		Detail:  req.Detail,
		Action:  req.Action,
	}
	if req.DueDate != nil {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err == nil {
			input.DueDate = &t
		}
	}

	item, err := h.service.UpdateItem(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "itemId"), input)
	if err != nil {
		writeComplianceError(w, err, "failed to update compliance item")
		return
	}

	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	if err := h.service.DeleteItem(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "itemId")); err != nil {
		writeComplianceError(w, err, "failed to delete compliance item")
		return
	}

	response.NoContent(w)
}

func (h *Handler) Assess(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Assess(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeComplianceError(w, err, "failed to assess compliance")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}
