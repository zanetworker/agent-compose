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
runtimes:
  claude-code:
    kind: harness
    image: ghcr.io/anthropics/claude-code:latest
    env-mapping:
      ANTHROPIC_BASE_URL: "${endpoint}"
      ANTHROPIC_API_KEY: "${key}"
      ANTHROPIC_DEFAULT_SONNET_MODEL: "${model}"
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
    runtime: claude-code
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

	if cfg.Runtimes["claude-code"].Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("runtime image = %q, want ghcr.io/anthropics/claude-code:latest", cfg.Runtimes["claude-code"].Image)
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
	if agent.Runtime != "claude-code" {
		t.Errorf("agent runtime = %q, want claude-code", agent.Runtime)
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
		rt, ok := cfg.Runtimes[name]
		if !ok {
			t.Errorf("default config missing built-in runtime %q", name)
		}
		if rt.Kind != "harness" {
			t.Errorf("runtime %q has kind=%q, want harness", name, rt.Kind)
		}
	}
}

func TestLoadConfig_RuntimesField(t *testing.T) {
	yaml := `
runtimes:
  claude-code:
    kind: harness
    image: ghcr.io/anthropics/claude-code:latest
    env-mapping:
      ANTHROPIC_BASE_URL: "${endpoint}"
      ANTHROPIC_API_KEY: "${key}"
      ANTHROPIC_DEFAULT_SONNET_MODEL: "${model}"
    entrypoint: ["claude", "--prompt-file", "/workspace/prompt.md"]
    tools: [shell, file-read]
agents:
  reviewer:
    runtime: claude-code
    prompt: "Review this code"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rt, ok := cfg.Runtimes["claude-code"]
	if !ok {
		t.Fatal("expected runtimes to contain claude-code")
	}
	if rt.Kind != "harness" {
		t.Errorf("expected kind=harness, got %q", rt.Kind)
	}
	if rt.EnvMapping["ANTHROPIC_BASE_URL"] != "${endpoint}" {
		t.Errorf("expected N-var env-mapping, got %v", rt.EnvMapping)
	}
	agent := cfg.Agents["reviewer"]
	if agent.Runtime != "claude-code" {
		t.Errorf("expected runtime=claude-code, got %q", agent.Runtime)
	}
}
