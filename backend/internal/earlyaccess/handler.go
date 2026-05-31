// backend/internal/earlyaccess/handler.go
package earlyaccess

import (
	"html"
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
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		SchemeName    string `json:"scheme_name"`
		UnitCount     int32  `json:"unit_count"`
		HoneypotField string `json:"website"`
	}
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	schemeName := strings.TrimSpace(req.SchemeName)

	if fullName == "" || email == "" || schemeName == "" || req.UnitCount <= 0 {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "full_name, email, scheme_name, and unit_count are required")
		return
	}
	if len(fullName) > 120 || len(schemeName) > 180 || len(email) > 254 || req.UnitCount > 100000 {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "full_name, email, scheme_name, and unit_count are required")
		return
	}
	if !auth.ValidateEmail(email) {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid email format")
		return
	}
	if req.HoneypotField != "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
		return
	}

	result, err := h.service.Submit(r.Context(), SubmitParams{
		FullName:   fullName,
		Email:      email,
		SchemeName: schemeName,
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
		if err == ErrAlreadyReviewed {
			response.Error(w, http.StatusConflict, response.CodeConflict, "request already reviewed")
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
		if err == ErrAlreadyReviewed {
			response.Error(w, http.StatusConflict, response.CodeConflict, "request already reviewed")
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

func decisionPage(title, body, actionURL, actionLabel string) string {
	return "<!DOCTYPE html><html><body><h2>" + html.EscapeString(title) + "</h2><p>" + html.EscapeString(body) + "</p><form method=\"POST\" action=\"" + html.EscapeString(actionURL) + "\"><button type=\"submit\">" + html.EscapeString(actionLabel) + "</button></form></body></html>"
}

func messagePage(title, body string) string {
	return "<!DOCTYPE html><html><body><h2>" + html.EscapeString(title) + "</h2><p>" + html.EscapeString(body) + "</p></body></html>"
}

func (h *Handler) ApproveWithTokenPage(w http.ResponseWriter, r *http.Request) {
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	id := chi.URLParam(r, "id")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" || h.service.ValidateActionToken(id, "approve", sig, exp) != nil {
		writeHTML(w, http.StatusBadRequest, messagePage("Invalid link", "This link is malformed."))
		return
	}

	actionURL := r.URL.RequestURI()
	writeHTML(w, http.StatusOK, decisionPage("Approve request", "Confirm that you want to approve this early access request.", actionURL, "Approve request"))
}

func (h *Handler) ApproveWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, messagePage("Invalid link", "This link is malformed."))
		return
	}
	_, err = h.service.ApproveByToken(r.Context(), id, sig, exp)
	if err != nil {
		if err == ErrInvalidToken {
			writeHTML(w, http.StatusUnauthorized, messagePage("Link expired", "This approve link is invalid or has expired."))
		} else if err == ErrAlreadyReviewed {
			writeHTML(w, http.StatusConflict, messagePage("Already reviewed", "This early access request has already been reviewed."))
		} else if err == ErrNotFound {
			writeHTML(w, http.StatusNotFound, messagePage("Not found", "This request no longer exists."))
		} else {
			writeHTML(w, http.StatusInternalServerError, messagePage("Error", "Something went wrong. Please try again."))
		}
		return
	}
	writeHTML(w, http.StatusOK, messagePage("Approved", "The early access request has been approved. The user will receive an email to set their password."))
}

func (h *Handler) RejectWithTokenPage(w http.ResponseWriter, r *http.Request) {
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	id := chi.URLParam(r, "id")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" || h.service.ValidateActionToken(id, "reject", sig, exp) != nil {
		writeHTML(w, http.StatusBadRequest, messagePage("Invalid link", "This link is malformed."))
		return
	}

	actionURL := r.URL.RequestURI()
	writeHTML(w, http.StatusOK, decisionPage("Reject request", "Confirm that you want to reject this early access request.", actionURL, "Reject request"))
}

func (h *Handler) RejectWithToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sig := r.URL.Query().Get("sig")
	expStr := r.URL.Query().Get("exp")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || sig == "" {
		writeHTML(w, http.StatusBadRequest, messagePage("Invalid link", "This link is malformed."))
		return
	}
	_, err = h.service.RejectByToken(r.Context(), id, sig, exp)
	if err != nil {
		if err == ErrInvalidToken {
			writeHTML(w, http.StatusUnauthorized, messagePage("Link expired", "This reject link is invalid or has expired."))
		} else if err == ErrAlreadyReviewed {
			writeHTML(w, http.StatusConflict, messagePage("Already reviewed", "This early access request has already been reviewed."))
		} else if err == ErrNotFound {
			writeHTML(w, http.StatusNotFound, messagePage("Not found", "This request no longer exists."))
		} else {
			writeHTML(w, http.StatusInternalServerError, messagePage("Error", "Something went wrong. Please try again."))
		}
		return
	}
	writeHTML(w, http.StatusOK, messagePage("Rejected", "The early access request has been rejected."))
}
