package compose

func DefaultConfig() *Config {
	return &Config{
		Harnesses: map[string]HarnessProfile{
			"claude-code": {
				Image: "ghcr.io/anthropics/claude-code:latest",
				EnvMapping: EnvMapping{
					Endpoint: "ANTHROPIC_BASE_URL",
					Key:      "ANTHROPIC_API_KEY",
					Model:    "ANTHROPIC_DEFAULT_SONNET_MODEL",
				},
				Entrypoint: []string{"claude", "--prompt-file", "/workspace/prompt.md"},
				Tools:      []string{"shell", "file-read", "file-write", "bundle-mcp"},
			},
			"codex": {
				Image: "ghcr.io/openai/codex:latest",
				EnvMapping: EnvMapping{
					Endpoint: "OPENAI_BASE_URL",
					Key:      "OPENAI_API_KEY",
					Model:    "OPENAI_MODEL",
				},
				Entrypoint: []string{"codex", "--prompt-file", "/workspace/prompt.md"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
			"goose": {
				Image: "ghcr.io/block/goose:latest",
				EnvMapping: EnvMapping{
					Endpoint: "OPENAI_BASE_URL",
					Key:      "OPENAI_API_KEY",
					Model:    "GOOSE_MODEL",
				},
				Entrypoint: []string{"goose", "session"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
			"adk": {
				Image: "python:3.12-slim",
				EnvMapping: EnvMapping{
					Endpoint: "GOOGLE_GENAI_BASE_URL",
					Key:      "GOOGLE_API_KEY",
					Model:    "GOOGLE_GENAI_MODEL",
				},
				Entrypoint: []string{"python", "-m", "agent"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
		},
		Inference: make(map[string]InferenceSpec),
		MCP:       make(map[string]MCPSpec),
		Agents:    make(map[string]Agent),
		Defaults: Defaults{
			Policy: "restricted",
			Sandbox: SandboxOpts{
				Scope: "session",
				Mode:  "all",
			},
		},
	}
}
