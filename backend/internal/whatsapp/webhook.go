package whatsapp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/response"
)

type WebhookHandler struct {
	db            *database.Pool
	sender        MessageSender
	bot           *Bot
	logger        *slog.Logger
	authToken     string
	skipSigVerify bool
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
		_, _ = w.Write([]byte(challenge))
		return
	}

	if r.URL.Query().Get("AccountSid") != "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
}

func (h *WebhookHandler) Inbound(w http.ResponseWriter, r *http.Request) {
	if !h.skipSigVerify {
		sig := r.Header.Get("X-Twilio-Signature")
		if sig == "" {
			h.logger.Warn("missing X-Twilio-Signature header")
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "missing signature")
			return
		}

		if err := r.ParseForm(); err != nil {
			h.logger.Error("failed to parse webhook form", "error", err)
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid form data")
			return
		}

		if !verifyTwilioSignature(h.authToken, r.URL.Path, r.Form, sig) {
			originalURL := r.Header.Get("X-Original-URL")
			if originalURL != "" && verifyTwilioSignatureOriginalURL(h.authToken, originalURL, r.Form, sig) {
			} else {
				h.logger.Warn("invalid Twilio signature", "path", r.URL.Path)
				response.Error(w, http.StatusForbidden, response.CodeForbidden, "invalid signature")
				return
			}
		}
	} else {
		if err := r.ParseForm(); err != nil {
			h.logger.Error("failed to parse webhook form", "error", err)
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid form data")
			return
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	threads, lookupErr := h.db.Q.GetConnectedWhatsAppThreadByPhone(ctx, pgtype.Text{String: phoneNumber, Valid: true})
	if lookupErr != nil {
		h.logger.Error("failed to lookup thread by phone", "phone", phoneNumber, "error", lookupErr)
		return
	}

	thread := findBestThread(threads)
	if thread == nil {
		h.logger.Warn("no connected thread for phone number", "phone", phoneNumber)
		return
	}

	if _, saveErr := h.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderResident,
		Body:                 body,
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	}); saveErr != nil {
		h.logger.Error("failed to save incoming message", "error", saveErr)
		return
	}

	if incrErr := h.db.Q.IncrementWhatsAppThreadUnread(ctx, thread.ID); incrErr != nil {
		h.logger.Error("failed to increment unread count", "error", incrErr)
	}

	reply, botErr := h.bot.Respond(ctx, thread.SchemeID, thread.UnitID, body)
	if botErr != nil {
		h.logger.Error("failed to generate bot response", "error", botErr)
		return
	}

	if _, replyErr := h.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderBot,
		Body:                 reply,
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	}); replyErr != nil {
		h.logger.Error("failed to save bot response", "error", replyErr)
		return
	}

	if sendErr := h.sender.SendWhatsAppMessage(phoneNumber, reply); sendErr != nil {
		h.logger.Error("failed to send WhatsApp reply", "phone", phoneNumber, "error", sendErr)
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
