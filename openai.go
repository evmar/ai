package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/bitly/go-simplejson"
)

type loggingTransport struct{}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, respBytes...)

	fmt.Printf("%s\n", bytes)

	return resp, err
}

type openAI struct {
	token string
}

func NewOpenAI() (*openAI, error) {
	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		return nil, fmt.Errorf("set OPENAI_API_KEY")
	}
	return &openAI{token: openaiToken}, nil
}

func (oai *openAI) call(url string, jsonReq map[string]interface{}) ([]byte, error) {
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

	if *flagVerbose {
		http.DefaultClient.Transport = &loggingTransport{}
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

func (oai *openAI) callText(sys string, prompts []string) (string, error) {
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

	body, err := oai.call("https://api.openai.com/v1/chat/completions", map[string]interface{}{
		"model":      "gpt-3.5-turbo",
		"messages":   messages,
		"max_tokens": 500,
	})
	if err != nil {
		return "", err
	}
	j, err := simplejson.NewJson(body)
	if err != nil {
		return "", err
	}

	msg := j.Get("choices").GetIndex(0).Get("message").Get("content").MustString()
	return msg, nil
}

func (oai *openAI) callVision(image *loadedImage, prompt string) (string, error) {
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
							"url":    fmt.Sprintf("data:%s;base64,%s", image.mimeType, base64.StdEncoding.EncodeToString(image.data)),
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
	j, err := simplejson.NewJson(body)
	if err != nil {
		return "", err
	}

	msg := j.Get("choices").GetIndex(0).Get("message").Get("content").MustString()
	return msg, nil

}

func (oai *openAI) callSpeech(text, outPath string) error {
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
