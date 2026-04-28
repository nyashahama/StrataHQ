package audit

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *ResourceService
}

func NewHandler(service *ResourceService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListSchemeEvents(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.ErrorWithRequest(w, r, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			response.ErrorWithRequest(w, r, http.StatusBadRequest, response.CodeBadRequest, "invalid limit")
			return
		}
		limit = int32(parsed)
	}
	result, err := h.service.ListSchemeEvents(r.Context(), identity, chi.URLParam(r, "schemeId"), limit)
	if err != nil {
		if errors.Is(err, ErrInvalidResourceEvent) {
			response.ErrorWithRequest(w, r, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.ErrorWithRequest(w, r, http.StatusForbidden, response.CodeForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.ErrorWithRequest(w, r, http.StatusNotFound, response.CodeNotFound, "scheme not found")
			return
		}
		response.ErrorWithRequest(w, r, http.StatusInternalServerError, response.CodeInternalError, "failed to load audit events")
		return
	}
	response.JSON(w, http.StatusOK, result)
}
