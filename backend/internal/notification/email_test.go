package notification

import (
	"testing"
)

func TestNewEmailClientHasTimeout(t *testing.T) {
	t.Parallel()

	c := NewEmailClient("test-api-key", "from@example.com")
	if c.httpClient.Timeout != emailRequestTimeout {
		t.Fatalf("expected http client timeout %s, got %s", emailRequestTimeout, c.httpClient.Timeout)
	}
}
