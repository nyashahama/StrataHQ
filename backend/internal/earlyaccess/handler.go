// backend/internal/earlyaccess/handler.go
package earlyaccess

import (
	"encoding/json"
	"net/http"
	"strconv"

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

func (h *Handler) ApproveWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`))
		return
	}
	_, err = h.service.ApproveByToken(r.Context(), id, sig, exp)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		if err == ErrInvalidToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Link expired</h2><p>This approve link is invalid or has expired.</p></body></html>`))
		} else if err == ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Not found</h2><p>This request no longer exists.</p></body></html>`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Error</h2><p>Something went wrong. Please try again.</p></body></html>`))
		}
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2 style="color:#16a34a">✓ Approved</h2><p>The early access request has been approved. The user will receive an email to set their password.</p></body></html>`))
}

func (h *Handler) RejectWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`))
		return
	}
	_, err = h.service.RejectByToken(r.Context(), id, sig, exp)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		if err == ErrInvalidToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Link expired</h2><p>This reject link is invalid or has expired.</p></body></html>`))
		} else if err == ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Not found</h2><p>This request no longer exists.</p></body></html>`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2>Error</h2><p>Something went wrong. Please try again.</p></body></html>`))
		}
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body style="font-family:sans-serif;padding:40px"><h2 style="color:#dc2626">✗ Rejected</h2><p>The early access request has been rejected.</p></body></html>`))
}
