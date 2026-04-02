package communications

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

type createNoticeRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Type  string `json:"type"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.List(r.Context(), identity, chi.URLParam(r, "schemeId"), r.URL.Query().Get("type"))
	if err != nil {
		writeCommunicationsError(w, err, "failed to load notices")
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

	var req createNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	notice, err := h.service.Create(r.Context(), identity, chi.URLParam(r, "schemeId"), CreateNoticeInput{
		Title: strings.TrimSpace(req.Title),
		Body:  strings.TrimSpace(req.Body),
		Type:  strings.TrimSpace(req.Type),
	})
	if err != nil {
		writeCommunicationsError(w, err, "failed to create notice")
		return
	}

	response.JSON(w, http.StatusCreated, notice)
}

func writeCommunicationsError(w http.ResponseWriter, err error, fallback string) {
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
