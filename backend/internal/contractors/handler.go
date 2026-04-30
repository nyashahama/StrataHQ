package contractors

import (
	"net/http"
	"strconv"

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

type contractorRequest struct {
	Phone         *string  `json:"phone"`
	Email         *string  `json:"email"`
	Notes         *string  `json:"notes"`
	Name          string   `json:"name"`
	Trade         string   `json:"trade"`
	Suburb        string   `json:"suburb"`
	City          string   `json:"city"`
	Province      string   `json:"province"`
	SchemeIDs     []string `json:"scheme_ids"`
	PublicProfile bool     `json:"public_profile"`
	Vetted        bool     `json:"vetted"`
	Active        bool     `json:"active"`
}

type reviewRequest struct {
	Comment              *string `json:"comment"`
	SchemeID             string  `json:"scheme_id"`
	MaintenanceRequestID string  `json:"maintenance_request_id"`
	Rating               int32   `json:"rating"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	result, err := h.service.List(r.Context(), identity, contractorFiltersFromRequest(r))
	if err != nil {
		writeContractorError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Marketplace(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	schemeID := r.URL.Query().Get("scheme_id")
	result, err := h.service.Marketplace(r.Context(), identity, schemeID, contractorFiltersFromRequest(r))
	if err != nil {
		writeContractorError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	var req contractorRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	result, err := h.service.Create(r.Context(), identity, contractorInput(req))
	if err != nil {
		writeContractorError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	var req contractorRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	result, err := h.service.Update(r.Context(), identity, chi.URLParam(r, "contractorId"), contractorInput(req))
	if err != nil {
		writeContractorError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	var req reviewRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	result, err := h.service.CreateReview(r.Context(), identity, chi.URLParam(r, "contractorId"), ReviewInput{
		SchemeID: req.SchemeID, MaintenanceRequestID: req.MaintenanceRequestID,
		Rating: req.Rating, Comment: req.Comment,
	})
	if err != nil {
		writeContractorError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func contractorInput(req contractorRequest) UpsertContractorInput {
	return UpsertContractorInput{
		Phone: req.Phone, Email: req.Email, Notes: req.Notes, Name: req.Name, Trade: req.Trade,
		Suburb: req.Suburb, City: req.City, Province: req.Province, SchemeIDs: req.SchemeIDs,
		PublicProfile: req.PublicProfile, Vetted: req.Vetted, Active: req.Active,
	}
}

func contractorFiltersFromRequest(r *http.Request) ContractorFilters {
	query := r.URL.Query()
	return ContractorFilters{
		SchemeID: query.Get("scheme_id"),
		Trade:    query.Get("trade"),
		Suburb:   query.Get("suburb"),
		Query:    query.Get("q"),
		Vetted:   boolQuery(query.Get("vetted")),
		Active:   boolQuery(query.Get("active")),
	}
}

func boolQuery(raw string) *bool {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}

func writeContractorError(w http.ResponseWriter, err error) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "contractor not found")
	case ErrConflict:
		response.Error(w, http.StatusConflict, response.CodeConflict, "contractor conflict")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "contractor operation failed")
	}
}
