//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
)

type fakeBillingProvider struct {
	checkoutURL string
	portalURL   string
	suffix      string
}

func (f *fakeBillingProvider) CreateCheckoutSession(_ context.Context, input billing.CheckoutSessionInput) (*billing.CheckoutSession, error) {
	return &billing.CheckoutSession{
		ID:         "cs_test_" + f.suffix,
		URL:        f.checkoutURL,
		CustomerID: ptr("cus_test_" + f.suffix),
	}, nil
}

func (f *fakeBillingProvider) CreatePortalSession(_ context.Context, customerID, returnURL string) (string, error) {
	return f.portalURL, nil
}

func (f *fakeBillingProvider) ParseWebhook(payload []byte, signature string) (*billing.WebhookEvent, error) {
	var event billing.WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func newBillingHandler(t *testing.T) (*billing.Handler, string) {
	t.Helper()
	suffix := fmt.Sprintf("%s_%d", strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_"), time.Now().UnixNano())
	provider := &fakeBillingProvider{
		checkoutURL: "https://checkout.stripe.test/session",
		portalURL:   "https://billing.stripe.test/portal",
		suffix:      suffix,
	}
	return billing.NewHandler(billing.NewService(testPool, provider, "http://localhost:3000")), suffix
}

func TestBilling_CheckoutPortalAndWebhook(t *testing.T) {
	h, suffix := newBillingHandler(t)
	accessToken, orgID := setupAgent(t)

	req := httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.GetSubscription(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get subscription: status=%d body=%s", w.Code, w.Body)
	}
	subscription := decodeSuccess[billing.SubscriptionResponse](t, w)
	if subscription.Status != "inactive" || subscription.HasPortalAccess {
		t.Fatalf("unexpected default subscription: %+v", subscription)
	}

	req = httptest.NewRequest(http.MethodPost, "/billing/checkout", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.CreateCheckoutSession(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create checkout session: status=%d body=%s", w.Code, w.Body)
	}
	checkout := decodeSuccess[billing.CheckoutResponse](t, w)
	if checkout.URL == "" || checkout.SessionID == "" {
		t.Fatalf("unexpected checkout session: %+v", checkout)
	}

	req = httptest.NewRequest(http.MethodPost, "/billing/portal", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.CreatePortalSession(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create portal session: status=%d body=%s", w.Code, w.Body)
	}
	portal := decodeSuccess[billing.PortalResponse](t, w)
	if portal.URL == "" {
		t.Fatalf("unexpected portal response: %+v", portal)
	}

	currentPeriodEnd := time.Now().AddDate(0, 1, 0).UTC()
	webhookBody, _ := json.Marshal(billing.WebhookEvent{
		Type:             "customer.subscription.updated",
		OrgID:            orgID,
		CustomerID:       "cus_test_" + suffix,
		SubscriptionID:   "sub_test_" + suffix,
		Status:           "active",
		PlanCode:         "starter",
		CurrentPeriodEnd: &currentPeriodEnd,
	})
	req = httptest.NewRequest(http.MethodPost, "/billing/webhooks/stripe", bytes.NewReader(webhookBody))
	w = httptest.NewRecorder()
	h.HandleStripeWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("handle webhook: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.GetSubscription(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get updated subscription: status=%d body=%s", w.Code, w.Body)
	}
	updated := decodeSuccess[billing.SubscriptionResponse](t, w)
	expectedSubscriptionID := "sub_test_" + suffix
	if updated.Status != "active" || !updated.EntitlementActive || updated.SubscriptionID == nil || *updated.SubscriptionID != expectedSubscriptionID {
		t.Fatalf("unexpected updated subscription: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "00000000-0000-0000-0000-000000000111", orgID, string(auth.RoleTrustee)))
	w = httptest.NewRecorder()
	h.GetSubscription(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("trustee billing access should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func ptr(value string) *string {
	return &value
}
