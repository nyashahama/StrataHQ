package whatsapp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInboundRejectsMissingTwilioSignature(t *testing.T) {
	handler := NewWebhookHandler(nil, nil, nil, nil, slog.Default(), "twilio-token")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhooks", strings.NewReader("From=whatsapp%3A%2B27123456789&Body=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.Inbound(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestInboundRejectsInvalidTwilioSignature(t *testing.T) {
	handler := NewWebhookHandler(nil, nil, nil, nil, slog.Default(), "twilio-token")

	form := url.Values{}
	form.Set("From", "whatsapp:+27123456789")
	form.Set("Body", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhooks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", "invalid")
	w := httptest.NewRecorder()

	handler.Inbound(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestInboundReturnsServiceUnavailableWhenWorkerQueueIsFull(t *testing.T) {
	handler := NewWebhookHandler(nil, nil, nil, nil, slog.Default(), "twilio-token")
	handler.SetSkipSigVerify(true)
	for range cap(handler.workerSlots) {
		handler.workerSlots <- struct{}{}
	}

	form := url.Values{}
	form.Set("From", "whatsapp:+27123456789")
	form.Set("Body", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhooks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.Inbound(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestWebhookHandlerHasAuthToken(t *testing.T) {
	handlerWithToken := NewWebhookHandler(nil, nil, nil, nil, slog.Default(), "twilio-token")
	if !handlerWithToken.HasAuthToken() {
		t.Fatal("expected HasAuthToken to return true when token is configured")
	}

	handlerWithoutToken := NewWebhookHandler(nil, nil, nil, nil, slog.Default(), "")
	if handlerWithoutToken.HasAuthToken() {
		t.Fatal("expected HasAuthToken to return false when token is missing")
	}
}
