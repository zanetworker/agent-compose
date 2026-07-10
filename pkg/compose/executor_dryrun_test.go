package compose

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDryRunExecutor_CreateSandbox(t *testing.T) {
	var buf bytes.Buffer
	ex := NewDryRunExecutor(&buf)

	spec := &ResolvedSpec{
		Image:     "ghcr.io/anthropics/claude-code:latest",
		Providers: []string{"maas-anthropic", "github-pat"},
		Env:       map[string]string{"ANTHROPIC_BASE_URL": "https://maas.example.com/v1"},
		Policy:    "restricted",
	}

	err := ex.CreateSandbox(context.Background(), "test-sandbox", spec)
	if err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox create") {
		t.Errorf("output missing 'openshell sandbox create': %s", output)
	}
	if !strings.Contains(output, "--image ghcr.io/anthropics/claude-code:latest") {
		t.Errorf("output missing --image: %s", output)
	}
	if !strings.Contains(output, "--provider maas-anthropic") {
		t.Errorf("output missing --provider maas-anthropic: %s", output)
	}
	if !strings.Contains(output, "--policy restricted") {
		t.Errorf("output missing --policy: %s", output)
	}
}

func TestDryRunExecutor_ExecInSandbox(t *testing.T) {
	var buf bytes.Buffer
	ex := NewDryRunExecutor(&buf)

	err := ex.ExecInSandbox(context.Background(), "test-sandbox", []string{"claude", "--prompt-file", "/workspace/prompt.md"})
	if err != nil {
		t.Fatalf("ExecInSandbox failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox exec") {
		t.Errorf("output missing 'openshell sandbox exec': %s", output)
	}
	if !strings.Contains(output, "test-sandbox") {
		t.Errorf("output missing sandbox name: %s", output)
	}
}

func TestDryRunExecutor_CreateSandbox_WithSandboxOpts(t *testing.T) {
	var buf bytes.Buffer
	exec := NewDryRunExecutor(&buf)
	spec := &ResolvedSpec{
		Image:   "test:latest",
		Sandbox: SandboxOpts{Scope: "session", Mode: "all", TTL: "30m"},
		Labels:  map[string]string{"agentctl.io/agent": "reviewer"},
		Env:     map[string]string{},
	}
	exec.CreateSandbox(nil, "test-sandbox", spec)
	output := buf.String()

	if !strings.Contains(output, "--scope session") {
		t.Errorf("expected --scope session in output: %s", output)
	}
	if !strings.Contains(output, "--ttl 30m") {
		t.Errorf("expected --ttl 30m in output: %s", output)
	}
	if !strings.Contains(output, "--mode all") {
		t.Errorf("expected --mode all in output: %s", output)
	}
	if !strings.Contains(output, "--label agentctl.io/agent=reviewer") {
		t.Errorf("expected --label in output: %s", output)
	}
}
