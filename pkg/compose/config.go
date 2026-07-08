package compose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Harnesses map[string]HarnessProfile `yaml:"harnesses"`
	Inference map[string]InferenceSpec  `yaml:"inference"`
	MCP       map[string]MCPSpec        `yaml:"mcp"`
	Defaults  Defaults                  `yaml:"defaults"`
	Agents    map[string]Agent          `yaml:"agents"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}
