package twilio

import (
	"testing"
)

func TestNewClientHasTimeout(t *testing.T) {
	t.Parallel()

	c := NewClient("AC123", "secret", "+15550000000")
	if c.httpClient.Timeout != requestTimeout {
		t.Fatalf("expected http client timeout %s, got %s", requestTimeout, c.httpClient.Timeout)
	}
}
