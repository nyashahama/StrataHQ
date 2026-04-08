package whatsapp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
)

func verifyTwilioSignature(authToken, path string, params url.Values, signature string) bool {
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

	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func verifyTwilioSignatureOriginalURL(authToken, originalURL string, params url.Values, signature string) bool {
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(originalURL))

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

	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
