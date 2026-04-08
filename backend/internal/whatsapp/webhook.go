package whatsapp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/response"
)

type WebhookHandler struct {
	db        *database.Pool
	sender    MessageSender
	bot       *Bot
	logger    *slog.Logger
	authToken string
}

func NewWebhookHandler(db *database.Pool, sender MessageSender, bot *Bot, logger *slog.Logger, authToken string) *WebhookHandler {
	return &WebhookHandler{
		db:        db,
		sender:    sender,
		bot:       bot,
		logger:    logger,
		authToken: authToken,
	}
}

func (h *WebhookHandler) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", h.Inbound)
	r.Get("/", h.Verify)
	return r
}

func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == h.authToken && challenge != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(challenge))
		return
	}

	if r.URL.Query().Get("AccountSid") != "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
}

func (h *WebhookHandler) Inbound(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse webhook form", "error", err)
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid form data")
		return
	}

	from := r.FormValue("From")
	body := r.FormValue("Body")
	profileName := r.FormValue("ProfileName")

	if from == "" || body == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	phoneNumber := strings.TrimPrefix(from, "whatsapp:")

	go h.processMessage(phoneNumber, body, profileName)

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) processMessage(phoneNumber, body, profileName string) {
	ctx := context.Background()

	threads, err := h.db.Q.GetConnectedWhatsAppThreadByPhone(ctx, pgtype.Text{String: phoneNumber, Valid: true})
	if err != nil {
		h.logger.Error("failed to lookup thread by phone", "phone", phoneNumber, "error", err)
		return
	}

	thread := findBestThread(threads)
	if thread == nil {
		h.logger.Warn("no connected thread for phone number", "phone", phoneNumber)
		return
	}

	if _, err := h.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderResident,
		Body:                 body,
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	}); err != nil {
		h.logger.Error("failed to save incoming message", "error", err)
		return
	}

	if err := h.db.Q.IncrementWhatsAppThreadUnread(ctx, thread.ID); err != nil {
		h.logger.Error("failed to increment unread count", "error", err)
	}

	reply, err := h.bot.Respond(ctx, thread.SchemeID, thread.UnitID, body)
	if err != nil {
		h.logger.Error("failed to generate bot response", "error", err)
		return
	}

	if _, err := h.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderBot,
		Body:                 reply,
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	}); err != nil {
		h.logger.Error("failed to save bot response", "error", err)
		return
	}

	if err := h.sender.SendWhatsAppMessage(phoneNumber, reply); err != nil {
		h.logger.Error("failed to send WhatsApp reply", "phone", phoneNumber, "error", err)
	}
}

func findBestThread(threads []dbgen.WhatsappThread) *dbgen.WhatsappThread {
	if len(threads) == 0 {
		return nil
	}
	best := &threads[0]
	for i := range threads {
		if threads[i].LastActiveAt.After(best.LastActiveAt) {
			best = &threads[i]
		}
	}
	return best
}
