package integrations

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const apiKeyPrefix = "shq_live"

var ErrInvalidAPIKey = errors.New("invalid api key")

type GeneratedAPIKey struct {
	Raw    string
	Prefix string
	Hash   string
}

func GenerateAPIKey() (GeneratedAPIKey, error) {
	prefixBytes := make([]byte, 4)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(prefixBytes); err != nil {
		return GeneratedAPIKey{}, err
	}
	if _, err := rand.Read(secretBytes); err != nil {
		return GeneratedAPIKey{}, err
	}
	prefix := hex.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	raw := apiKeyPrefix + "_" + prefix + "_" + secret
	return GeneratedAPIKey{Raw: raw, Prefix: prefix, Hash: HashAPIKey(raw)}, nil
}

func ParseAPIKeyPrefix(raw string) (string, error) {
	parts := strings.SplitN(raw, "_", 4)
	if len(parts) != 4 || parts[0] != "shq" || parts[1] != "live" || parts[2] == "" || parts[3] == "" {
		return "", ErrInvalidAPIKey
	}
	return parts[2], nil
}

func HashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func CompareAPIKeyHash(raw, expectedHash string) bool {
	actual := HashAPIKey(raw)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}
