package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/evmar/ai/google"
	"github.com/evmar/ai/image"
	"github.com/evmar/ai/llm"
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

type LLM interface {
	// TODO: streaming only
	Call(prompt *llm.Prompt) (string, error)
}

func run(args []string) error {
	config, err := llm.LoadConfig()
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("specify mode")
	}
	mode, args := args[0], args[1:]

	var backend LLM
	var oai *openai.Client

	backendName := *flagBackend
	if backendName == "" {
		backendName = config.DefaultBackend
	}
	if backendName == "" {
		return fmt.Errorf("specify -backend or set default_backend in config")
	}
	backendConfig, ok := config.Backend[backendName]
	if !ok {
		return fmt.Errorf("backend %q not found", backendName)
	}

	switch backendConfig.Mode {
	case "":
		return fmt.Errorf("backend %q needs mode= config", backendName)
	case "openai":
		oai, err = openai.New()
		if err != nil {
			return err
		}
		oai.Verbose = *flagVerbose
		backend = oai
	case "ollama":
		c, err := ollama.New(backendConfig)
		if err != nil {
			return err
		}
		backend = c
	case "google":
		c, err := google.New(backendConfig)
		if err != nil {
			return err
		}
		c.Verbose = *flagVerbose
		backend = c
	default:
		return fmt.Errorf("invalid backend mode %q", backendConfig.Mode)
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
		prompt := &llm.Prompt{}

		{
			flags := flag.NewFlagSet("text", flag.ExitOnError)
			flags.StringVar(&prompt.System, "sys", "", "system prompt")
			multi := flags.String("multi", "", "multi-shot input")
			flags.BoolVar(&prompt.JSON, "json", false, "output json")
			flags.Parse(args)

			if *multi != "" {
				msgs, err := parseMulti(*multi)
				if err != nil {
					return err
				}
				for _, msg := range msgs {
					prompt.Messages = append(prompt.Messages, msg)
				}
			}
			args = flags.Args()
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			if len(args) == 1 {
				arg, err := argOrStdin(args[0])
				if err != nil {
					return err
				}
				prompt.Messages = append(prompt.Messages, arg)
			}
		}

		if s, ok := backend.(llm.Streamed); ok {
			stream, err := s.CallStreamed(prompt.System, prompt.JSON, prompt.Messages)
			if err != nil {
				return err
			}
			for {
				msg, err := stream.Next()
				if err != nil {
					if err != io.EOF {
						return err
					}
					break
				}
				fmt.Print(msg)
			}
		} else {
			msg, err := backend.Call(prompt)
			if err != nil {
				return err
			}
			fmt.Println(msg)
		}
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

	case "config":
		fmt.Println("config file:", llm.ConfigPath())
		t, err := config.ToTOML()
		if err != nil {
			return err
		}
		fmt.Println(t)
		return nil
	}

	return fmt.Errorf("invalid mode, must be one of {img,text,tts,config}")
}

func main() {
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
