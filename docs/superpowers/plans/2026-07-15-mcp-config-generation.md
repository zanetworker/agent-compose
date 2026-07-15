# MCP Config Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an agent references MCP servers, agent-compose generates the agent-specific config file (Claude's `settings.json`, Goose's `config.yaml`, etc.) and uploads it to the sandbox so the agent discovers its MCP servers without manual configuration.

**Architecture:** Expand `MCPSpec` with connection details (type, command, url, env). Add an `MCPConfig` field to `RuntimeProfile` that declares the target format and path. A new `mcpconfig` package generates the agent-native config file from resolved MCP specs. The resolver calls the generator and adds the output as a `SkillMount` for upload.

**Tech Stack:** Go, YAML/JSON marshaling (stdlib `encoding/json`, `gopkg.in/yaml.v3`)

## Global Constraints

- Follow existing patterns in `pkg/compose/` (resolver interface, test patterns, config loading)
- No new dependencies beyond stdlib + existing yaml.v3
- Backward compatible: existing configs without the new fields continue to work
- Test-first: failing test before implementation

---

### Task 1: Expand MCPSpec with connection details

**Files:**
- Modify: `pkg/compose/types.go:23-27` (MCPSpec struct)
- Modify: `pkg/compose/types.go:69-85` (ResolvedSpec struct)
- Test: `pkg/compose/resolver_mcp_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: `MCPSpec{Name, Type, Command, Args, URL, Env, Provider, Egress}`, `ResolvedMCP{Name, Type, Command, Args, URL, Env}`

- [ ] **Step 1: Write failing test for new MCPSpec fields**

Add to `pkg/compose/resolver_mcp_test.go`:

```go
func TestConfigMCPResolver_Resolve_WithConnectionDetails(t *testing.T) {
	cfg := &Config{
		MCP: map[string]MCPSpec{
			"github": {
				Type:     "stdio",
				Command:  "github-mcp-server",
				Env:      map[string]string{"GITHUB_TOKEN": "${provider:github}"},
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
		t.Errorf("command = %q, want github-mcp-server", github.Command)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/compose/ -run TestConfigMCPResolver_Resolve_WithConnectionDetails -v`
Expected: FAIL (fields don't exist on MCPSpec)

- [ ] **Step 3: Update MCPSpec and add ResolvedMCP**

In `pkg/compose/types.go`, replace the MCPSpec struct and add ResolvedMCP:

```go
type MCPSpec struct {
	Name     string            `yaml:"name,omitempty"`
	Type     string            `yaml:"type,omitempty"`    // stdio | http
	Command  string            `yaml:"command,omitempty"` // binary for stdio servers
	Args     []string          `yaml:"args,omitempty"`    // args for stdio command
	URL      string            `yaml:"url,omitempty"`     // endpoint for http servers
	Env      map[string]string `yaml:"env,omitempty"`     // env vars for the MCP server process
	Provider string            `yaml:"provider"`
	Egress   []string          `yaml:"egress"`
}

type ResolvedMCP struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}
```

Add `MCPServers []ResolvedMCP` to `ResolvedSpec`:

```go
type ResolvedSpec struct {
	// ... existing fields ...
	MCPServers  []ResolvedMCP `json:"mcp_servers,omitempty"`
	// ... rest of fields ...
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/compose/ -run TestConfigMCPResolver_Resolve_WithConnectionDetails -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/compose/types.go pkg/compose/resolver_mcp_test.go
git commit -m "feat: expand MCPSpec with connection details (type, command, url, env)"
```

---

### Task 2: Add MCPConfig to RuntimeProfile

**Files:**
- Modify: `pkg/compose/types.go:3-12` (RuntimeProfile struct)
- Modify: `pkg/compose/defaults.go` (add MCPConfig to default runtimes)
- Test: `pkg/compose/config_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: `RuntimeProfile.MCPConfig{Format, Path}` where Format is `"claude"`, `"goose"`, `"codex"`, or `""` (no generation)

- [ ] **Step 1: Write failing test**

Add to `pkg/compose/config_test.go`:

```go
func TestConfig_RuntimeProfile_MCPConfig(t *testing.T) {
	cfg := DefaultConfig()
	claude := cfg.Runtimes["claude-code"]
	if claude.MCPConfig.Format != "claude" {
		t.Errorf("claude-code MCPConfig.Format = %q, want claude", claude.MCPConfig.Format)
	}
	if claude.MCPConfig.Path != "/sandbox/.claude.json" {
		t.Errorf("claude-code MCPConfig.Path = %q", claude.MCPConfig.Path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/compose/ -run TestConfig_RuntimeProfile_MCPConfig -v`
Expected: FAIL (MCPConfig field doesn't exist)

- [ ] **Step 3: Add MCPConfig struct and field**

In `pkg/compose/types.go`, add the struct and field to RuntimeProfile:

```go
type MCPConfig struct {
	Format string `yaml:"format,omitempty"` // claude | codex | goose | ""
	Path   string `yaml:"path,omitempty"`   // target path inside sandbox
}

type RuntimeProfile struct {
	// ... existing fields ...
	MCPConfig  MCPConfig `yaml:"mcp-config,omitempty"`
}
```

In `pkg/compose/defaults.go`, add MCPConfig to each runtime:

```go
"claude-code": {
	// ... existing fields ...
	MCPConfig: MCPConfig{Format: "claude", Path: "/sandbox/.claude.json"},
},
"claude-code-vertex": {
	// ... existing fields ...
	MCPConfig: MCPConfig{Format: "claude", Path: "/sandbox/.claude.json"},
},
"codex": {
	// ... existing fields ...
	MCPConfig: MCPConfig{Format: "codex", Path: "/sandbox/.codex.json"},
},
"goose": {
	// ... existing fields ...
	MCPConfig: MCPConfig{Format: "goose", Path: "/sandbox/.config/goose/config.yaml"},
},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/compose/ -run TestConfig_RuntimeProfile_MCPConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/compose/types.go pkg/compose/defaults.go pkg/compose/config_test.go
git commit -m "feat: add MCPConfig (format, path) to RuntimeProfile"
```

---

### Task 3: MCP config generators

**Files:**
- Create: `pkg/compose/mcpconfig.go`
- Create: `pkg/compose/mcpconfig_test.go`

**Interfaces:**
- Consumes: `[]ResolvedMCP`, `MCPConfig{Format, Path}`
- Produces: `GenerateMCPConfig(servers []ResolvedMCP, format string) ([]byte, error)`

- [ ] **Step 1: Write failing tests for each format**

Create `pkg/compose/mcpconfig_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/compose/ -run TestGenerateMCPConfig -v`
Expected: FAIL (function doesn't exist)

- [ ] **Step 3: Implement GenerateMCPConfig**

Create `pkg/compose/mcpconfig.go`:

```go
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
		URL string `json:"url"`
	}

	mcpServers := make(map[string]interface{}, len(servers))
	for _, s := range servers {
		switch s.Type {
		case "http":
			mcpServers[s.Name] = httpServer{URL: s.URL}
		default:
			mcpServers[s.Name] = stdioServer{
				Command: s.Command,
				Args:    s.Args,
				Env:     s.Env,
			}
		}
	}

	return json.MarshalIndent(map[string]interface{}{
		"mcpServers": mcpServers,
	}, "", "  ")
}

func generateGooseConfig(servers []ResolvedMCP) ([]byte, error) {
	extensions := make(map[string]interface{}, len(servers))
	for _, s := range servers {
		ext := map[string]interface{}{
			"type": s.Type,
		}
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

	return yaml.Marshal(map[string]interface{}{
		"extensions": extensions,
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/compose/ -run TestGenerateMCPConfig -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/compose/mcpconfig.go pkg/compose/mcpconfig_test.go
git commit -m "feat: MCP config generators for Claude, Codex, and Goose formats"
```

---

### Task 4: Wire MCP config generation into the resolver

**Files:**
- Modify: `pkg/compose/resolver.go:107-141` (MCP resolution loop)
- Test: `pkg/compose/resolver_test.go`

**Interfaces:**
- Consumes: `MCPSpec{Type, Command, Args, URL, Env}`, `RuntimeProfile.MCPConfig{Format, Path}`, `GenerateMCPConfig()`
- Produces: `ResolvedSpec.MCPServers` populated, `ResolvedSpec.SkillMounts` includes generated config file

- [ ] **Step 1: Write failing test**

Add to `pkg/compose/resolver_test.go`:

```go
func TestResolver_MCPConfigGeneration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP["github"] = MCPSpec{
		Type:     "stdio",
		Command:  "github-mcp-server",
		Env:      map[string]string{"GITHUB_TOKEN": "test-token"},
		Provider: "github-pat",
		Egress:   []string{"api.github.com:443"},
	}

	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		MCP:     []string{"github"},
		Prompt:  "Review.",
	}

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
}

func TestResolver_MCPConfigMount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP["github"] = MCPSpec{
		Type:    "stdio",
		Command: "github-mcp-server",
	}

	r := NewResolver(
		NewConfigRuntimeResolver(cfg),
		NewConfigInferenceResolver(cfg),
		NewConfigMCPResolver(cfg),
		NewLocalSkillResolver(t.TempDir()),
		NewConfigPolicyResolver(),
		cfg.Defaults,
	)

	agent := Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		MCP:     []string{"github"},
	}

	spec, err := r.Resolve(context.Background(), agent)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	found := false
	for _, m := range spec.SkillMounts {
		if m.Target == "/sandbox/.claude.json" {
			found = true
			break
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/compose/ -run TestResolver_MCPConfig -v`
Expected: FAIL (MCPServers not populated)

- [ ] **Step 3: Update resolver to populate MCPServers and generate config**

In `pkg/compose/resolver.go`, modify the MCP resolution loop (around line 132). After resolving each MCP spec for providers/egress, also collect it into `ResolvedMCP`. After the loop, generate the config file and add it as a mount.

Replace the MCP resolution section:

```go
	// Resolve each MCP
	for _, mcpName := range mcpNames {
		mcpSpec, err := r.mcp.Resolve(ctx, mcpName)
		if err != nil {
			return nil, fmt.Errorf("resolving mcp %q: %w", mcpName, err)
		}
		if mcpSpec.Provider != "" {
			spec.Providers = appendUnique(spec.Providers, mcpSpec.Provider)
		}
		spec.Egress = appendUnique(spec.Egress, mcpSpec.Egress...)

		if mcpSpec.Type != "" {
			spec.MCPServers = append(spec.MCPServers, ResolvedMCP{
				Name:    mcpSpec.Name,
				Type:    mcpSpec.Type,
				Command: mcpSpec.Command,
				Args:    mcpSpec.Args,
				URL:     mcpSpec.URL,
				Env:     mcpSpec.Env,
			})
		}
	}

	// Generate MCP config file for the runtime
	var mcpConfigFormat string
	var mcpConfigPath string
	if agent.Runtime != "" {
		if profile, err := r.runtimes.Resolve(ctx, agent.Runtime); err == nil {
			mcpConfigFormat = profile.MCPConfig.Format
			mcpConfigPath = profile.MCPConfig.Path
		}
	}
	if len(spec.MCPServers) > 0 && mcpConfigFormat != "" {
		configData, err := GenerateMCPConfig(spec.MCPServers, mcpConfigFormat)
		if err != nil {
			return nil, fmt.Errorf("generating MCP config: %w", err)
		}
		if configData != nil {
			tmpfile, err := os.CreateTemp("", "ac-mcp-config-*")
			if err != nil {
				return nil, fmt.Errorf("creating MCP config temp file: %w", err)
			}
			if _, err := tmpfile.Write(configData); err != nil {
				tmpfile.Close()
				os.Remove(tmpfile.Name())
				return nil, fmt.Errorf("writing MCP config: %w", err)
			}
			tmpfile.Close()
			spec.SkillMounts = append(spec.SkillMounts, Mount{
				Source: tmpfile.Name(),
				Target: mcpConfigPath,
			})
		}
	}
```

Add `"os"` to the import list in `resolver.go`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/compose/ -run TestResolver_MCPConfig -v`
Expected: all PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/compose/resolver.go pkg/compose/resolver_test.go
git commit -m "feat: wire MCP config generation into resolver, mount as config file"
```

---

### Task 5: Update config.yaml and docs

**Files:**
- Modify: `~/.ac/config.yaml` (add Type/Command/URL/Env to MCP entries, add MCPConfig to runtimes)
- Modify: `docs/status-and-next.md`

**Interfaces:**
- Consumes: all previous tasks
- Produces: working end-to-end flow

- [ ] **Step 1: Update config.yaml MCP section**

```yaml
mcp:
  github:
    type: stdio
    command: github-mcp-server
    env:
      GITHUB_TOKEN: "${provider:github}"
    provider: github
    egress:
      - api.github.com:443
      - github.com:443
```

- [ ] **Step 2: Update config.yaml runtimes with MCPConfig**

Add to each runtime profile:

```yaml
  claude-code-vertex:
    # ... existing fields ...
    mcp-config:
      format: claude
      path: /sandbox/.claude.json
```

- [ ] **Step 3: Build and live test**

```bash
go build -o /tmp/ac ./cmd/ac/

# Dry-run to verify MCP config is generated
/tmp/ac get quick-review --json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps(d.get('mcp_servers',[]), indent=2))"

# Verify the upload mount
/tmp/ac run quick-review --prompt "test" --dry-run 2>/dev/null | grep '.claude.json'
```

- [ ] **Step 4: Update status doc**

Mark MCP config generation as done in `docs/status-and-next.md`.

- [ ] **Step 5: Commit**

```bash
git add docs/status-and-next.md
git commit -m "feat: MCP config generation end-to-end, update docs"
```
