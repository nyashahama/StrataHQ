package auth

import (
	"net/mail"
	"strings"
)

func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	if len(email) > 254 {
		return false
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return false
	}
	if !strings.Contains(email, "@") {
		return false
	}
	local := strings.SplitN(email, "@", 2)[0]
	return local != ""
}
