package billing

import (
	"net/http"
	"github.com/stratahq/backend/internal/platform/response"
)

func (h *Handler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "stripe webhook received"})
}
