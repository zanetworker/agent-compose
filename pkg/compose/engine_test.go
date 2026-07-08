package compose

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func testEngine(t *testing.T) (*Engine, *bytes.Buffer) {
	t.Helper()
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{
		Name:    "reviewer",
		Harness: "claude-code",
		Prompt:  "Review code.",
		MCP:     []string{"github"},
	}

	var buf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&buf)),
		WithSkillsDir(t.TempDir()),
		WithStore(NewMemoryStore()),
	)
	return engine, &buf
}

func TestEngine_Resolve(t *testing.T) {
	engine, _ := testEngine(t)

	spec, err := engine.Resolve(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if spec.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("image = %q", spec.Image)
	}
}

func TestEngine_Resolve_NotFound(t *testing.T) {
	engine, _ := testEngine(t)

	_, err := engine.Resolve(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestEngine_Run_DryRun(t *testing.T) {
	engine, buf := testEngine(t)

	run, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if run.Agent != "reviewer" {
		t.Errorf("run.Agent = %q", run.Agent)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox create") {
		t.Errorf("missing sandbox create in output: %s", output)
	}
	if !strings.Contains(output, "openshell sandbox exec") {
		t.Errorf("missing sandbox exec in output: %s", output)
	}
}

func TestEngine_Run_WithPromptOverride(t *testing.T) {
	engine, buf := testEngine(t)

	_, err := engine.Run(context.Background(), "reviewer", RunOpts{Prompt: "Focus on auth."})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox exec") {
		t.Errorf("missing exec: %s", output)
	}
}

func TestEngine_Run_InlineAgent(t *testing.T) {
	engine, buf := testEngine(t)

	_, err := engine.Run(context.Background(), "", RunOpts{
		Agent: &Agent{
			Name:      "inline-test",
			Harness:   "claude-code",
			Inference: "maas",
			Prompt:    "Inline agent.",
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox create") {
		t.Errorf("missing sandbox create: %s", output)
	}
}

func TestEngine_Stop(t *testing.T) {
	engine, buf := testEngine(t)

	engine.Run(context.Background(), "reviewer", RunOpts{})
	buf.Reset()

	err := engine.Stop(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox delete") {
		t.Errorf("missing sandbox delete: %s", output)
	}
}

func TestEngine_List(t *testing.T) {
	engine, _ := testEngine(t)
	engine.Run(context.Background(), "reviewer", RunOpts{})

	agents, err := engine.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("len = %d, want 1", len(agents))
	}
	if agents[0].Name != "reviewer" {
		t.Errorf("name = %q, want reviewer", agents[0].Name)
	}
}

func TestEngine_Logs(t *testing.T) {
	engine, _ := testEngine(t)
	engine.Run(context.Background(), "reviewer", RunOpts{})

	logs, err := engine.Logs(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(logs)
	output := buf.String()
	if !strings.Contains(output, "[dry-run] logs for") {
		t.Errorf("unexpected logs: %s", output)
	}
}

func TestEngine_Inspect(t *testing.T) {
	engine, _ := testEngine(t)

	spec, err := engine.Inspect(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if spec.Name != "reviewer" {
		t.Errorf("name = %q, want reviewer", spec.Name)
	}
}

func TestEngine_Validate(t *testing.T) {
	engine, _ := testEngine(t)

	// Validate should pass for well-formed config
	// We'll test with a temp file
	tmpfile := t.TempDir() + "/config.yaml"
	data := `
harnesses:
  test:
    image: example:latest
agents:
  test:
    harness: test
`
	if err := os.WriteFile(tmpfile, []byte(data), 0644); err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	errs, err := engine.Validate(context.Background(), tmpfile)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}
