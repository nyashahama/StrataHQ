package earlyaccess

import (
	"testing"
	"time"
)

func TestValidateActionToken_RejectsEmptySecret(t *testing.T) {
	id := "request-123"
	action := "approve"
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := generateActionToken("", id, action, exp)

	if validateActionToken("", id, action, sig, exp) {
		t.Fatal("expected empty secret token validation to fail")
	}
}

func TestValidateActionToken_AcceptsValidSignedToken(t *testing.T) {
	id := "request-123"
	action := "approve"
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := generateActionToken("platform-secret", id, action, exp)

	if !validateActionToken("platform-secret", id, action, sig, exp) {
		t.Fatal("expected valid token to pass")
	}
}
