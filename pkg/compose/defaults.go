package compose

func DefaultConfig() *Config {
	return &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Kind:  "harness",
				Image: "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
				EnvMapping: map[string]string{
					"ANTHROPIC_BASE_URL":             "${endpoint}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
				},
				Entrypoint: []string{"claude"},
				Tools:      []string{"shell", "file-read", "file-write", "bundle-mcp"},
				Providers:  []string{"claude-code"},
			},
			"claude-code-vertex": {
				Kind:  "harness",
				Image: "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
				EnvMapping: map[string]string{
					"CLAUDE_CODE_USE_VERTEX":         "1",
					"CLOUD_ML_REGION":                "${region}",
					"ANTHROPIC_VERTEX_PROJECT_ID":    "${project}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
				},
				Entrypoint: []string{"claude"},
				Tools:      []string{"shell", "file-read", "file-write", "bundle-mcp"},
				Providers:  []string{"vertex", "gcp"},
			},
			"codex": {
				Kind:  "harness",
				Image: "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
				EnvMapping: map[string]string{
					"OPENAI_BASE_URL": "${endpoint}",
					"OPENAI_MODEL":    "${model}",
				},
				Entrypoint: []string{"codex"},
				Tools:      []string{"shell", "file-read", "file-write"},
				Providers:  []string{"codex"},
			},
			"goose": {
				Kind:  "harness",
				Image: "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
				EnvMapping: map[string]string{
					"OPENAI_BASE_URL": "${endpoint}",
					"GOOSE_MODEL":     "${model}",
				},
				Entrypoint: []string{"goose", "session"},
				Tools:      []string{"shell", "file-read", "file-write"},
			},
		},
		Inference: make(map[string]InferenceSpec),
		MCP:       make(map[string]MCPSpec),
		Agents:    make(map[string]Agent),
		Defaults: Defaults{
			Sandbox: SandboxOpts{
				Scope: "session",
				Mode:  "all",
				TTL:   "30m",
			},
		},
	}
}
