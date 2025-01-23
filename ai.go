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

type TTS interface {
	CallSpeech(text, outPath string) error
}

func getBackend(config *llm.Config, name string) (llm.LLM, error) {
	if name == "" {
		name = config.DefaultBackend
	}
	if name == "" {
		return nil, fmt.Errorf("specify -backend or set default_backend in config")
	}
	cfg, ok := config.Backend[name]
	if !ok {
		return nil, fmt.Errorf("backend %q not found", name)
	}

	switch cfg.Mode {
	case "":
		return nil, fmt.Errorf("backend %q needs mode= config", name)
	case "openai":
		c, err := openai.New()
		if err != nil {
			return nil, err
		}
		c.Verbose = *flagVerbose
		return c, nil
	case "ollama":
		c, err := ollama.New(cfg)
		if err != nil {
			return nil, err
		}
		return c, nil
	case "google":
		c, err := google.New(cfg)
		if err != nil {
			return nil, err
		}
		c.Verbose = *flagVerbose
		return c, nil
	default:
		return nil, fmt.Errorf("invalid backend mode %q", cfg.Mode)
	}
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

	backend, err := getBackend(config, *flagBackend)
	if err != nil {
		return err
	}

	switch mode {
	case "text":
		prompt := &llm.Prompt{}

		{
			flags := flag.NewFlagSet("text", flag.ExitOnError)
			flags.StringVar(&prompt.System, "sys", "", "system prompt")
			multi := flags.String("multi", "", "multi-shot input")
			flags.BoolVar(&prompt.JSON, "json", false, "output json")
			flags.Func("image", "image to attach", func(val string) error {
				img, err := image.LoadImage(val)
				if err != nil {
					return err
				}
				prompt.Images = append(prompt.Images, img)
				return nil
			})
			flags.Parse(args)

			if *multi != "" {
				msgs, err := parseMulti(*multi)
				if err != nil {
					return err
				}
				prompt.Messages = append(prompt.Messages, msgs...)
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
		tts, ok := backend.(TTS)
		if !ok {
			return fmt.Errorf("backend doesn't support TTS")
		}

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
		if err := tts.CallSpeech(text, "out.mp3"); err != nil {
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

	return fmt.Errorf("invalid mode, must be one of {text,tts,config}")
}

func main() {
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
