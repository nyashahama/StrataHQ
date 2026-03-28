package scheme

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "list schemes"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "get scheme", "id": id})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusCreated, map[string]string{"message": "create scheme"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "update scheme", "id": id})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "delete scheme", "id": id})
}
