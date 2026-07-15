package compose

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateMCPConfig_Claude(t *testing.T) {
	servers := []ResolvedMCP{
		{Name: "github", Type: "stdio", Command: "github-mcp-server", Env: map[string]string{"GITHUB_TOKEN": "tok123"}},
		{Name: "jira", Type: "http", URL: "https://jira-mcp.internal:8080"},
	}

	data, err := GenerateMCPConfig(servers, "claude")
	if err != nil {
		t.Fatalf("GenerateMCPConfig: %v", err)
	}

	var result struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.MCPServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(result.MCPServers))
	}
	if _, ok := result.MCPServers["github"]; !ok {
		t.Error("missing github server")
	}
	if _, ok := result.MCPServers["jira"]; !ok {
		t.Error("missing jira server")
	}

	var githubEntry struct {
		Command string            `json:"command"`
		Env     map[string]string `json:"env"`
	}
	json.Unmarshal(result.MCPServers["github"], &githubEntry)
	if githubEntry.Command != "github-mcp-server" {
		t.Errorf("github command = %q", githubEntry.Command)
	}
	if githubEntry.Env["GITHUB_TOKEN"] != "tok123" {
		t.Errorf("github env = %v", githubEntry.Env)
	}

	var jiraEntry struct {
		URL string `json:"url"`
	}
	json.Unmarshal(result.MCPServers["jira"], &jiraEntry)
	if jiraEntry.URL != "https://jira-mcp.internal:8080" {
		t.Errorf("jira url = %q", jiraEntry.URL)
	}
}

func TestGenerateMCPConfig_Goose(t *testing.T) {
	servers := []ResolvedMCP{
		{Name: "github", Type: "stdio", Command: "github-mcp-server"},
	}

	data, err := GenerateMCPConfig(servers, "goose")
	if err != nil {
		t.Fatalf("GenerateMCPConfig: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	extensions, ok := result["extensions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected extensions map, got %T", result["extensions"])
	}
	if _, ok := extensions["github"]; !ok {
		t.Error("missing github extension")
	}
}

func TestGenerateMCPConfig_EmptyServers(t *testing.T) {
	data, err := GenerateMCPConfig(nil, "claude")
	if err != nil {
		t.Fatalf("GenerateMCPConfig: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for empty servers, got %d bytes", len(data))
	}
}

func TestGenerateMCPConfig_UnknownFormat(t *testing.T) {
	servers := []ResolvedMCP{{Name: "test", Type: "stdio", Command: "test"}}
	_, err := GenerateMCPConfig(servers, "unknown")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}
