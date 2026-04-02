package documents

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

type createDocumentRequest struct {
	Name       string `json:"name"`
	StorageKey string `json:"storage_key"`
	FileType   string `json:"file_type"`
	Category   string `json:"category"`
	SizeBytes  int64  `json:"size_bytes"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.List(r.Context(), identity, chi.URLParam(r, "schemeId"), r.URL.Query().Get("category"))
	if err != nil {
		writeDocumentsError(w, err, "failed to load documents")
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

	var req createDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	document, err := h.service.Create(r.Context(), identity, chi.URLParam(r, "schemeId"), CreateDocumentInput{
		Name:       strings.TrimSpace(req.Name),
		StorageKey: strings.TrimSpace(req.StorageKey),
		FileType:   strings.TrimSpace(req.FileType),
		Category:   strings.TrimSpace(req.Category),
		SizeBytes:  req.SizeBytes,
	})
	if err != nil {
		writeDocumentsError(w, err, "failed to create document")
		return
	}

	response.JSON(w, http.StatusCreated, document)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	if err := h.service.Delete(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "id")); err != nil {
		writeDocumentsError(w, err, "failed to delete document")
		return
	}

	response.NoContent(w)
}

func writeDocumentsError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "document not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
