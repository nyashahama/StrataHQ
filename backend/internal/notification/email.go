package notification

type EmailClient struct {
	apiKey string
}

func NewEmailClient(apiKey string) *EmailClient {
	return &EmailClient{apiKey: apiKey}
}
