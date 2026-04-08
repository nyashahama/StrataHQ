package auth

import (
	"unicode"
)

const (
	PasswordMinLength = 8
	PasswordMaxLength = 128
)

type PasswordValidationError struct {
	Message string
}

func (e *PasswordValidationError) Error() string {
	return e.Message
}

func ValidatePassword(password string) error {
	if len(password) < PasswordMinLength {
		return &PasswordValidationError{Message: "password must be at least 8 characters"}
	}
	if len(password) > PasswordMaxLength {
		return &PasswordValidationError{Message: "password must be at most 128 characters"}
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return &PasswordValidationError{Message: "password must contain at least one uppercase letter"}
	}
	if !hasLower {
		return &PasswordValidationError{Message: "password must contain at least one lowercase letter"}
	}
	if !hasDigit {
		return &PasswordValidationError{Message: "password must contain at least one digit"}
	}
	if !hasSpecial {
		return &PasswordValidationError{Message: "password must contain at least one special character"}
	}

	return nil
}
