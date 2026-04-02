package ai

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

//nolint:govet // Keep request DTO fields grouped by API meaning rather than field packing.
type copilotRequest struct {
	SchemeID string    `json:"scheme_id"`
	Message  string    `json:"message"`
	History  []Message `json:"history"`
}

type copilotResponse struct {
	Answer string `json:"answer"`
}

func (h *Handler) Copilot(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req copilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	answer, err := h.service.Ask(r.Context(), identity, strings.TrimSpace(req.SchemeID), req.History, strings.TrimSpace(req.Message))
	if err != nil {
		writeAIError(w, err, "failed to generate copilot response")
		return
	}

	response.JSON(w, http.StatusOK, copilotResponse{Answer: answer})
}

func writeAIError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "AI scope not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
