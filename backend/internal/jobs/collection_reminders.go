package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
)

type CollectionReminderEmailSender interface {
	SendCollectionReminder(ctx context.Context, to, subject, htmlBody string) error
}

type CollectionReminderWhatsAppSender interface {
	SendWhatsAppMessage(to, body string) error
}

type CollectionDeliveryStore interface {
	MarkCollectionEventEmailDelivery(ctx context.Context, arg dbgen.MarkCollectionEventEmailDeliveryParams) (dbgen.CollectionEvent, error)
	MarkCollectionEventWhatsAppDelivery(ctx context.Context, arg dbgen.MarkCollectionEventWhatsAppDeliveryParams) (dbgen.CollectionEvent, error)
}

type CollectionReminderEmailHandler struct {
	store  CollectionDeliveryStore
	sender CollectionReminderEmailSender
}

func NewCollectionReminderEmailHandler(store CollectionDeliveryStore, sender CollectionReminderEmailSender) *CollectionReminderEmailHandler {
	return &CollectionReminderEmailHandler{store: store, sender: sender}
}

func (h *CollectionReminderEmailHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var payload CollectionReminderEmailPayload
	if err := decodePayload(raw, &payload); err != nil {
		return err
	}

	if err := h.sender.SendCollectionReminder(ctx, payload.To, payload.Subject, payload.HTMLBody); err != nil {
		_, markErr := h.store.MarkCollectionEventEmailDelivery(ctx, dbgen.MarkCollectionEventEmailDeliveryParams{
			ID:          payload.CollectionEventID,
			EmailStatus: pgtype.Text{String: "failed", Valid: true},
			EmailError:  pgtype.Text{String: err.Error(), Valid: true},
		})
		if markErr != nil {
			return fmt.Errorf("send email failed and mark failed failed: send=%v mark=%w", err, markErr)
		}
		return err
	}

	_, err := h.store.MarkCollectionEventEmailDelivery(ctx, dbgen.MarkCollectionEventEmailDeliveryParams{
		ID:          payload.CollectionEventID,
		EmailStatus: pgtype.Text{String: "sent", Valid: true},
		EmailError:  pgtype.Text{},
	})
	if err != nil {
		return fmt.Errorf("mark email sent: %w", err)
	}
	return nil
}

type CollectionReminderWhatsAppHandler struct {
	store  CollectionDeliveryStore
	sender CollectionReminderWhatsAppSender
}

func NewCollectionReminderWhatsAppHandler(store CollectionDeliveryStore, sender CollectionReminderWhatsAppSender) *CollectionReminderWhatsAppHandler {
	return &CollectionReminderWhatsAppHandler{store: store, sender: sender}
}

func (h *CollectionReminderWhatsAppHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var payload CollectionReminderWhatsAppPayload
	if err := decodePayload(raw, &payload); err != nil {
		return err
	}

	if err := h.sender.SendWhatsAppMessage(payload.To, payload.Body); err != nil {
		_, markErr := h.store.MarkCollectionEventWhatsAppDelivery(ctx, dbgen.MarkCollectionEventWhatsAppDeliveryParams{
			ID:             payload.CollectionEventID,
			WhatsappStatus: pgtype.Text{String: "failed", Valid: true},
			WhatsappError:  pgtype.Text{String: err.Error(), Valid: true},
		})
		if markErr != nil {
			return fmt.Errorf("send whatsapp failed and mark failed failed: send=%v mark=%w", err, markErr)
		}
		return err
	}

	_, err := h.store.MarkCollectionEventWhatsAppDelivery(ctx, dbgen.MarkCollectionEventWhatsAppDeliveryParams{
		ID:             payload.CollectionEventID,
		WhatsappStatus: pgtype.Text{String: "sent", Valid: true},
		WhatsappError:  pgtype.Text{},
	})
	if err != nil {
		return fmt.Errorf("mark whatsapp sent: %w", err)
	}
	return nil
}
