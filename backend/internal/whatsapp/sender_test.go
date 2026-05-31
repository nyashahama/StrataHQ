package whatsapp

import (
	"errors"
	"testing"
)

func TestNoOpSenderReturnsNil(t *testing.T) {
	err := NewNoOpSender().SendWhatsAppMessage("+27820000000", "hello")
	if err != nil {
		t.Fatalf("no-op sender error = %v", err)
	}
}

func TestDisabledSenderReturnsNotConfigured(t *testing.T) {
	err := NewDisabledSender().SendWhatsAppMessage("+27820000000", "hello")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want %q", err, ErrNotConfigured)
	}
}
