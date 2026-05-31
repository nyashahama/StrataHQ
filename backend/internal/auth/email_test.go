package auth

import "testing"

func TestValidateEmail_Valid(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.co.za",
		"a@b.io",
	}
	for _, e := range valid {
		if !ValidateEmail(e) {
			t.Errorf("ValidateEmail(%q) = false, want true", e)
		}
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"notanemail",
		"@example.com",
		"user@",
		"user@.com",
		"user..double@example.com",
		"User Example <user@example.com>",
	}
	for _, e := range invalid {
		if ValidateEmail(e) {
			t.Errorf("ValidateEmail(%q) = true, want false", e)
		}
	}
}

func TestNormalizeEmail_CanonicalizesMailboxAddress(t *testing.T) {
	email, err := NormalizeEmail("  User.Name+Tag@Example.COM  ")
	if err != nil {
		t.Fatalf("NormalizeEmail() error = %v", err)
	}
	if email != "user.name+tag@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want %q", email, "user.name+tag@example.com")
	}
}
