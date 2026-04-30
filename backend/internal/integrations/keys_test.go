package integrations

import "testing"

func TestGenerateAPIKeyParseHashAndCompare(t *testing.T) {
	generated, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	if generated.Raw == "" || generated.Prefix == "" || generated.Hash == "" {
		t.Fatalf("generated key missing fields: %+v", generated)
	}

	prefix, err := ParseAPIKeyPrefix(generated.Raw)
	if err != nil {
		t.Fatalf("parse prefix: %v", err)
	}
	if prefix != generated.Prefix {
		t.Fatalf("prefix = %q, want %q", prefix, generated.Prefix)
	}
	if !CompareAPIKeyHash(generated.Raw, generated.Hash) {
		t.Fatalf("generated raw key should match hash")
	}
	if CompareAPIKeyHash(generated.Raw+"x", generated.Hash) {
		t.Fatalf("modified key should not match hash")
	}
}

func TestParseAPIKeyPrefixRejectsInvalidFormat(t *testing.T) {
	values := []string{"", "Bearer abc", "shq_live_onlytwo", "shq_test_abc_def"}
	for _, value := range values {
		if _, err := ParseAPIKeyPrefix(value); err == nil {
			t.Fatalf("expected parse error for %q", value)
		}
	}
}
