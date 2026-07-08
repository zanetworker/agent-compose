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
