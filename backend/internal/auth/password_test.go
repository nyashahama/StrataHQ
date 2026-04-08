package auth

import "testing"

func TestValidatePassword_TooShort(t *testing.T) {
	if err := ValidatePassword("Ab1!"); err == nil {
		t.Error("expected error for short password")
	}
}

func TestValidatePassword_TooLong(t *testing.T) {
	long := "Aa1!" + string(make([]byte, 125))
	if err := ValidatePassword(long); err == nil {
		t.Error("expected error for long password")
	}
}

func TestValidatePassword_NoUppercase(t *testing.T) {
	if err := ValidatePassword("abcdefg1!"); err == nil {
		t.Error("expected error for no uppercase")
	}
}

func TestValidatePassword_NoLowercase(t *testing.T) {
	if err := ValidatePassword("ABCDEFG1!"); err == nil {
		t.Error("expected error for no lowercase")
	}
}

func TestValidatePassword_NoDigit(t *testing.T) {
	if err := ValidatePassword("Abcdefg!"); err == nil {
		t.Error("expected error for no digit")
	}
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	if err := ValidatePassword("Abcdefg1"); err == nil {
		t.Error("expected error for no special character")
	}
}

func TestValidatePassword_Valid(t *testing.T) {
	passwords := []string{"Pass_1234", "Abcdefg1!", "MyP@ssw0rd", "Hello_World1"}
	for _, pw := range passwords {
		if err := ValidatePassword(pw); err != nil {
			t.Errorf("ValidatePassword(%q) = %v, want nil", pw, err)
		}
	}
}
