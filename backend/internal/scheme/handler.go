package scheme

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

type schemeRequest struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	UnitCount int32  `json:"unit_count"`
}

type unitRequest struct {
	Identifier      string `json:"identifier"`
	OwnerName       string `json:"owner_name"`
	Floor           int32  `json:"floor"`
	SectionValueBps int32  `json:"section_value_bps"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	schemes, err := h.service.List(r.Context(), identity)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to list schemes")
		return
	}

	response.JSON(w, http.StatusOK, schemes)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	detail, err := h.service.Get(r.Context(), identity, chi.URLParam(r, "id"))
	if err != nil {
		writeSchemeError(w, err, "failed to fetch scheme")
		return
	}

	response.JSON(w, http.StatusOK, detail)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req schemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	input, valid := normalizeSchemeRequest(req)
	if !valid {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "name, address, and unit_count are required")
		return
	}

	created, err := h.service.Create(r.Context(), identity, input)
	if err != nil {
		writeSchemeError(w, err, "failed to create scheme")
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req schemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	input, valid := normalizeSchemeRequest(req)
	if !valid {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "name, address, and unit_count are required")
		return
	}

	updated, err := h.service.Update(r.Context(), identity, chi.URLParam(r, "id"), UpdateSchemeInput(input))
	if err != nil {
		writeSchemeError(w, err, "failed to update scheme")
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	if err := h.service.Delete(r.Context(), identity, chi.URLParam(r, "id")); err != nil {
		writeSchemeError(w, err, "failed to delete scheme")
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListUnits(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	units, err := h.service.ListUnits(r.Context(), identity, chi.URLParam(r, "id"))
	if err != nil {
		writeSchemeError(w, err, "failed to list units")
		return
	}

	response.JSON(w, http.StatusOK, units)
}

func (h *Handler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req unitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	input, valid := normalizeUnitRequest(req)
	if !valid {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "identifier, owner_name, floor, and section_value_bps are required")
		return
	}

	unit, err := h.service.CreateUnit(r.Context(), identity, chi.URLParam(r, "id"), input)
	if err != nil {
		writeSchemeError(w, err, "failed to create unit")
		return
	}

	response.JSON(w, http.StatusCreated, unit)
}

func (h *Handler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req unitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	input, valid := normalizeUnitRequest(req)
	if !valid {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "identifier, owner_name, floor, and section_value_bps are required")
		return
	}

	unit, err := h.service.UpdateUnit(r.Context(), identity, chi.URLParam(r, "id"), chi.URLParam(r, "unitId"), UpdateUnitInput(input))
	if err != nil {
		writeSchemeError(w, err, "failed to update unit")
		return
	}

	response.JSON(w, http.StatusOK, unit)
}

func normalizeSchemeRequest(req schemeRequest) (CreateSchemeInput, bool) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	if name == "" || address == "" || req.UnitCount <= 0 {
		return CreateSchemeInput{}, false
	}
	return CreateSchemeInput{
		Name:      name,
		Address:   address,
		UnitCount: req.UnitCount,
	}, true
}

func normalizeUnitRequest(req unitRequest) (CreateUnitInput, bool) {
	identifier := strings.TrimSpace(req.Identifier)
	ownerName := strings.TrimSpace(req.OwnerName)
	if identifier == "" || ownerName == "" || req.SectionValueBps <= 0 {
		return CreateUnitInput{}, false
	}
	return CreateUnitInput{
		Identifier:      identifier,
		OwnerName:       ownerName,
		Floor:           req.Floor,
		SectionValueBps: req.SectionValueBps,
	}, true
}

func writeSchemeError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "scheme not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid scheme identifier")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
