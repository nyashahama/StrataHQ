// backend/internal/notification/email.go
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Sender is the interface all notification implementations must satisfy.
type Sender interface {
	SendInvitation(ctx context.Context, to, name, inviteURL string) error
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

type EmailClient struct {
	apiKey     string
	fromAddr   string
	httpClient *http.Client
}

func NewEmailClient(apiKey, fromAddr string) *EmailClient {
	return &EmailClient{
		apiKey:     apiKey,
		fromAddr:   fromAddr,
		httpClient: &http.Client{},
	}
}

func (c *EmailClient) SendInvitation(ctx context.Context, to, name, inviteURL string) error {
	subject, body := InvitationEmail(name, inviteURL)
	return c.send(ctx, to, subject, body)
}

func (c *EmailClient) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	subject, body := PasswordResetEmail(resetURL)
	return c.send(ctx, to, subject, body)
}

func (c *EmailClient) send(ctx context.Context, to, subject, htmlBody string) error {
	payload := map[string]any{
		"from":    c.fromAddr,
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	}
	b, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return marshalErr
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend: unexpected status %d", resp.StatusCode)
	}
	return nil
}
