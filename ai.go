package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/evmar/ai/config"
	"github.com/evmar/ai/google"
	"github.com/evmar/ai/image"
	"github.com/evmar/ai/ollama"
	"github.com/evmar/ai/openai"
)

var (
	flagBackend = flag.String("backend", "", "backend name to use from config")
	flagVerbose = flag.Bool("v", false, "log http")
)

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
	CallText(sys string, json bool, prompts []string) (string, error)
}

func run(args []string) error {
	config, err := config.Load()
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("specify mode")
	}
	mode, args := args[0], args[1:]

	var llm llm
	var oai *openai.Client

	backendName := *flagBackend
	if backendName == "" {
		backendName = config.DefaultBackend
	}
	if backendName == "" {
		return fmt.Errorf("specify -backend or set default_backend in config")
	}
	backend, ok := config.Backend[backendName]
	if !ok {
		return fmt.Errorf("backend %q not found", backendName)
	}

	switch backend.Mode {
	case "":
		return fmt.Errorf("backend %q needs mode= config", backendName)
	case "openai":
		oai, err = openai.New()
		if err != nil {
			return err
		}
		oai.Verbose = *flagVerbose
		llm = oai
	case "ollama":
		c, err := ollama.New(backend)
		if err != nil {
			return err
		}
		llm = c
	case "google":
		c, err := google.New(backend)
		if err != nil {
			return err
		}
		c.Verbose = *flagVerbose
		llm = c
	default:
		return fmt.Errorf("invalid backend mode %q", backend.Mode)
	}

	switch mode {
	case "img":
		flags := flag.NewFlagSet("img", flag.ExitOnError)
		prompt := flags.String("prompt", "", "prompt")
		imagePath := flags.String("image", "", "image path")
		flags.Parse(args)
		if *prompt == "" {
			return fmt.Errorf("specify -prompt")
		}
		if *imagePath == "" {
			return fmt.Errorf("specify -image path")
		}
		imageBytes, err := image.LoadImage(*imagePath)
		if err != nil {
			return err
		}
		msg, err := oai.CallVision(imageBytes, *prompt)
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
		msg, err := llm.CallText(*sys, *json, prompts)
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
		if err := oai.CallSpeech(text, "out.mp3"); err != nil {
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
