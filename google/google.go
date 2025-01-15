package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/evmar/ai/llm"
	"github.com/evmar/ai/net"
)

type Client struct {
	apikey  string
	model   string
	Verbose bool
}

func New(config *llm.BackendConfig) (*Client, error) {
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
	panic("todo")
}

type Stream struct {
	s *StreamedReader
}

func (s *Stream) Next() (string, error) {
	var resp GenerateContentResponse
	if err := s.s.Read(&resp); err != nil {
		return "", err
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func (c *Client) CallStreamed(sys string, json bool, prompts []string) (llm.Stream, error) {
	// Confusingly, the docs say that the "Content" type used for system_instruction and contents
	// should have a parts[] array, but in fact it's just a single part.

	contents := []map[string]interface{}{}
	for _, prompt := range prompts {
		contents = append(contents, map[string]interface{}{
			"parts": map[string]interface{}{
				"text": prompt,
			},
		})
	}

	jsonReq := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": map[string]interface{}{
				"text": sys,
			},
		},
		"contents": contents,
	}

	r, err := c.call(jsonReq)
	if err != nil {
		return nil, err
	}

	return &Stream{s: NewStreamedReader(r)}, nil
}

var _ llm.Streamed = &Client{}
