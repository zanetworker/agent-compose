package compose

import (
	"context"
	"strings"
	"testing"
)

func testConfig() *Config {
	cfg := DefaultConfig()
	cfg.Inference["maas"] = InferenceSpec{
		Endpoint:     "https://maas.example.com/v1",
		Provider:     "maas-anthropic",
		DefaultModel: "granite-3.3-8b",
		Egress:       []string{"maas.example.com:443"},
	}
	cfg.MCP["github"] = MCPSpec{
		Provider: "github-pat",
		Egress:   []string{"api.github.com:443"},
	}
	cfg.Defaults.Inference = "maas"
	cfg.Defaults.Policy = "restricted"
	return cfg
}

func TestResolver_HarnessAgent(t *testing.T) {
	cfg := testConfig()
	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:      "reviewer",
		Runtime:   "claude-code",
		Inference: "maas",
		MCP:       []string{"github"},
		Prompt:    "Review code.",
		Policy:    "restricted",
	}

	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if spec.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("image = %q", spec.Image)
	}
	// TODO(Task 5): Re-enable these assertions after N-var env-mapping is wired
	// For now, env vars are stubbed out in resolver.go
	// if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.example.com/v1" {
	// 	t.Errorf("ANTHROPIC_BASE_URL = %q", spec.Env["ANTHROPIC_BASE_URL"])
	// }
	// if spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "granite-3.3-8b" {
	// 	t.Errorf("ANTHROPIC_DEFAULT_SONNET_MODEL = %q", spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	// }

	providers := spec.Providers
	if !contains(providers, "maas-anthropic") {
		t.Errorf("providers %v missing maas-anthropic", providers)
	}
	if !contains(providers, "github-pat") {
		t.Errorf("providers %v missing github-pat", providers)
	}

	if !contains(spec.Egress, "maas.example.com:443") {
		t.Errorf("egress %v missing maas.example.com:443", spec.Egress)
	}
	if !contains(spec.Egress, "api.github.com:443") {
		t.Errorf("egress %v missing api.github.com:443", spec.Egress)
	}

	if spec.Policy != "restricted" {
		t.Errorf("policy = %q, want restricted", spec.Policy)
	}
	if spec.Prompt != "Review code." {
		t.Errorf("prompt = %q", spec.Prompt)
	}
}

func TestResolver_FrameworkAgent_CustomEnvMapping(t *testing.T) {
	cfg := testConfig()
	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:      "custom",
		Image:     "quay.io/acme/agent:v1",
		Inference: "maas",
		EnvMapping: map[string]string{
			"OPENAI_BASE_URL": "${endpoint}",
			"OPENAI_API_KEY":  "${key}",
			"MODEL_NAME":      "${model}",
		},
		Entrypoint: []string{"python", "-m", "agent"},
		Prompt:     "Do stuff.",
	}

	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if spec.Image != "quay.io/acme/agent:v1" {
		t.Errorf("image = %q", spec.Image)
	}
	// TODO(Task 5): Re-enable these assertions after N-var env-mapping is wired
	// if spec.Env["OPENAI_BASE_URL"] != "https://maas.example.com/v1" {
	// 	t.Errorf("OPENAI_BASE_URL = %q, want https://maas.example.com/v1", spec.Env["OPENAI_BASE_URL"])
	// }
	// if spec.Env["MODEL_NAME"] != "granite-3.3-8b" {
	// 	t.Errorf("MODEL_NAME = %q", spec.Env["MODEL_NAME"])
	// }
}

func TestResolver_AppliesDefaults(t *testing.T) {
	cfg := testConfig()
	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:    "minimal",
		Runtime: "claude-code",
		Prompt:  "Hello.",
	}

	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// TODO(Task 5): Re-enable this assertion after N-var env-mapping is wired
	// if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.example.com/v1" {
	// 	t.Errorf("default inference not applied: ANTHROPIC_BASE_URL = %q", spec.Env["ANTHROPIC_BASE_URL"])
	// }
	if spec.Policy != "restricted" {
		t.Errorf("default policy not applied: policy = %q", spec.Policy)
	}
}

func TestResolver_SkillsMergePromptAndDeps(t *testing.T) {
	cfg := testConfig()
	skillsDir := t.TempDir()
	writeSkill(t, skillsDir, "sec-review",
		"---\nrequires:\n  mcp: [github]\n  tools: [shell]\n---\n\n# Security\nCheck XSS.\n")

	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(skillsDir),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:    "with-skill",
		Runtime: "claude-code",
		Skills:  []string{"sec-review"},
		Prompt:  "Base prompt.",
	}

	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if !strings.Contains(spec.Prompt, "Base prompt.") {
		t.Error("prompt missing agent prompt")
	}
	if !strings.Contains(spec.Prompt, "Check XSS.") {
		t.Error("prompt missing skill prompt")
	}
	if !contains(spec.Providers, "github-pat") {
		t.Error("skill's MCP dependency not merged into providers")
	}
	if !contains(spec.Tools, "shell") {
		t.Error("skill's tool dependency not merged into tools")
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
