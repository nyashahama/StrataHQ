package whatsapp

import "log/slog"

type MessageSender interface {
	SendWhatsAppMessage(to, body string) error
}

type NoOpSender struct{}

func NewNoOpSender() *NoOpSender { return &NoOpSender{} }

func (n *NoOpSender) SendWhatsAppMessage(to, body string) error {
	slog.Info("whatsapp: no-op send", "to", to, "body_len", len(body))
	return nil
}
