package whatsapp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"testing"
)

func TestVerifyTwilioSignature_Valid(t *testing.T) {
	authToken := "test-auth-token"
	path := "/api/v1/whatsapp/webhooks"
	params := url.Values{
		"From":       {"+1234567890"},
		"Body":       {"Hello"},
		"AccountSid": {"AC123"},
	}
	sig := computeTestSignature(authToken, path, params)

	if !verifyTwilioSignature(authToken, path, params, sig) {
		t.Error("expected valid signature to pass")
	}
}

func TestVerifyTwilioSignature_InvalidSignature(t *testing.T) {
	authToken := "test-auth-token"
	path := "/api/v1/whatsapp/webhooks"
	params := url.Values{
		"From": {"+1234567890"},
		"Body": {"Hello"},
	}

	if verifyTwilioSignature(authToken, path, params, "invalidsig") {
		t.Error("expected invalid signature to fail")
	}
}

func TestVerifyTwilioSignature_WrongToken(t *testing.T) {
	path := "/api/v1/whatsapp/webhooks"
	params := url.Values{
		"From": {"+1234567890"},
		"Body": {"Hello"},
	}
	sig := computeTestSignature("correct-token", path, params)

	if verifyTwilioSignature("wrong-token", path, params, sig) {
		t.Error("expected signature with wrong token to fail")
	}
}

func TestVerifyTwilioSignature_ParamOrdering(t *testing.T) {
	authToken := "test-auth-token"
	path := "/api/v1/whatsapp/webhooks"
	params := url.Values{
		"From": {"+1234567890"},
		"Body": {"Test message"},
		"To":   {"+0987654321"},
	}
	sig := computeTestSignature(authToken, path, params)

	if !verifyTwilioSignature(authToken, path, params, sig) {
		t.Error("expected valid signature with multiple params to pass")
	}
}

func computeTestSignature(authToken, path string, params url.Values) string {
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(path))

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		for _, v := range params[k] {
			mac.Write([]byte(k))
			mac.Write([]byte(v))
		}
	}

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
