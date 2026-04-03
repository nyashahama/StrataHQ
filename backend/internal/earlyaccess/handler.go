// backend/internal/earlyaccess/handler.go
package earlyaccess

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		SchemeName string `json:"scheme_name"`
		UnitCount  int32  `json:"unit_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.FullName == "" || req.Email == "" || req.SchemeName == "" || req.UnitCount <= 0 {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "full_name, email, scheme_name, and unit_count are required")
		return
	}
	result, err := h.service.Submit(r.Context(), SubmitParams{
		FullName:   req.FullName,
		Email:      req.Email,
		SchemeName: req.SchemeName,
		UnitCount:  req.UnitCount,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to submit request")
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	results, err := h.service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to list requests")
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	id := chi.URLParam(r, "id")
	result, err := h.service.Approve(r.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "request not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to approve request")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	id := chi.URLParam(r, "id")
	result, err := h.service.Reject(r.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "request not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to reject request")
		return
	}
	response.JSON(w, http.StatusOK, result)
}
