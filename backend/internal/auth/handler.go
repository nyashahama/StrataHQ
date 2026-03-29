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
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, and full_name are required")
		return
	}
	res, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName)
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

type setupRequest struct {
	OrgName       string `json:"org_name"`
	ContactEmail  string `json:"contact_email"`
	SchemeName    string `json:"scheme_name"`
	SchemeAddress string `json:"scheme_address"`
	UnitCount     int32  `json:"unit_count"`
}

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can complete onboarding")
		return
	}
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.OrgName == "" || req.ContactEmail == "" || req.SchemeName == "" || req.SchemeAddress == "" || req.UnitCount <= 0 {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "org_name, contact_email, scheme_name, scheme_address, and unit_count are required")
		return
	}
	orgID, _ := r.Context().Value(OrgIDKey).(string)
	res, err := h.service.Setup(r.Context(), orgID, req.OrgName, req.ContactEmail, req.SchemeName, req.SchemeAddress, req.UnitCount)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "onboarding failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req) // best-effort; always 200
	_ = h.service.ForgotPassword(r.Context(), req.Email)
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "token and password are required")
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		if err == ErrInvalidToken {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired reset token")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "password reset failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
