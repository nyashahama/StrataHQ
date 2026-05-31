package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
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
	service       *Service
	logger        *slog.Logger
	authToken     string
	skipSigVerify bool
	workerCtx     context.Context
	cancelWorkers context.CancelFunc
	workerSlots   chan struct{}
}

const webhookWorkerCapacity = 64

func NewWebhookHandler(db *database.Pool, sender MessageSender, bot *Bot, service *Service, logger *slog.Logger, authToken string) *WebhookHandler {
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	return &WebhookHandler{
		db:            db,
		sender:        sender,
		bot:           bot,
		service:       service,
		logger:        logger,
		authToken:     authToken,
		workerCtx:     workerCtx,
		cancelWorkers: cancelWorkers,
		workerSlots:   make(chan struct{}, webhookWorkerCapacity),
	}
}

func (h *WebhookHandler) SetSkipSigVerify(v bool) {
	h.skipSigVerify = v
}

func (h *WebhookHandler) Close() {
	if h.cancelWorkers != nil {
		h.cancelWorkers()
	}
}

func (h *WebhookHandler) HasAuthToken() bool {
	return strings.TrimSpace(h.authToken) != ""
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

	media := parseInboundMedia(r.Form)

	if from == "" || (body == "" && len(media) == 0) {
		w.WriteHeader(http.StatusOK)
		return
	}

	phoneNumber := strings.TrimPrefix(from, "whatsapp:")

	select {
	case h.workerSlots <- struct{}{}:
	default:
		response.Error(w, http.StatusServiceUnavailable, response.CodeInternalError, "webhook processor busy")
		return
	}

	workerCtx := h.workerCtx
	if workerCtx == nil {
		workerCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(workerCtx, 30*time.Second)

	threads, lookupErr := h.db.Q.GetConnectedWhatsAppThreadByPhone(ctx, pgtype.Text{String: phoneNumber, Valid: true})
	if lookupErr != nil {
		cancel()
		<-h.workerSlots
		h.logger.Error("failed to lookup thread by phone", "phone", phoneNumber, "error", lookupErr)
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "thread lookup failed")
		return
	}

	thread := findBestThread(threads)
	if thread == nil {
		cancel()
		<-h.workerSlots
		h.logger.Warn("no connected thread for phone number", "phone", phoneNumber)
		w.WriteHeader(http.StatusOK)
		return
	}

	incoming, saveErr := h.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderResident,
		Body:                 body,
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	})
	if saveErr != nil {
		cancel()
		<-h.workerSlots
		h.logger.Error("failed to save incoming message", "error", saveErr)
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "message save failed")
		return
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("webhook message processing panic",
					"error", fmt.Sprint(r),
					"stack", string(debug.Stack()))
			}
			cancel()
			<-h.workerSlots
		}()
		h.processMessageAfterSave(ctx, phoneNumber, body, profileName, media, thread, incoming)
	}()
}

func (h *WebhookHandler) processMessageAfterSave(ctx context.Context, phoneNumber, body, profileName string, media []inboundMedia, thread *dbgen.WhatsappThread, incoming dbgen.WhatsappMessage) {
	for _, item := range media {
		if _, err := h.db.Q.CreateWhatsAppMessageMedia(ctx, dbgen.CreateWhatsAppMessageMediaParams{
			MessageID:        incoming.ID,
			Provider:         "twilio",
			ProviderMediaSid: item.ProviderMediaSID,
			MediaUrl:         item.URL,
			ContentType:      item.ContentType,
		}); err != nil {
			h.logger.Error("failed to save whatsapp media", "error", err)
		}
	}

	if incrErr := h.db.Q.IncrementWhatsAppThreadUnread(ctx, thread.ID); incrErr != nil {
		h.logger.Error("failed to increment unread count", "error", incrErr)
	}

	var reply string

	classification := classifyMaintenanceIntent(body, len(media))
	if classification.IsMaintenance {
		text := buildMaintenanceIntakeText(body, len(media))
		intake, err := h.service.createMaintenanceTicketForMessage(
			ctx,
			thread.SchemeID,
			thread.ID,
			incoming.ID,
			thread.UnitID,
			text.Title,
			text.Description,
			classification.Category,
			len(media),
		)
		if err != nil {
			h.logger.Error("failed to create whatsapp maintenance ticket", "message_id", incoming.ID, "error", err)
			reply = "I received your maintenance message, but could not create the ticket automatically. Please try again or contact your managing agent."
		} else if intake.MaintenanceRequestID != nil {
			reply = fmt.Sprintf("Thanks. I've logged a maintenance request from your WhatsApp message.\n\nRef: %s\nStatus: Pending approval", (*intake.MaintenanceRequestID)[:8])
		}
	} else if shouldCreateMaintenanceCandidate(classification, len(media)) {
		text := buildMaintenanceIntakeText(body, len(media))
		if _, err := h.db.Q.CreateWhatsAppMaintenanceIntake(ctx, dbgen.CreateWhatsAppMaintenanceIntakeParams{
			SchemeID:             thread.SchemeID,
			ThreadID:             thread.ID,
			MessageID:            incoming.ID,
			UnitID:               thread.UnitID,
			MaintenanceRequestID: pgtype.UUID{},
			Status:               "candidate",
			Category:             dbgen.MaintenanceCategory(classification.Category),
			Title:                text.Title,
			Description:          text.Description,
			MediaCount:           int32(len(media)),
		}); err != nil {
			h.logger.Error("failed to create whatsapp maintenance candidate", "message_id", incoming.ID, "error", err)
		}
	}

	if reply == "" {
		var botErr error
		reply, botErr = h.bot.Respond(ctx, thread.SchemeID, thread.UnitID, body)
		if botErr != nil {
			h.logger.Error("failed to generate bot response", "error", botErr)
			return
		}
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

type inboundMedia struct {
	ProviderMediaSID pgtype.Text
	ContentType      pgtype.Text
	URL              string
}

func parseInboundMedia(form url.Values) []inboundMedia {
	count, _ := strconv.Atoi(form.Get("NumMedia"))
	media := make([]inboundMedia, 0, count)
	for i := 0; i < count; i++ {
		urlValue := strings.TrimSpace(form.Get(fmt.Sprintf("MediaUrl%d", i)))
		if urlValue == "" {
			continue
		}
		item := inboundMedia{URL: urlValue}
		if sid := strings.TrimSpace(form.Get(fmt.Sprintf("MediaSid%d", i))); sid != "" {
			item.ProviderMediaSID = pgtype.Text{String: sid, Valid: true}
		}
		if contentType := strings.TrimSpace(form.Get(fmt.Sprintf("MediaContentType%d", i))); contentType != "" {
			item.ContentType = pgtype.Text{String: contentType, Valid: true}
		}
		media = append(media, item)
	}
	return media
}
