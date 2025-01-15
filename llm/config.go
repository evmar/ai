package llm

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultBackend string                    `toml:"default_backend"`
	Backend        map[string]*BackendConfig `toml:"backend"`
}

type BackendConfig struct {
	Mode  string `toml:"mode"`
	URL   string `toml:"url"`
	Model string `toml:"model"`
}

func ConfigPath() string {
	return os.ExpandEnv("$HOME/.config/ai.toml")
}

func LoadConfig() (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(ConfigPath(), &config); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &config, nil
}

func (c *Config) ToTOML() (string, error) {
	b, err := toml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	return string(b), nil
}
