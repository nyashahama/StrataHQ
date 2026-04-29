package whatsapp

import (
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

type createBroadcastRequest struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type createMaintenanceFromMessageRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type dismissMaintenanceIntakeRequest struct {
	Dismissed bool `json:"dismissed"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeWhatsAppError(w, err, "failed to load WhatsApp dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) CreateBroadcast(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createBroadcastRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	broadcast, err := h.service.CreateBroadcast(r.Context(), identity, chi.URLParam(r, "schemeId"), CreateBroadcastInput{
		Message: strings.TrimSpace(req.Message),
		Type:    strings.TrimSpace(req.Type),
	})
	if err != nil {
		writeWhatsAppError(w, err, "failed to create WhatsApp broadcast")
		return
	}

	response.JSON(w, http.StatusCreated, broadcast)
}

func (h *Handler) CreateMaintenanceFromMessage(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createMaintenanceFromMessageRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	intake, err := h.service.CreateMaintenanceFromMessage(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "messageId"), CreateMaintenanceFromMessageInput{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Category:    strings.TrimSpace(req.Category),
	})
	if err != nil {
		writeWhatsAppError(w, err, "failed to create maintenance request")
		return
	}

	response.JSON(w, http.StatusCreated, intake)
}

func (h *Handler) DismissMaintenanceIntake(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	intake, err := h.service.DismissMaintenanceIntake(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "intakeId"))
	if err != nil {
		writeWhatsAppError(w, err, "failed to dismiss maintenance intake")
		return
	}

	response.JSON(w, http.StatusOK, intake)
}

func writeWhatsAppError(w http.ResponseWriter, err error, fallback string) {
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
