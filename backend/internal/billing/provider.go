package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StripeProvider struct {
	secretKey     string
	webhookSecret string
	priceID       string
	client        *http.Client
}

func NewStripeProvider(secretKey, webhookSecret, priceID string) *StripeProvider {
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		priceID:       priceID,
		client:        &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, input CheckoutSessionInput) (*CheckoutSession, error) {
	if p.secretKey == "" || p.priceID == "" {
		return nil, ErrNotConfigured
	}

	values := url.Values{}
	values.Set("mode", "subscription")
	values.Set("success_url", input.SuccessURL)
	values.Set("cancel_url", input.CancelURL)
	values.Set("line_items[0][price]", p.priceID)
	values.Set("line_items[0][quantity]", "1")
	values.Set("metadata[org_id]", input.OrgID)
	values.Set("metadata[plan_code]", input.PlanCode)
	values.Set("subscription_data[metadata][org_id]", input.OrgID)
	values.Set("subscription_data[metadata][plan_code]", input.PlanCode)
	values.Set("client_reference_id", input.OrgID)
	if input.CustomerID != nil && *input.CustomerID != "" {
		values.Set("customer", *input.CustomerID)
	} else if input.CustomerEmail != nil && *input.CustomerEmail != "" {
		values.Set("customer_email", *input.CustomerEmail)
	}

	var response struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Customer string `json:"customer"`
	}
	if err := p.postForm(ctx, "/v1/checkout/sessions", values, &response); err != nil {
		return nil, err
	}

	session := &CheckoutSession{
		ID:  response.ID,
		URL: response.URL,
	}
	if response.Customer != "" {
		session.CustomerID = &response.Customer
	}
	return session, nil
}

func (p *StripeProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	if p.secretKey == "" {
		return "", ErrNotConfigured
	}
	values := url.Values{}
	values.Set("customer", customerID)
	values.Set("return_url", returnURL)

	var response struct {
		URL string `json:"url"`
	}
	if err := p.postForm(ctx, "/v1/billing_portal/sessions", values, &response); err != nil {
		return "", err
	}
	return response.URL, nil
}

func (p *StripeProvider) ParseWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	if p.webhookSecret == "" {
		return nil, ErrNotConfigured
	}
	if err := verifyStripeSignature(payload, signature, p.webhookSecret); err != nil {
		return nil, ErrInvalidInput
	}

	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, ErrInvalidInput
	}

	switch event.Type {
	case "checkout.session.completed":
		var object stripeCheckoutSession
		if err := json.Unmarshal(event.Data.Object, &object); err != nil {
			return nil, ErrInvalidInput
		}
		return &WebhookEvent{
			Type:              event.Type,
			OrgID:             object.Metadata.OrgID,
			CustomerID:        object.Customer,
			SubscriptionID:    object.Subscription,
			CheckoutSessionID: object.ID,
			Status:            "checkout_completed",
			PlanCode:          object.Metadata.PlanCode,
		}, nil
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var object stripeSubscription
		if err := json.Unmarshal(event.Data.Object, &object); err != nil {
			return nil, ErrInvalidInput
		}
		var currentPeriodEnd *time.Time
		if object.CurrentPeriodEnd > 0 {
			value := time.Unix(object.CurrentPeriodEnd, 0).UTC()
			currentPeriodEnd = &value
		}
		return &WebhookEvent{
			Type:              event.Type,
			OrgID:             object.Metadata.OrgID,
			CustomerID:        object.Customer,
			SubscriptionID:    object.ID,
			Status:            object.Status,
			PlanCode:          pickString(object.Metadata.PlanCode, object.FirstPriceID()),
			CurrentPeriodEnd:  currentPeriodEnd,
			CancelAtPeriodEnd: object.CancelAtPeriodEnd,
		}, nil
	default:
		return &WebhookEvent{Type: event.Type}, nil
	}
}

func (p *StripeProvider) postForm(ctx context.Context, path string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com"+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe API error: %s", strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

type stripeEvent struct {
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeMetadata struct {
	OrgID    string `json:"org_id"`
	PlanCode string `json:"plan_code"`
}

type stripeCheckoutSession struct {
	ID           string         `json:"id"`
	Customer     string         `json:"customer"`
	Subscription string         `json:"subscription"`
	Metadata     stripeMetadata `json:"metadata"`
}

type stripeSubscription struct {
	ID                string         `json:"id"`
	Customer          string         `json:"customer"`
	Status            string         `json:"status"`
	CurrentPeriodEnd  int64          `json:"current_period_end"`
	CancelAtPeriodEnd bool           `json:"cancel_at_period_end"`
	Metadata          stripeMetadata `json:"metadata"`
	Items             struct {
		Data []struct {
			Price struct {
				ID string `json:"id"`
			} `json:"price"`
		} `json:"data"`
	} `json:"items"`
}

func (s stripeSubscription) FirstPriceID() string {
	if len(s.Items.Data) == 0 {
		return ""
	}
	return s.Items.Data[0].Price.ID
}

func verifyStripeSignature(payload []byte, header, secret string) error {
	if header == "" {
		return errors.New("missing signature")
	}

	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("invalid signature header")
	}

	secs, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	if time.Since(time.Unix(secs, 0)) > 5*time.Minute {
		return errors.New("stale signature")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, signature := range signatures {
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return nil
		}
	}
	return errors.New("signature mismatch")
}
