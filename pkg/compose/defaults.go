package compose

func DefaultConfig() *Config {
	return &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Kind:  "harness",
				Image: "ghcr.io/anthropics/claude-code:latest",
				EnvMapping: map[string]string{
					"ANTHROPIC_BASE_URL":             "${endpoint}",
					"ANTHROPIC_API_KEY":              "${key}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
				},
				Entrypoint: []string{"claude", "--prompt-file", "/workspace/prompt.md"},
				Tools:      []string{"shell", "file-read", "file-write", "bundle-mcp"},
			},
			"codex": {
				Kind:  "harness",
				Image: "ghcr.io/openai/codex:latest",
				EnvMapping: map[string]string{
					"OPENAI_BASE_URL": "${endpoint}",
					"OPENAI_API_KEY":  "${key}",
					"OPENAI_MODEL":    "${model}",
				},
				Entrypoint: []string{"codex", "--prompt-file", "/workspace/prompt.md"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
			"goose": {
				Kind:  "harness",
				Image: "ghcr.io/block/goose:latest",
				EnvMapping: map[string]string{
					"OPENAI_BASE_URL": "${endpoint}",
					"OPENAI_API_KEY":  "${key}",
					"GOOSE_MODEL":     "${model}",
				},
				Entrypoint: []string{"goose", "session"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
			"adk": {
				Kind:  "harness",
				Image: "python:3.12-slim",
				EnvMapping: map[string]string{
					"GOOGLE_GENAI_BASE_URL": "${endpoint}",
					"GOOGLE_API_KEY":        "${key}",
					"GOOGLE_GENAI_MODEL":    "${model}",
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
				TTL:   "30m",
			},
		},
	}
}
