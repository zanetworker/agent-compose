//go:build integration

package compose

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestIntegration_FullResolveAndDryRun(t *testing.T) {
	// Create a config with inference, MCP, and agent
	cfg := DefaultConfig()
	cfg.Inference["test-maas"] = InferenceSpec{
		Endpoint:     "https://maas.test.com/v1",
		Provider:     "test-provider",
		DefaultModel: "test-model",
		Egress:       []string{"maas.test.com:443"},
	}
	cfg.MCP["test-github"] = MCPSpec{
		Provider: "gh-test",
		Egress:   []string{"api.github.com:443"},
	}
	cfg.Defaults.Inference = "test-maas"
	cfg.Agents["test-agent"] = Agent{
		Harness: "claude-code",
		Prompt:  "Test prompt.",
		MCP:     []string{"test-github"},
	}

	// Build engine with DryRunExecutor writing to buffer
	var buf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&buf)),
		WithSkillsDir(t.TempDir()),
	)

	// Run the agent
	run, err := engine.Run(context.Background(), "test-agent", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check output contains sandbox create with correct parameters
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (create + exec), got %d:\n%s", len(lines), output)
	}

	createLine := lines[0]

	// Verify sandbox create command contains expected elements
	if !strings.Contains(createLine, "openshell sandbox create") {
		t.Errorf("create missing 'openshell sandbox create': %s", createLine)
	}
	if !strings.Contains(createLine, "--image ghcr.io/anthropics/claude-code:latest") {
		t.Errorf("create missing --image: %s", createLine)
	}
	if !strings.Contains(createLine, "--provider test-provider") {
		t.Errorf("create missing inference provider: %s", createLine)
	}
	if !strings.Contains(createLine, "--provider gh-test") {
		t.Errorf("create missing mcp provider: %s", createLine)
	}
	if !strings.Contains(createLine, "ANTHROPIC_BASE_URL=https://maas.test.com/v1") {
		t.Errorf("create missing env var: %s", createLine)
	}

	// Verify run result
	if run.Agent != "test-agent" {
		t.Errorf("run.Agent = %q, want test-agent", run.Agent)
	}

	// Stop the agent and verify delete command
	buf.Reset()
	err = engine.Stop(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	stopOutput := buf.String()
	if !strings.Contains(stopOutput, "openshell sandbox delete") {
		t.Errorf("stop missing 'openshell sandbox delete': %s", stopOutput)
	}
}
