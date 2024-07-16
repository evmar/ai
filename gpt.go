package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

var (
	flagServer  = flag.String("server", "ollama", "ollama or openai")
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

type llm interface {
	callText(sys string, json bool, prompts []string) (string, error)
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("specify mode")
	}
	mode, args := args[0], args[1:]

	var llm llm
	var oai *openAI

	switch *flagServer {
	case "openai":
		oai, err := NewOpenAI()
		if err != nil {
			return err
		}
		llm = oai
	case "ollama":
		ollama, err := NewOllama()
		if err != nil {
			return err
		}
		llm = ollama
	default:
		return fmt.Errorf("-server must be openai or ollama")
	}

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
		msg, err := oai.callVision(imageBytes, *prompt)
		if err != nil {
			return err
		}
		fmt.Println(msg)
		return nil

	case "text":
		flags := flag.NewFlagSet("text", flag.ExitOnError)
		sys := flags.String("sys", "", "system prompt")
		multi := flags.String("multi", "", "multi-shot input")
		json := flags.Bool("json", false, "output json")
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
		msg, err := llm.callText(*sys, *json, prompts)
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
		if err := oai.callSpeech(text, "out.mp3"); err != nil {
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
