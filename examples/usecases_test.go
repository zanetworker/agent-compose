package examples_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/zanetworker/agent-compose/pkg/compose"
)

// --- Use case: Fan out the same agent across multiple repos ---

func TestSDK_BatchReviewMultipleRepos(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.Agents["reviewer"] = compose.Agent{
		Runtime: "claude-code",
		Prompt:  "Review for security issues. Be concise.",
	}

	var buf bytes.Buffer
	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(&buf)),
	)

	repos := []string{"./api-server", "./web-frontend", "./auth-service"}

	for _, repo := range repos {
		buf.Reset()
		run, err := engine.Start(context.Background(), "reviewer", compose.RunOpts{
			Workspace: repo,
		})
		if err != nil {
			t.Fatalf("Start %s: %v", repo, err)
		}

		output := buf.String()
		if !strings.Contains(output, "openshell sandbox create") {
			t.Errorf("repo %s: missing sandbox create", repo)
		}
		if !strings.Contains(output, repo) {
			t.Errorf("repo %s: workspace not in command", repo)
		}

		t.Logf("Started %s in sandbox %s", repo, run.Sandbox)
	}
}

// --- Use case: Dashboard previews spec before launching ---

func TestSDK_PreviewBeforeLaunch(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.Inference["maas"] = compose.InferenceSpec{
		Endpoint:     "https://maas.example.com/v1",
		Provider:     "maas-anthropic",
		DefaultModel: "granite-3.3-8b",
		Egress:       []string{"maas.example.com:443"},
	}
	cfg.MCP["github"] = compose.MCPSpec{
		Type:     "stdio",
		Command:  "github-mcp-server",
		Provider: "github-pat",
		Egress:   []string{"api.github.com:443"},
	}
	cfg.Agents["reviewer"] = compose.Agent{
		Runtime:   "claude-code",
		Inference: "maas",
		MCP:       []string{"github"},
		Prompt:    "Review code.",
	}

	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(os.Stdout)),
	)

	// Step 1: Resolve to preview (no sandbox created)
	spec, err := engine.Resolve(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// A dashboard would display this to the user
	preview := map[string]interface{}{
		"image":     spec.Image,
		"model":     spec.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"],
		"providers": spec.Providers,
		"egress":    spec.Egress,
		"mcp":       spec.MCPServers,
		"prompt":    spec.Prompt,
	}
	data, _ := json.MarshalIndent(preview, "", "  ")
	t.Logf("Preview for user:\n%s", string(data))

	// Step 2: User clicks "Run"
	run, err := engine.Run(context.Background(), "reviewer", compose.RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("Launched sandbox: %s", run.Sandbox)
}

// --- Use case: Compose with MCP servers ---

func TestSDK_ComposeWithMCP(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.MCP["github"] = compose.MCPSpec{
		Type:     "stdio",
		Command:  "github-mcp-server",
		Args:     []string{"--read-only"},
		Provider: "github-pat",
		Egress:   []string{"api.github.com:443"},
	}
	cfg.MCP["sentry"] = compose.MCPSpec{
		Type:   "http",
		URL:    "https://mcp.sentry.dev/mcp",
		Egress: []string{"mcp.sentry.dev:443"},
	}
	cfg.Agents["debugger"] = compose.Agent{
		Runtime: "claude-code",
		MCP:     []string{"github", "sentry"},
		Prompt:  "Debug this issue using GitHub and Sentry.",
	}

	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(os.Stdout)),
	)

	spec, err := engine.Resolve(context.Background(), "debugger")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Verify MCP servers are resolved
	if len(spec.MCPServers) != 2 {
		t.Fatalf("MCPServers = %d, want 2", len(spec.MCPServers))
	}

	// Verify egress is merged from both MCP servers
	egress := strings.Join(spec.Egress, " ")
	if !strings.Contains(egress, "api.github.com") {
		t.Error("missing github egress")
	}
	if !strings.Contains(egress, "mcp.sentry.dev") {
		t.Error("missing sentry egress")
	}

	// Verify providers are merged
	providers := strings.Join(spec.Providers, " ")
	if !strings.Contains(providers, "github-pat") {
		t.Error("missing github provider")
	}

	t.Logf("MCP servers: %v", spec.MCPServers)
	t.Logf("Egress: %v", spec.Egress)
}

// --- Use case: Custom framework runtime ---

func TestSDK_CustomRuntime(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.Runtimes["my-agent"] = compose.RuntimeProfile{
		Kind:  "framework",
		Image: "my-registry.com/my-agent:latest",
		EnvMapping: map[string]string{
			"LLM_ENDPOINT": "${endpoint}",
			"LLM_MODEL":    "${model}",
		},
		Entrypoint: []string{"python3", "/app/agent.py"},
	}
	cfg.Inference["local"] = compose.InferenceSpec{
		Endpoint:     "http://localhost:8000/v1",
		DefaultModel: "qwen3-14b",
	}
	cfg.Agents["my-agent"] = compose.Agent{
		Runtime:   "my-agent",
		Inference: "local",
		Prompt:    "Analyze the data.",
	}

	var buf bytes.Buffer
	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(&buf)),
	)

	_, err := engine.Run(context.Background(), "my-agent", compose.RunOpts{
		Workspace: "./data-pipeline",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-registry.com/my-agent:latest") {
		t.Error("custom image not in command")
	}
	if !strings.Contains(output, "python3") {
		t.Error("custom entrypoint not in command")
	}
	if !strings.Contains(output, "./data-pipeline") {
		t.Error("workspace not uploaded")
	}

	t.Logf("Commands:\n%s", output)
}

// --- Use case: Background agent with output collection ---

func TestSDK_BackgroundAgent(t *testing.T) {
	cfg := compose.DefaultConfig()
	cfg.Agents["worker"] = compose.Agent{
		Runtime: "claude-code",
		Prompt:  "Refactor the auth module.",
	}

	var buf bytes.Buffer
	engine := compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(compose.NewDryRunExecutor(&buf)),
	)

	// Start returns immediately
	run, err := engine.Start(context.Background(), "worker", compose.RunOpts{
		Workspace: "./my-repo",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Logf("Agent %s started in sandbox %s", run.Agent, run.Sandbox)

	// In a real app, you'd poll for completion:
	//   output, _ := engine.AgentOutput(ctx, run.Sandbox)
	//   engine.Stop(ctx, run.Sandbox)

	output := buf.String()
	if !strings.Contains(output, "setsid") {
		t.Error("expected background exec via setsid")
	}

	// In production with a real executor, you'd check:
	//   statuses, _ := engine.List(ctx)  // shows running sandboxes
	//   engine.Stop(ctx, run.Sandbox)    // clean up when done

	_ = fmt.Sprintf("collected output from %s", run.Sandbox)
}
