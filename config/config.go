package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultBackend string              `toml:"default_backend"`
	Backend        map[string]*Backend `toml:"backend"`
}

type Backend struct {
	Mode  string `toml:"mode"`
	URL   string `toml:"url"`
	Model string `toml:"model"`
}

func Load() (*Config, error) {
	path := os.ExpandEnv("$HOME/.config/ai.toml")
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &config, nil
}
