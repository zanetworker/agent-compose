package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`
harnesses:
  claude-code:
    image: ghcr.io/anthropics/claude-code:latest
    env-mapping:
      endpoint: ANTHROPIC_BASE_URL
      key: ANTHROPIC_API_KEY
      model: ANTHROPIC_DEFAULT_SONNET_MODEL
    entrypoint: ["claude", "--prompt-file", "/workspace/prompt.md"]
    tools: [shell, file-read, file-write]

inference:
  maas:
    endpoint: https://maas.example.com/v1
    provider: maas-anthropic
    default-model: granite-3.3-8b-instruct
    egress: [maas.example.com:443]

mcp:
  github:
    provider: github-pat
    egress: [api.github.com:443]

defaults:
  inference: maas
  policy: restricted
  sandbox:
    scope: session
    mode: all

agents:
  reviewer:
    harness: claude-code
    prompt: "Review code."
    mcp: [github]
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Harnesses["claude-code"].Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("harness image = %q, want ghcr.io/anthropics/claude-code:latest", cfg.Harnesses["claude-code"].Image)
	}
	if cfg.Inference["maas"].Endpoint != "https://maas.example.com/v1" {
		t.Errorf("inference endpoint = %q, want https://maas.example.com/v1", cfg.Inference["maas"].Endpoint)
	}
	if cfg.MCP["github"].Provider != "github-pat" {
		t.Errorf("mcp provider = %q, want github-pat", cfg.MCP["github"].Provider)
	}
	if cfg.Defaults.Inference != "maas" {
		t.Errorf("defaults.inference = %q, want maas", cfg.Defaults.Inference)
	}
	agent, ok := cfg.Agents["reviewer"]
	if !ok {
		t.Fatal("agent 'reviewer' not found")
	}
	if agent.Harness != "claude-code" {
		t.Errorf("agent harness = %q, want claude-code", agent.Harness)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDefaultConfig_HasBuiltInHarnesses(t *testing.T) {
	cfg := DefaultConfig()
	for _, name := range []string{"claude-code", "codex", "goose", "adk"} {
		if _, ok := cfg.Harnesses[name]; !ok {
			t.Errorf("default config missing built-in harness %q", name)
		}
	}
}
