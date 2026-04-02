package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Completer interface {
	Complete(ctx context.Context, systemPrompt string, history []Message, message string) (string, error)
}

type Client struct {
	model  string
	client openai.Client
}

func NewClient(cfg Config) *Client {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)

	return &Client{
		client: client,
		model:  cfg.Model,
	}
}

func (c *Client) Complete(ctx context.Context, systemPrompt string, history []Message, message string) (string, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}
	for _, item := range history {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		switch item.Role {
		case "assistant":
			messages = append(messages, openai.AssistantMessage(content))
		default:
			messages = append(messages, openai.UserMessage(content))
		}
	}
	messages = append(messages, openai.UserMessage(strings.TrimSpace(message)))

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
