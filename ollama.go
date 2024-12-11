package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/evmar/ai/config"
	"github.com/ollama/ollama/api"
)

type ollama struct {
	client *api.Client
	model  string
}

func getClientURL(config *config.Backend) (*url.URL, error) {
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

func NewOllama(config *config.Backend) (*ollama, error) {
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
	return &ollama{client: client, model: config.Model}, nil
}

func (oll *ollama) CallText(sys string, json bool, prompts []string) (string, error) {
	if json {
		panic("json not implemented")
	}
	ctx := context.Background()
	if len(prompts) == 1 {
		req := &api.GenerateRequest{Model: oll.model, Prompt: prompts[0]}
		if sys != "" {
			req.System = sys
		}

		resp := func(resp api.GenerateResponse) error {
			fmt.Print(resp.Response)
			return nil
		}
		err := oll.client.Generate(ctx, req, resp)
		if err != nil {
			return "", err
		}
	} else {
		req := &api.ChatRequest{Model: oll.model}
		for i, prompt := range prompts {
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
		err := oll.client.Chat(ctx, req, resp)
		if err != nil {
			return "", err
		}
	}

	return "", nil
}
