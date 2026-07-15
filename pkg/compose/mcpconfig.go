package compose

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func GenerateMCPConfig(servers []ResolvedMCP, format string) ([]byte, error) {
	if len(servers) == 0 {
		return nil, nil
	}
	switch format {
	case "claude", "codex":
		return generateClaudeConfig(servers)
	case "goose":
		return generateGooseConfig(servers)
	default:
		return nil, fmt.Errorf("unsupported MCP config format: %q", format)
	}
}

func generateClaudeConfig(servers []ResolvedMCP) ([]byte, error) {
	type stdioServer struct {
		Command string            `json:"command"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
	}
	type httpServer struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	mcpServers := make(map[string]interface{}, len(servers))
	for _, s := range servers {
		switch s.Type {
		case "http":
			mcpServers[s.Name] = httpServer{Type: "http", URL: s.URL}
		default:
			mcpServers[s.Name] = stdioServer{
				Command: s.Command,
				Args:    s.Args,
				Env:     s.Env,
			}
		}
	}
	return json.MarshalIndent(map[string]interface{}{"mcpServers": mcpServers}, "", "  ")
}

func generateGooseConfig(servers []ResolvedMCP) ([]byte, error) {
	extensions := make(map[string]interface{}, len(servers))
	for _, s := range servers {
		ext := map[string]interface{}{"type": s.Type}
		if s.Command != "" {
			ext["command"] = s.Command
		}
		if len(s.Args) > 0 {
			ext["args"] = s.Args
		}
		if s.URL != "" {
			ext["url"] = s.URL
		}
		if len(s.Env) > 0 {
			ext["env"] = s.Env
		}
		extensions[s.Name] = ext
	}
	return yaml.Marshal(map[string]interface{}{"extensions": extensions})
}
