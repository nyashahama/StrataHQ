package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	dbgen "github.com/stratahq/backend/db/gen"
)

type fakeEmailSender struct {
	to      string
	subject string
	body    string
	err     error
}

func (f *fakeEmailSender) SendCollectionReminder(ctx context.Context, to, subject, htmlBody string) error {
	f.to = to
	f.subject = subject
	f.body = htmlBody
	return f.err
}

type fakeWhatsAppSender struct {
	to   string
	body string
	err  error
}

func (f *fakeWhatsAppSender) SendWhatsAppMessage(to, body string) error {
	f.to = to
	f.body = body
	return f.err
}

type fakeCollectionDeliveryStore struct {
	emailStatus      pgtype.Text
	emailError       pgtype.Text
	whatsappStatus   pgtype.Text
	whatsappError    pgtype.Text
	emailMarkErr     error
	whatsappMarkErr  error
}

func (f *fakeCollectionDeliveryStore) MarkCollectionEventEmailDelivery(ctx context.Context, arg dbgen.MarkCollectionEventEmailDeliveryParams) (dbgen.CollectionEvent, error) {
	f.emailStatus = arg.EmailStatus
	f.emailError = arg.EmailError
	if f.emailMarkErr != nil {
		return dbgen.CollectionEvent{}, f.emailMarkErr
	}
	return dbgen.CollectionEvent{ID: arg.ID}, nil
}

func (f *fakeCollectionDeliveryStore) MarkCollectionEventWhatsAppDelivery(ctx context.Context, arg dbgen.MarkCollectionEventWhatsAppDeliveryParams) (dbgen.CollectionEvent, error) {
	f.whatsappStatus = arg.WhatsappStatus
	f.whatsappError = arg.WhatsappError
	if f.whatsappMarkErr != nil {
		return dbgen.CollectionEvent{}, f.whatsappMarkErr
	}
	return dbgen.CollectionEvent{ID: arg.ID}, nil
}

func TestCollectionReminderEmailHandlerMarksSent(t *testing.T) {
	eventID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	store := &fakeCollectionDeliveryStore{}
	sender := &fakeEmailSender{}
	handler := NewCollectionReminderEmailHandler(store, sender)
	payload, err := json.Marshal(CollectionReminderEmailPayload{
		CollectionEventID: eventID,
		To:                "owner@example.com",
		Subject:           "Reminder",
		HTMLBody:          "<p>Pay</p>",
	})
	require.NoError(t, err)

	err = handler.Handle(context.Background(), payload)

	require.NoError(t, err)
	require.Equal(t, "owner@example.com", sender.to)
	require.Equal(t, "Reminder", sender.subject)
	require.Equal(t, "<p>Pay</p>", sender.body)
	require.Equal(t, pgtype.Text{String: "sent", Valid: true}, store.emailStatus)
	require.False(t, store.emailError.Valid)
}

func TestCollectionReminderEmailHandlerReturnsProviderError(t *testing.T) {
	eventID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	store := &fakeCollectionDeliveryStore{}
	sender := &fakeEmailSender{err: errors.New("resend unavailable")}
	handler := NewCollectionReminderEmailHandler(store, sender)
	payload, err := json.Marshal(CollectionReminderEmailPayload{
		CollectionEventID: eventID,
		To:                "owner@example.com",
		Subject:           "Reminder",
		HTMLBody:          "<p>Pay</p>",
	})
	require.NoError(t, err)

	err = handler.Handle(context.Background(), payload)

	require.Error(t, err)
	require.Contains(t, err.Error(), "resend unavailable")
	require.Equal(t, pgtype.Text{String: "failed", Valid: true}, store.emailStatus)
	require.Equal(t, pgtype.Text{String: "resend unavailable", Valid: true}, store.emailError)
}

func TestCollectionReminderWhatsAppHandlerMarksSent(t *testing.T) {
	eventID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	store := &fakeCollectionDeliveryStore{}
	sender := &fakeWhatsAppSender{}
	handler := NewCollectionReminderWhatsAppHandler(store, sender)
	payload, err := json.Marshal(CollectionReminderWhatsAppPayload{
		CollectionEventID: eventID,
		To:                "+27820000000",
		Body:              "Please pay",
	})
	require.NoError(t, err)

	err = handler.Handle(context.Background(), payload)

	require.NoError(t, err)
	require.Equal(t, "+27820000000", sender.to)
	require.Equal(t, "Please pay", sender.body)
	require.Equal(t, pgtype.Text{String: "sent", Valid: true}, store.whatsappStatus)
	require.False(t, store.whatsappError.Valid)
}

func TestCollectionReminderEmailHandlerReturnsStoreErrorAfterSend(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	store := &fakeCollectionDeliveryStore{emailMarkErr: errors.New("db unavailable")}
	sender := &fakeEmailSender{}
	handler := NewCollectionReminderEmailHandler(store, sender)
	payload, err := json.Marshal(CollectionReminderEmailPayload{
		CollectionEventID: eventID,
		To:                "owner@example.com",
		Subject:           "Reminder",
		HTMLBody:          "<p>Pay</p>",
	})
	require.NoError(t, err)

	err = handler.Handle(context.Background(), payload)

	require.Error(t, err)
	require.Contains(t, err.Error(), "db unavailable")
}

func TestCollectionReminderWhatsAppHandlerReturnsStoreErrorAfterSend(t *testing.T) {
	eventID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	store := &fakeCollectionDeliveryStore{whatsappMarkErr: errors.New("db unavailable")}
	sender := &fakeWhatsAppSender{}
	handler := NewCollectionReminderWhatsAppHandler(store, sender)
	payload, err := json.Marshal(CollectionReminderWhatsAppPayload{
		CollectionEventID: eventID,
		To:                "+27820000000",
		Body:              "Please pay",
	})
	require.NoError(t, err)

	err = handler.Handle(context.Background(), payload)

	require.Error(t, err)
	require.Contains(t, err.Error(), "db unavailable")
}

func TestCollectionReminderWhatsAppHandlerRejectsBadPayload(t *testing.T) {
	store := &fakeCollectionDeliveryStore{}
	sender := &fakeWhatsAppSender{}
	handler := NewCollectionReminderWhatsAppHandler(store, sender)

	err := handler.Handle(context.Background(), json.RawMessage(`{"to":"","body":"x"}`))

	require.ErrorIs(t, err, ErrBadPayload)
}
