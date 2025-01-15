package openai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/evmar/ai/image"
	"github.com/evmar/ai/net"
	"github.com/evmar/ai/rawjson"
)

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("openai: %s", e.Message)
}

func getError(j *rawjson.RJSON) *Error {
	j = j.Get("error")
	if j == nil {
		return nil
	}
	return &Error{
		Message: j.Get("message").String(),
	}
}

type Client struct {
	token   string
	Verbose bool
}

func New() (*Client, error) {
	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		return nil, fmt.Errorf("set OPENAI_API_KEY")
	}
	return &Client{token: openaiToken}, nil
}

func (oai *Client) call(url string, jsonReq map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+oai.token)

	if oai.Verbose {
		http.DefaultClient.Transport = &net.LoggingTransport{}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	processing := resp.Header.Get("Openai-Processing-Ms")
	if processing != "" {
		log.Printf("processing time: %s", processing)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// snip := body
	// if len(snip) > 1000 {
	// 	snip = snip[:1000]
	// }
	return body, err
}

func parse(body []byte) (string, error) {
	j, err := rawjson.Parse(body)
	if err != nil {
		return "", err
	}
	if err := getError(j); err != nil {
		return "", err
	}

	return j.Get("choices").GetIndex(0).Get("message").Get("content").String(), nil
}

func (oai *Client) CallText(sys string, json bool, prompts []string) (string, error) {
	messages := []interface{}{
		map[string]interface{}{
			"role":    "system",
			"content": sys,
		},
	}
	for i, prompt := range prompts {
		var role string
		if i%2 == 0 {
			role = "user"
		} else {
			role = "assistant"
		}
		messages = append(messages, map[string]interface{}{
			"role":    role,
			"content": prompt,
		})
	}

	params := map[string]interface{}{
		"model":      "gpt-3.5-turbo",
		"messages":   messages,
		"max_tokens": 500,
	}
	if json {
		params["response_format"] = map[string]interface{}{"type": "json_object"}
	}

	body, err := oai.call("https://api.openai.com/v1/chat/completions", params)
	if err != nil {
		return "", err
	}
	return parse(body)
}

func (oai *Client) CallVision(image *image.LoadedImage, prompt string) (string, error) {
	body, err := oai.call("https://api.openai.com/v1/chat/completions", map[string]interface{}{
		"model": "gpt-4-vision-preview",
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": prompt,
					},
					map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url":    fmt.Sprintf("data:%s;base64,%s", image.MimeType, base64.StdEncoding.EncodeToString(image.Data)),
							"detail": "high",
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	})
	if err != nil {
		return "", err
	}
	j, err := rawjson.Parse(body)
	if err != nil {
		return "", err
	}
	if err := getError(j); err != nil {
		return "", err
	}

	msg := j.Get("choices").GetIndex(0).Get("message").Get("content").String()
	return msg, nil
}

func (oai *Client) CallSpeech(text, outPath string) error {
	body, err := oai.call("https://api.openai.com/v1/audio/speech", map[string]interface{}{
		"model": "tts-1",
		"input": text,
		"voice": "alloy",
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, body, 0666); err != nil {
		return err
	}
	log.Println("wrote", outPath)
	return nil
}
