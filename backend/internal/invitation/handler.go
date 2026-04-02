// backend/internal/invitation/handler.go
package invitation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service    Servicer
	appBaseURL string
}

func NewHandler(service Servicer, appBaseURL string) *Handler {
	return &Handler{service: service, appBaseURL: appBaseURL}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	if !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "only org admins can send invitations")
		return
	}

	var req struct {
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		SchemeID string `json:"scheme_id"`
		UnitID   string `json:"unit_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.FullName == "" || req.Role == "" || req.SchemeID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "email, full_name, role, and scheme_id are required")
		return
	}
	if !auth.IsInvitableRole(req.Role) {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "role must be trustee or resident")
		return
	}
	if req.Role == "resident" && req.UnitID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "unit_id is required for residents")
		return
	}

	inv, err := h.service.Create(r.Context(), identity.OrgID, CreateParams{
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		SchemeID: req.SchemeID,
		UnitID:   req.UnitID,
	}, h.appBaseURL)
	if err != nil {
		switch err.Error() {
		case "invalid scheme_id", "invalid unit_id":
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to create invitation")
		}
		return
	}
	response.JSON(w, http.StatusCreated, inv)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	if !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "only org admins can list invitations")
		return
	}
	invs, err := h.service.List(r.Context(), identity.OrgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to list invitations")
		return
	}
	response.JSON(w, http.StatusOK, invs)
}

func (h *Handler) Resend(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	if !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "only org admins can resend invitations")
		return
	}
	id := chi.URLParam(r, "id")
	inv, err := h.service.Resend(r.Context(), identity.OrgID, id, h.appBaseURL)
	if err != nil {
		switch err {
		case ErrNotFound:
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "invitation not found")
		case ErrForbidden:
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "invitation belongs to a different org")
		default:
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to resend invitation")
		}
		return
	}
	response.JSON(w, http.StatusOK, inv)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	if !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "only org admins can revoke invitations")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Revoke(r.Context(), identity.OrgID, id); err != nil {
		switch err {
		case ErrNotFound:
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "invitation not found")
		case ErrForbidden:
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "invitation belongs to a different org")
		default:
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to revoke invitation")
		}
		return
	}
	response.NoContent(w)
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	v, err := h.service.Verify(r.Context(), token)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired invitation token")
		return
	}
	response.JSON(w, http.StatusOK, v)
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "password is required")
		return
	}
	authResp, err := h.service.Accept(r.Context(), token, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidToken:
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired invitation token")
		case ErrEmailExists:
			response.Error(w, http.StatusConflict, response.CodeConflict, "email already registered")
		default:
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to accept invitation")
		}
		return
	}
	response.JSON(w, http.StatusCreated, authResp)
}
