package ollama

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/evmar/ai/llm"
	"github.com/ollama/ollama/api"
)

type Client struct {
	client *api.Client
	model  string
}

var _ llm.LLM = (*Client)(nil)

func getClientURL(config *llm.BackendConfig) (*url.URL, error) {
	if config.URL != "" {
		return url.Parse(config.URL)
	}

	ollamaHost, err := api.GetOllamaHost()
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Scheme: ollamaHost.Scheme,
		Host:   net.JoinHostPort(ollamaHost.Host, ollamaHost.Port),
	}, nil
}

func New(config *llm.BackendConfig) (*Client, error) {
	clientURL, err := getClientURL(config)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		},
	}
	client := api.NewClient(clientURL, httpClient)
	return &Client{client: client, model: config.Model}, nil
}

func (c *Client) Call(prompt *llm.Prompt) (string, error) {
	if prompt.JSON {
		panic("json not implemented")
	}
	ctx := context.Background()
	if len(prompt.Messages) == 1 {
		if prompt.System != "" {
			panic("system prompt not implemented")
		}

		req := &api.ChatRequest{Model: c.model}
		for i, prompt := range prompt.Messages {
			var role string
			if i%2 == 0 {
				role = "user"
			} else {
				role = "assistant"
			}
			req.Messages = append(req.Messages, api.Message{
				Role:    role,
				Content: prompt,
			})
		}
		resp := func(resp api.ChatResponse) error {
			fmt.Print(resp.Message.Content)
			return nil
		}
		err := c.client.Chat(ctx, req, resp)
		if err != nil {
			return "", err
		}
	}

	// TODO: streaming, not printing
	return "", nil
}
