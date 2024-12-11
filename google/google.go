package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bitly/go-simplejson"
	"github.com/evmar/ai/net"
)

type Client struct {
	apikey  string
	Verbose bool
}

func New() (*Client, error) {
	apikey := os.Getenv("GOOGLE_API_KEY")
	if apikey == "" {
		return nil, fmt.Errorf("set GOOGLE_API_KEY")
	}
	return &Client{apikey: apikey}, nil
}

func (c *Client) call(jsonReq map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + c.apikey

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if c.Verbose {
		http.DefaultClient.Transport = &net.LoggingTransport{}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

func parseText(body []byte) (string, error) {
	j, err := simplejson.NewJson(body)
	if err != nil {
		return "", err
	}

	cand := j.Get("candidates").GetIndex(0)
	content := cand.Get("content")
	parts := content.Get("parts").MustArray()
	if len(parts) != 1 {
		return "", fmt.Errorf("expected 1 part, got %d", len(parts))
	}
	text := parts[0].(map[string]interface{})["text"].(string)

	return text, nil
}

func (c *Client) CallText(sys string, json bool, prompts []string) (string, error) {
	parts := []map[string]interface{}{}
	for _, prompt := range prompts {
		parts = append(parts, map[string]interface{}{
			"text": prompt,
		})
	}

	jsonReq := map[string]interface{}{
		"contents": map[string]interface{}{
			"parts": parts,
		},
	}

	out, err := c.call(jsonReq)
	if err != nil {
		return "", err
	}
	return parseText(out)
}
