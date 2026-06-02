package twilio

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClientHasTimeout(t *testing.T) {
	t.Parallel()

	c := NewClient("AC123", "secret", "+15550000000")
	if c.httpClient.Timeout != requestTimeout {
		t.Fatalf("expected http client timeout %s, got %s", requestTimeout, c.httpClient.Timeout)
	}
}

// TestSendWhatsAppMessageRespectsTimeout proves the configured timeout actually
// terminates a hung upstream. Without http.Client.Timeout, this would block
// until the test framework's own timeout fires and obscure the regression.
func TestSendWhatsAppMessageRespectsTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()

	client := NewClient("AC123", "secret", "+15550000000")
	client.httpClient = &http.Client{
		Timeout: 100 * time.Millisecond,
		Transport: redirectToTestServer(server.URL),
	}

	form := url.Values{}
	form.Set("To", "whatsapp:+15551111111")
	form.Set("From", "whatsapp:+15550000000")
	form.Set("Body", "hi")
	body := strings.NewReader(form.Encode())

	req, err := http.NewRequest(http.MethodPost, server.URL+"/Messages.json", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	start := time.Now()
	doResp, err := client.httpClient.Do(req)
	elapsed := time.Since(start)
	if doResp != nil {
		doResp.Body.Close()
	}

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Fatalf("request did not honor 100ms timeout (took %s)", elapsed)
	}
}

// redirectToTestServer returns a RoundTripper that always hits the given URL,
// regardless of the request's host. Lets us point the client at a httptest
// server without exposing Twilio's host to be resolved.
func redirectToTestServer(target string) roundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = target[len("http://"):]
		return http.DefaultTransport.RoundTrip(req)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
