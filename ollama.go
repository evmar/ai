package main

import (
	"context"
	"fmt"

	"github.com/ollama/ollama/api"
)

type ollama struct {
	client *api.Client
	model  string
}

func NewOllama(config *ConfigBackend) (*ollama, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &ollama{client: client, model: config.Model}, nil
}

func (oll *ollama) callText(sys string, json bool, prompts []string) (string, error) {
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
