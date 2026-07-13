package examples_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/zanetworker/agent-compose/pkg/compose"
)

func TestSDK_ResolveAgent(t *testing.T) {
	cfg := &compose.Config{
		Runtimes: map[string]compose.RuntimeProfile{
			"claude-code": {
				Kind:  "harness",
				Image: "ghcr.io/anthropics/claude-code:latest",
				EnvMapping: map[string]string{
					"ANTHROPIC_BASE_URL":             "${endpoint}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
				},
				Entrypoint: []string{"claude", "--prompt-file", "/workspace/prompt.md"},
				Tools:      []string{"shell", "file-read", "file-write"},
				Providers:  []string{"claude-code"},
			},
		},
		Inference: map[string]compose.InferenceSpec{
			"maas": {
				Endpoint:     "https://maas.apps.cluster.com/v1",
				Provider:     "maas-anthropic",
				DefaultModel: "granite-3.3-8b-instruct",
				Models: map[string]string{
					"opus":  "granite-3.3-8b-instruct",
					"haiku": "granite-3.3-2b-instruct",
				},
				Egress: []string{"maas.apps.cluster.com:443"},
			},
		},
		MCP: map[string]compose.MCPSpec{
			"github": {
				Provider: "github-pat",
				Egress:   []string{"api.github.com:443"},
			},
		},
		Agents: map[string]compose.Agent{
			"code-reviewer": {
				Runtime: "claude-code",
				MCP:     []string{"github"},
				Prompt:  "Review code for security issues.",
			},
		},
		Defaults: compose.Defaults{
			Inference: "maas",
			Sandbox: compose.SandboxOpts{
				Scope: "session",
				Mode:  "all",
				TTL:   "30m",
			},
		},
	}

	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(os.Stdout)),
	)

	spec, err := engine.Resolve(context.Background(), "code-reviewer")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if spec.RuntimeKind != "harness" {
		t.Errorf("RuntimeKind: got %q, want harness", spec.RuntimeKind)
	}
	if spec.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("Image: got %q", spec.Image)
	}
	if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.apps.cluster.com/v1" {
		t.Errorf("ANTHROPIC_BASE_URL: got %q", spec.Env["ANTHROPIC_BASE_URL"])
	}
	if spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "granite-3.3-8b-instruct" {
		t.Errorf("model: got %q", spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if len(spec.Providers) != 3 {
		t.Errorf("Providers: got %v, want [claude-code, maas-anthropic, github-pat]", spec.Providers)
	}
	if spec.Sandbox.TTL != "30m" {
		t.Errorf("Sandbox.TTL: got %q", spec.Sandbox.TTL)
	}
	if spec.Prompt != "Review code for security issues." {
		t.Errorf("Prompt: got %q", spec.Prompt)
	}

	data, _ := json.MarshalIndent(spec, "", "  ")
	fmt.Println(string(data))
}

func TestSDK_ResolveWithOverrides(t *testing.T) {
	cfg := &compose.Config{
		Runtimes: map[string]compose.RuntimeProfile{
			"claude-code": {
				Kind:  "harness",
				Image: "ghcr.io/anthropics/claude-code:latest",
				EnvMapping: map[string]string{
					"ANTHROPIC_BASE_URL":             "${endpoint}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
				},
				Entrypoint: []string{"claude"},
				Providers:  []string{"claude-code"},
			},
		},
		Inference: map[string]compose.InferenceSpec{
			"maas": {
				Endpoint:     "https://maas.apps.cluster.com/v1",
				Provider:     "maas-anthropic",
				DefaultModel: "granite-3.3-8b-instruct",
			},
			"local-vllm": {
				Endpoint:     "https://vllm.internal:8000/v1",
				Provider:     "vllm-local",
				DefaultModel: "llama-3.3-70b",
			},
		},
		MCP:    map[string]compose.MCPSpec{},
		Agents: map[string]compose.Agent{
			"reviewer": {
				Runtime: "claude-code",
				Prompt:  "Review code",
			},
		},
		Defaults: compose.Defaults{
			Inference: "maas",
			Sandbox:   compose.SandboxOpts{Scope: "session"},
		},
	}

	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(os.Stdout)),
	)

	// Override inference and model
	run, err := engine.Run(context.Background(), "reviewer", compose.RunOpts{
		Inference: "local-vllm",
		Model:     "custom-7b",
		Prompt:    "Focus on auth bypass",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Agent != "reviewer" {
		t.Errorf("Agent: got %q", run.Agent)
	}
}

func TestSDK_InlineAgent(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.Inference["gpu"] = compose.InferenceSpec{
		Endpoint:     "https://qwen3-14b.apps.cluster.dev/v1",
		Provider:     "gpu-vllm",
		DefaultModel: "qwen3-14b",
	}

	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(os.Stdout)),
	)

	// No config.yaml agent needed; compose inline
	_, err := engine.Run(context.Background(), "", compose.RunOpts{
		Agent: &compose.Agent{
			Runtime:   "claude-code",
			MCP:       []string{},
			Prompt:    "Hello from the SDK",
			Workspace: "/tmp/test",
		},
		Inference: "gpu",
		Model:     "qwen3-14b",
	})
	if err != nil {
		t.Fatalf("Run inline: %v", err)
	}
}
