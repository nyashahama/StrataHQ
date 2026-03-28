package maintenance

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
	response.JSON(w, http.StatusOK, map[string]string{"message": "list maintenance requests"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "get maintenance request", "id": id})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusCreated, map[string]string{"message": "create maintenance request"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "update maintenance request", "id": id})
}
