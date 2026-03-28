package auth

import (
	"encoding/json"
	"net/http"

	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	OrgName  string `json:"org_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.OrgName == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, full_name, and org_name are required")
		return
	}

	res, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName, req.OrgName)
	if err != nil {
		if err == ErrEmailExists {
			response.Error(w, http.StatusConflict, "CONFLICT", "email already registered")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "registration failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email and password are required")
		return
	}

	res, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "login failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "refresh_token is required")
		return
	}

	res, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if err == ErrInvalidToken {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "token refresh failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "refresh_token is required")
		return
	}
	// Idempotent — ignore service errors (token may already be revoked)
	_ = h.service.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)
	orgID, _ := r.Context().Value(OrgIDKey).(string)
	if userID == "" || orgID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}

	res, err := h.service.Me(r.Context(), userID, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch user")
		return
	}
	response.JSON(w, http.StatusOK, res)
}
