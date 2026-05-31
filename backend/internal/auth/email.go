package auth

import (
	"errors"
	"net/mail"
	"strings"
)

// NormalizeEmail trims and validates email syntax into a canonical lowercase address.
func NormalizeEmail(raw string) (string, error) {
	email := strings.TrimSpace(raw)
	if email == "" {
		return "", errors.New("email is required")
	}
	if len(email) > 254 {
		return "", errors.New("email is too long")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", err
	}
	// Reject display-name forms such as "Bob <bob@example.com>".
	if addr.Name != "" || addr.Address != email {
		return "", errors.New("invalid email format")
	}

	local := strings.SplitN(addr.Address, "@", 2)[0]
	if local == "" || !strings.Contains(addr.Address, "@") {
		return "", errors.New("invalid email format")
	}

	return strings.ToLower(addr.Address), nil
}

func NormalizeOptionalEmail(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	normalized, err := NormalizeEmail(trimmed)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func ValidateEmail(email string) bool {
	_, err := NormalizeEmail(email)
	return err == nil
}
