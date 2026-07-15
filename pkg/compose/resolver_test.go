package compose

import (
	"context"
	"fmt"
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

	if spec.Image != "ghcr.io/nvidia/openshell-community/sandboxes/base:latest" {
		t.Errorf("image = %q", spec.Image)
	}
	if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.example.com/v1" {
		t.Errorf("ANTHROPIC_BASE_URL = %q", spec.Env["ANTHROPIC_BASE_URL"])
	}
	if spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "granite-3.3-8b" {
		t.Errorf("ANTHROPIC_DEFAULT_SONNET_MODEL = %q", spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}

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

func TestResolver_MCPConfigGeneration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP["github"] = MCPSpec{
		Type:     "stdio",
		Command:  "github-mcp-server",
		Env:      map[string]string{"GITHUB_TOKEN": "test-token"},
		Provider: "github-pat",
		Egress:   []string{"api.github.com:443"},
	}
	cfg.Runtimes["claude-code"] = RuntimeProfile{
		Kind:       "harness",
		Image:      "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
		EnvMapping: map[string]string{},
		Entrypoint: []string{"claude"},
		MCPConfig:  MCPConfig{Format: "claude", Path: "/sandbox/.claude.json"},
	}

	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{Name: "reviewer", Runtime: "claude-code", MCP: []string{"github"}}
	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(spec.MCPServers) != 1 {
		t.Fatalf("MCPServers len = %d, want 1", len(spec.MCPServers))
	}
	if spec.MCPServers[0].Name != "github" {
		t.Errorf("MCPServers[0].Name = %q", spec.MCPServers[0].Name)
	}
	if spec.MCPServers[0].Command != "github-mcp-server" {
		t.Errorf("MCPServers[0].Command = %q", spec.MCPServers[0].Command)
	}

	found := false
	for _, m := range spec.SkillMounts {
		if m.Target == "/sandbox/.claude.json" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MCP config mount at /sandbox/.claude.json, mounts: %v", spec.SkillMounts)
	}
}

func TestResolver_NoMCPConfig_WhenNoServers(t *testing.T) {
	cfg := DefaultConfig()
	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{Name: "reviewer", Runtime: "claude-code"}
	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(spec.MCPServers) != 0 {
		t.Errorf("expected no MCPServers, got %d", len(spec.MCPServers))
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
	if spec.Env["OPENAI_BASE_URL"] != "https://maas.example.com/v1" {
		t.Errorf("OPENAI_BASE_URL = %q, want https://maas.example.com/v1", spec.Env["OPENAI_BASE_URL"])
	}
	if spec.Env["MODEL_NAME"] != "granite-3.3-8b" {
		t.Errorf("MODEL_NAME = %q", spec.Env["MODEL_NAME"])
	}
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

	if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.example.com/v1" {
		t.Errorf("default inference not applied: ANTHROPIC_BASE_URL = %q", spec.Env["ANTHROPIC_BASE_URL"])
	}
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

// staticRuntimeResolver implements RuntimeResolver for tests
type staticRuntimeResolver struct {
	profiles map[string]RuntimeProfile
}

func (s *staticRuntimeResolver) Resolve(_ context.Context, name string) (*RuntimeProfile, error) {
	profile, ok := s.profiles[name]
	if !ok {
		return nil, fmt.Errorf("runtime profile %q not found", name)
	}
	return &profile, nil
}

func (s *staticRuntimeResolver) List(_ context.Context) ([]RuntimeProfile, error) {
	profiles := make([]RuntimeProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// staticInferenceResolver implements InferenceResolver for tests
type staticInferenceResolver struct {
	specs map[string]InferenceSpec
}

func (s *staticInferenceResolver) Resolve(_ context.Context, name string) (*InferenceSpec, error) {
	spec, ok := s.specs[name]
	if !ok {
		return nil, fmt.Errorf("inference spec %q not found", name)
	}
	return &spec, nil
}

func (s *staticInferenceResolver) List(_ context.Context) ([]InferenceSpec, error) {
	specs := make([]InferenceSpec, 0, len(s.specs))
	for _, s := range s.specs {
		specs = append(specs, s)
	}
	return specs, nil
}

// noopMCPResolver implements MCPResolver for tests
type noopMCPResolver struct{}

func (n *noopMCPResolver) Resolve(_ context.Context, _ string) (*MCPSpec, error) {
	return &MCPSpec{}, nil
}

func (n *noopMCPResolver) List(_ context.Context) ([]MCPSpec, error) {
	return nil, nil
}

// noopSkillResolver implements SkillResolver for tests
type noopSkillResolver struct{}

func (n *noopSkillResolver) Resolve(_ context.Context, _ string) (*Skill, error) {
	return &Skill{}, nil
}

func (n *noopSkillResolver) List(_ context.Context) ([]Skill, error) {
	return nil, nil
}

// noopPolicyResolver implements PolicyResolver for tests
type noopPolicyResolver struct{}

func (n *noopPolicyResolver) Resolve(_ context.Context, _ string) (*Policy, error) {
	return &Policy{}, nil
}

func (n *noopPolicyResolver) List(_ context.Context) ([]Policy, error) {
	return nil, nil
}

func TestResolver_NVarEnvMapping(t *testing.T) {
	runtimes := &staticRuntimeResolver{
		profiles: map[string]RuntimeProfile{
			"claude-code": {
				Kind:  "harness",
				Image: "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
				EnvMapping: map[string]string{
					"ANTHROPIC_BASE_URL":             "${endpoint}",
					"ANTHROPIC_API_KEY":              "${key}",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
					"ANTHROPIC_DEFAULT_OPUS_MODEL":   "${model.opus}",
					"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "${model.haiku}",
				},
				Entrypoint: []string{"claude"},
				Tools:      []string{"shell"},
			},
		},
	}
	inference := &staticInferenceResolver{
		specs: map[string]InferenceSpec{
			"maas": {
				Endpoint:     "https://maas.example.com/v1",
				Provider:     "maas-anthropic",
				DefaultModel: "granite-3.3-8b",
				Models: map[string]string{
					"opus":  "granite-3.3-8b",
					"haiku": "granite-3.3-2b",
				},
				Egress: []string{"maas.example.com:443"},
			},
		},
	}

	r := NewResolver(runtimes, inference, &noopMCPResolver{}, &noopSkillResolver{}, &noopPolicyResolver{}, Defaults{Inference: "maas", Sandbox: SandboxOpts{Scope: "session", Mode: "all", TTL: "30m"}})
	spec, err := r.Resolve(nil, Agent{Name: "test", Runtime: "claude-code"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if spec.Env["ANTHROPIC_BASE_URL"] != "https://maas.example.com/v1" {
		t.Errorf("endpoint: got %q", spec.Env["ANTHROPIC_BASE_URL"])
	}
	if spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "granite-3.3-8b" {
		t.Errorf("model: got %q", spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if spec.Env["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "granite-3.3-8b" {
		t.Errorf("opus tier: got %q", spec.Env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if spec.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "granite-3.3-2b" {
		t.Errorf("haiku tier: got %q", spec.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if spec.RuntimeKind != "harness" {
		t.Errorf("runtime kind: got %q", spec.RuntimeKind)
	}
	if spec.Sandbox.TTL != "30m" {
		t.Errorf("sandbox TTL: got %q", spec.Sandbox.TTL)
	}
}
