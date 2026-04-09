// backend/internal/earlyaccess/handler.go
package earlyaccess

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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
	if err := response.DecodeJSON(r.Body, &req); err != nil {
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
		if err == ErrForbidden {
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
			return
		}
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
		if err == ErrForbidden {
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
			return
		}
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
		if err == ErrForbidden {
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
			return
		}
		if err == ErrNotFound {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "request not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to reject request")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		slog.Error("failed to write html response", "status", status, "error", err)
	}
}

func (h *Handler) ApproveWithTokenPage(w http.ResponseWriter, r *http.Request) {
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	if _, err := strconv.ParseInt(expStr, 10, 64); err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, `<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`)
		return
	}

	actionURL := htmlEscapeAttribute(r.URL.RequestURI())
	writeHTML(w, http.StatusOK, `<html><body style="font-family:sans-serif;padding:40px"><h2>Approve request</h2><p>Confirm that you want to approve this early access request.</p><form method="POST" action="`+actionURL+`"><button type="submit" style="background:#16a34a;color:white;border:0;border-radius:8px;padding:12px 18px;font:inherit;cursor:pointer">Approve request</button></form></body></html>`)
}

func (h *Handler) ApproveWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, `<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`)
		return
	}
	_, err = h.service.ApproveByToken(r.Context(), id, sig, exp)
	if err != nil {
		if err == ErrInvalidToken {
			writeHTML(w, http.StatusUnauthorized, `<html><body style="font-family:sans-serif;padding:40px"><h2>Link expired</h2><p>This approve link is invalid or has expired.</p></body></html>`)
		} else if err == ErrNotFound {
			writeHTML(w, http.StatusNotFound, `<html><body style="font-family:sans-serif;padding:40px"><h2>Not found</h2><p>This request no longer exists.</p></body></html>`)
		} else {
			writeHTML(w, http.StatusInternalServerError, `<html><body style="font-family:sans-serif;padding:40px"><h2>Error</h2><p>Something went wrong. Please try again.</p></body></html>`)
		}
		return
	}
	writeHTML(w, http.StatusOK, `<html><body style="font-family:sans-serif;padding:40px"><h2 style="color:#16a34a">Approved</h2><p>The early access request has been approved. The user will receive an email to set their password.</p></body></html>`)
}

func (h *Handler) RejectWithTokenPage(w http.ResponseWriter, r *http.Request) {
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	if _, err := strconv.ParseInt(expStr, 10, 64); err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, `<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`)
		return
	}

	actionURL := htmlEscapeAttribute(r.URL.RequestURI())
	writeHTML(w, http.StatusOK, `<html><body style="font-family:sans-serif;padding:40px"><h2>Reject request</h2><p>Confirm that you want to reject this early access request.</p><form method="POST" action="`+actionURL+`"><button type="submit" style="background:#dc2626;color:white;border:0;border-radius:8px;padding:12px 18px;font:inherit;cursor:pointer">Reject request</button></form></body></html>`)
}

func (h *Handler) RejectWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, `<html><body style="font-family:sans-serif;padding:40px"><h2>Invalid link</h2><p>This link is malformed.</p></body></html>`)
		return
	}
	_, err = h.service.RejectByToken(r.Context(), id, sig, exp)
	if err != nil {
		if err == ErrInvalidToken {
			writeHTML(w, http.StatusUnauthorized, `<html><body style="font-family:sans-serif;padding:40px"><h2>Link expired</h2><p>This reject link is invalid or has expired.</p></body></html>`)
		} else if err == ErrNotFound {
			writeHTML(w, http.StatusNotFound, `<html><body style="font-family:sans-serif;padding:40px"><h2>Not found</h2><p>This request no longer exists.</p></body></html>`)
		} else {
			writeHTML(w, http.StatusInternalServerError, `<html><body style="font-family:sans-serif;padding:40px"><h2>Error</h2><p>Something went wrong. Please try again.</p></body></html>`)
		}
		return
	}
	writeHTML(w, http.StatusOK, `<html><body style="font-family:sans-serif;padding:40px"><h2 style="color:#dc2626">Rejected</h2><p>The early access request has been rejected.</p></body></html>`)
}

func htmlEscapeAttribute(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`"`, "&quot;",
		`'`, "&#39;",
		`<`, "&lt;",
		`>`, "&gt;",
	)
	return replacer.Replace(value)
}
