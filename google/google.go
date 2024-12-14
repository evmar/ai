package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/evmar/ai/config"
	"github.com/evmar/ai/net"
)

type Client struct {
	apikey  string
	model   string
	Verbose bool
}

func New(config *config.Backend) (*Client, error) {
	apikey := os.Getenv("GOOGLE_API_KEY")
	if apikey == "" {
		return nil, fmt.Errorf("set GOOGLE_API_KEY")
	}
	return &Client{apikey: apikey, model: config.Model}, nil
}

func (c *Client) call(jsonReq map[string]interface{}) (io.Reader, error) {
	body, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s", c.model, c.apikey)

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
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func parseText(body []byte) (string, error) {
	var resp GenerateContentResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&resp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	text := resp.Candidates[0].Content.Parts[0].Text
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

	r, err := c.call(jsonReq)
	if err != nil {
		return "", err
	}

	s := NewStreamedReader(r)
	var body string
	var resp GenerateContentResponse
	for {
		if err := s.Read(&resp); err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}
		body += resp.Candidates[0].Content.Parts[0].Text
	}
	return body, nil
}
