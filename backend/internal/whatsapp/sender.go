package whatsapp

import (
	"errors"
	"log/slog"
)

type MessageSender interface {
	SendWhatsAppMessage(to, body string) error
}

var ErrNotConfigured = errors.New("whatsapp sender not configured")

type NoOpSender struct{}

func NewNoOpSender() *NoOpSender { return &NoOpSender{} }

func (n *NoOpSender) SendWhatsAppMessage(to, body string) error {
	slog.Info("whatsapp: no-op send", "to", to, "body_len", len(body))
	return nil
}

type DisabledSender struct{}

func NewDisabledSender() *DisabledSender { return &DisabledSender{} }

func (n *DisabledSender) SendWhatsAppMessage(to, body string) error {
	slog.Warn("whatsapp sender not configured", "to", to, "body_len", len(body))
	return ErrNotConfigured
}
