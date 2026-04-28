package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func stripeSignatureHeader(payload []byte, secret string, ts time.Time) string {
	signed := fmt.Sprintf("%d.%s", ts.Unix(), payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyStripeSignatureAcceptsValidHeader(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	header := stripeSignatureHeader(payload, "whsec_test", time.Now())

	if err := verifyStripeSignature(payload, header, "whsec_test"); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifyStripeSignatureRejectsMissingHeader(t *testing.T) {
	if err := verifyStripeSignature([]byte(`{}`), "", "whsec_test"); err == nil {
		t.Fatal("missing signature accepted")
	}
}

func TestVerifyStripeSignatureRejectsStaleHeader(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	header := stripeSignatureHeader(payload, "whsec_test", time.Now().Add(-10*time.Minute))

	if err := verifyStripeSignature(payload, header, "whsec_test"); err == nil {
		t.Fatal("stale signature accepted")
	}
}

func TestVerifyStripeSignatureRejectsMismatchedPayload(t *testing.T) {
	header := stripeSignatureHeader([]byte(`{"ok":true}`), "whsec_test", time.Now())

	if err := verifyStripeSignature([]byte(`{"ok":false}`), header, "whsec_test"); err == nil {
		t.Fatal("mismatched payload accepted")
	}
}
