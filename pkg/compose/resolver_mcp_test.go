package compose

import (
	"context"
	"errors"
	"testing"
)

func TestConfigMCPResolver_Resolve(t *testing.T) {
	cfg := &Config{
		MCP: map[string]MCPSpec{
			"github": {
				Provider: "github-pat",
				Egress:   []string{"api.github.com:443", "github.com:443"},
			},
		},
	}
	r := NewConfigMCPResolver(cfg)

	spec, err := r.Resolve(context.Background(), "github")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if spec.Provider != "github-pat" {
		t.Errorf("provider = %q, want github-pat", spec.Provider)
	}
	if len(spec.Egress) != 2 {
		t.Errorf("egress len = %d, want 2", len(spec.Egress))
	}
	if spec.Name != "github" {
		t.Errorf("name = %q, want github", spec.Name)
	}
}

func TestConfigMCPResolver_NotFound(t *testing.T) {
	cfg := &Config{MCP: map[string]MCPSpec{}}
	r := NewConfigMCPResolver(cfg)

	_, err := r.Resolve(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestConfigMCPResolver_Resolve_WithConnectionDetails(t *testing.T) {
	cfg := &Config{
		MCP: map[string]MCPSpec{
			"github": {
				Type:     "stdio",
				Command:  "github-mcp-server",
				Env:      map[string]string{"GITHUB_TOKEN": "tok"},
				Provider: "github-pat",
				Egress:   []string{"api.github.com:443"},
			},
			"jira": {
				Type:     "http",
				URL:      "https://jira-mcp.internal:8080",
				Provider: "jira-oauth",
				Egress:   []string{"jira-mcp.internal:8080"},
			},
		},
	}
	r := NewConfigMCPResolver(cfg)

	github, err := r.Resolve(context.Background(), "github")
	if err != nil {
		t.Fatalf("Resolve github: %v", err)
	}
	if github.Type != "stdio" {
		t.Errorf("type = %q, want stdio", github.Type)
	}
	if github.Command != "github-mcp-server" {
		t.Errorf("command = %q", github.Command)
	}

	jira, err := r.Resolve(context.Background(), "jira")
	if err != nil {
		t.Fatalf("Resolve jira: %v", err)
	}
	if jira.Type != "http" {
		t.Errorf("type = %q, want http", jira.Type)
	}
	if jira.URL != "https://jira-mcp.internal:8080" {
		t.Errorf("url = %q", jira.URL)
	}
}

func TestConfigMCPResolver_List(t *testing.T) {
	cfg := &Config{
		MCP: map[string]MCPSpec{
			"github": {
				Provider: "github-pat",
			},
			"slack": {
				Provider: "slack-oauth",
			},
		},
	}
	r := NewConfigMCPResolver(cfg)

	specs, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(specs) != 2 {
		t.Errorf("len = %d, want 2", len(specs))
	}
}
