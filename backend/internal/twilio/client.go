package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// requestTimeout bounds any single Twilio request. Set on the http.Client so
// it covers DNS, TLS, write, and read — every phase a Twilio call can hang on.
const requestTimeout = 15 * time.Second

type Client struct {
	httpClient *http.Client
	accountSID string
	authToken  string
	fromNumber string
}

func NewClient(accountSID, authToken, fromNumber string) *Client {
	return &Client{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) SendWhatsAppMessage(to, body string) error {
	endpoint := fmt.Sprintf(
		"https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		c.accountSID,
	)

	data := url.Values{}
	data.Set("From", "whatsapp:"+c.fromNumber)
	data.Set("To", "whatsapp:"+to)
	data.Set("Body", body)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) FromNumber() string {
	return c.fromNumber
}

func ValidateRequest(authToken, urlStr string, formData url.Values, signature string) bool {
	expected := ComputeSignature(authToken, urlStr, formData)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func ComputeSignature(authToken, urlStr string, formData url.Values) string {
	sortedKeys := make([]string, 0, len(formData))
	for k := range formData {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var sb strings.Builder
	sb.WriteString(urlStr)
	for _, k := range sortedKeys {
		vs := formData[k]
		sort.Strings(vs)
		for _, v := range vs {
			sb.WriteString(k)
			sb.WriteString(v)
		}
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
