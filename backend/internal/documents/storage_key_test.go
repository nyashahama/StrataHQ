package documents

import "testing"

func TestNormalizeStorageKeyAllowsMatchingDataURL(t *testing.T) {
	got, err := normalizeStorageKey("data:application/pdf;base64,VEVTVA==", "pdf")
	if err != nil {
		t.Fatalf("normalizeStorageKey() error = %v", err)
	}
	if got != "data:application/pdf;base64,VEVTVA==" {
		t.Fatalf("normalizeStorageKey() = %q", got)
	}
}

func TestNormalizeStorageKeyAllowsRelativePath(t *testing.T) {
	got, err := normalizeStorageKey("/documents/test.pdf", "pdf")
	if err != nil {
		t.Fatalf("normalizeStorageKey() error = %v", err)
	}
	if got != "/documents/test.pdf" {
		t.Fatalf("normalizeStorageKey() = %q", got)
	}
}

func TestNormalizeStorageKeyRejectsJavaScriptURL(t *testing.T) {
	if _, err := normalizeStorageKey("javascript:alert(1)", "pdf"); err != ErrInvalidInput {
		t.Fatalf("normalizeStorageKey() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestNormalizeStorageKeyRejectsMismatchedMimeType(t *testing.T) {
	if _, err := normalizeStorageKey("data:text/html;base64,PHNjcmlwdD4=", "pdf"); err != ErrInvalidInput {
		t.Fatalf("normalizeStorageKey() error = %v, want %v", err, ErrInvalidInput)
	}
}
