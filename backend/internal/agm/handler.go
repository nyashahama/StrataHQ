package agm

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

//nolint:govet // Keep request DTO fields grouped by API meaning rather than field packing.
type scheduleMeetingRequest struct {
	Date           string                      `json:"date"`
	QuorumRequired int32                       `json:"quorum_required"`
	Resolutions    []scheduleResolutionRequest `json:"resolutions"`
}

type scheduleResolutionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type castVoteRequest struct {
	Choice string `json:"choice"`
}

type assignProxyRequest struct {
	GranteeUserID string `json:"grantee_user_id"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeAgmError(w, err, "failed to load AGM dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) ScheduleMeeting(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req scheduleMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	meetingDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Date))
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "date must be YYYY-MM-DD")
		return
	}

	resolutions := make([]ScheduleResolutionInput, 0, len(req.Resolutions))
	for _, resolution := range req.Resolutions {
		resolutions = append(resolutions, ScheduleResolutionInput{
			Title:       strings.TrimSpace(resolution.Title),
			Description: strings.TrimSpace(resolution.Description),
		})
	}

	meeting, err := h.service.ScheduleMeeting(r.Context(), identity, chi.URLParam(r, "schemeId"), ScheduleMeetingInput{
		MeetingDate:    meetingDate,
		QuorumRequired: req.QuorumRequired,
		Resolutions:    resolutions,
	})
	if err != nil {
		writeAgmError(w, err, "failed to schedule AGM")
		return
	}

	response.JSON(w, http.StatusCreated, meeting)
}

func (h *Handler) CastVote(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req castVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	resolution, err := h.service.CastVote(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "resolutionId"), CastVoteInput{
		Choice: strings.TrimSpace(req.Choice),
	})
	if err != nil {
		writeAgmError(w, err, "failed to cast vote")
		return
	}

	response.JSON(w, http.StatusOK, resolution)
}

func (h *Handler) AssignProxy(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req assignProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	if err := h.service.AssignProxy(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "meetingId"), AssignProxyInput{
		GranteeUserID: strings.TrimSpace(req.GranteeUserID),
	}); err != nil {
		writeAgmError(w, err, "failed to assign proxy")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeAgmError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "AGM resource not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
