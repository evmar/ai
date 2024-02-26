package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
)

var (
	flagVerbose = flag.Bool("v", false, "log http")
)

type loadedImage struct {
	mimeType string
	data     []byte
}

func loadImage(imagePath string) (*loadedImage, error) {
	mimeType := ""
	switch ext := path.Ext(imagePath); ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	default:
		return nil, fmt.Errorf("unknown ext %s", ext)
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	return &loadedImage{mimeType: mimeType, data: data}, nil
}

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

func callAPI(url string, jsonReq map[string]interface{}) ([]byte, error) {
	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		return nil, fmt.Errorf("set OPENAI_API_KEY")
	}
	body, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+openaiToken)

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

func callText(sys string, prompts []string) (string, error) {
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

	body, err := callAPI("https://api.openai.com/v1/chat/completions", map[string]interface{}{
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

func callVision(image *loadedImage, prompt string) (string, error) {
	body, err := callAPI("https://api.openai.com/v1/chat/completions", map[string]interface{}{
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

func callSpeech(text, outPath string) error {
	body, err := callAPI("https://api.openai.com/v1/audio/speech", map[string]interface{}{
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

func parseMulti(multi string) ([]string, error) {
	parts := strings.SplitAfterN(multi, "\n", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("expected separator as first line of multi")
	}
	sep, rest := parts[0], parts[1]
	parts = strings.Split(rest, sep)
	var prompts []string
	for _, prompt := range parts {
		prompt = strings.TrimSpace(prompt)
		if len(prompt) == 0 {
			continue
		}
		prompts = append(prompts, prompt)
	}
	if len(parts) < 2 {
		return nil, fmt.Errorf("didn't find separator %q in prompt", sep)
	}
	if len(parts)%2 != 0 {
		return nil, fmt.Errorf("expected even number of parts in multi")
	}
	return prompts, nil
}

func argOrStdin(arg string) (string, error) {
	if arg == "-" {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(buf), nil
	}
	return arg, nil
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("specify mode")
	}
	mode, args := args[0], args[1:]
	switch mode {
	case "img":
		flags := flag.NewFlagSet("img", flag.ExitOnError)
		prompt := flags.String("prompt", "", "prompt")
		image := flags.String("image", "", "image path")
		flags.Parse(args)
		if *prompt == "" {
			return fmt.Errorf("specify -prompt")
		}
		if *image == "" {
			return fmt.Errorf("specify -image path")
		}
		imageBytes, err := loadImage(*image)
		if err != nil {
			return err
		}
		msg, err := callVision(imageBytes, *prompt)
		if err != nil {
			return err
		}
		fmt.Println(msg)
		return nil

	case "text":
		flags := flag.NewFlagSet("text", flag.ExitOnError)
		sys := flags.String("sys", "", "system prompt")
		multi := flags.String("multi", "", "multi-shot input")
		flags.Parse(args)
		if *sys == "" {
			return fmt.Errorf("specify -sys")
		}
		prompts := []string{}
		if *multi != "" {
			var err error
			prompts, err = parseMulti(*multi)
			if err != nil {
				return err
			}
		}
		args = flags.Args()
		if len(args) != 1 {
			return fmt.Errorf("specify prompt")
		}
		prompt, err := argOrStdin(args[0])
		if err != nil {
			return err
		}
		prompts = append(prompts, prompt)
		msg, err := callText(*sys, prompts)
		if err != nil {
			return err
		}
		fmt.Println(msg)
		return nil

	case "tts":
		flags := flag.NewFlagSet("tts", flag.ExitOnError)
		flags.Parse(args)
		args = flags.Args()
		if len(args) != 1 {
			return fmt.Errorf("specify text")
		}
		text, err := argOrStdin(args[0])
		if err != nil {
			return err
		}
		if err := callSpeech(text, "out.mp3"); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("invalid mode")
}

func main() {
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
