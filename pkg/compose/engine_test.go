package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func testEngine(t *testing.T) (*Engine, *bytes.Buffer) {
	t.Helper()
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		Prompt:  "Review code.",
		MCP:     []string{"github"},
	}

	var buf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&buf)),
		WithSkillsDir(t.TempDir()),
	)
	return engine, &buf
}

func TestEngine_Resolve(t *testing.T) {
	engine, _ := testEngine(t)

	spec, err := engine.Resolve(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if spec.Image != "ghcr.io/nvidia/openshell-community/sandboxes/base:latest" {
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
			Runtime:   "claude-code",
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

	run, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	buf.Reset()

	err = engine.Stop(context.Background(), run.Sandbox)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openshell sandbox delete") {
		t.Errorf("missing sandbox delete: %s", output)
	}
}

func TestEngine_List(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		Prompt:  "Review code.",
		MCP:     []string{"github"},
	}

	exec := &mockExecutorWithList{}
	engine := New(
		WithConfig(cfg),
		WithExecutor(exec),
		WithSkillsDir(t.TempDir()),
	)

	engine.Run(context.Background(), "reviewer", RunOpts{})

	agents, err := engine.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("len = %d, want 1", len(agents))
	}
}

func TestEngine_Logs(t *testing.T) {
	engine, _ := testEngine(t)
	run, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logs, err := engine.Logs(context.Background(), run.Sandbox)
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

func TestEngine_Get(t *testing.T) {
	engine, _ := testEngine(t)

	spec, err := engine.Get(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
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
runtimes:
  test:
    kind: harness
    image: example:latest
agents:
  test:
    runtime: test
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

// mockExecutorWithList implements Executor with a fake ListSandboxes method
type mockExecutorWithList struct {
	sandboxes []string
	buf       *bytes.Buffer
}

func (m *mockExecutorWithList) CreateSandbox(_ context.Context, name string, spec *ResolvedSpec) error {
	m.sandboxes = append(m.sandboxes, name)
	if m.buf != nil {
		fmt.Fprintf(m.buf, "created sandbox %s\n", name)
	}
	return nil
}

func (m *mockExecutorWithList) ExecInSandbox(_ context.Context, name string, cmd []string) error {
	if m.buf != nil {
		fmt.Fprintf(m.buf, "exec in %s: %v\n", name, cmd)
	}
	return nil
}

func (m *mockExecutorWithList) DeleteSandbox(_ context.Context, name string) error {
	for i, s := range m.sandboxes {
		if s == name {
			m.sandboxes = append(m.sandboxes[:i], m.sandboxes[i+1:]...)
			break
		}
	}
	if m.buf != nil {
		fmt.Fprintf(m.buf, "deleted sandbox %s\n", name)
	}
	return nil
}

func (m *mockExecutorWithList) SandboxLogs(_ context.Context, name string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(fmt.Sprintf("[mock] logs for %s\n", name))), nil
}

func (m *mockExecutorWithList) SandboxStatus(_ context.Context, name string) (SandboxState, error) {
	for _, s := range m.sandboxes {
		if s == name {
			return SandboxRunning, nil
		}
	}
	return SandboxUnknown, nil
}

func (m *mockExecutorWithList) ListSandboxes(_ context.Context, labelSelector string) ([]string, error) {
	return m.sandboxes, nil
}

func TestEngine_List_FromExecutor(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{Name: "reviewer", Runtime: "claude-code", Prompt: "test"}

	var buf bytes.Buffer
	exec := &mockExecutorWithList{buf: &buf}

	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	// Run an agent to create a sandbox
	run, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// List should query the executor, not the store
	statuses, err := engine.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Sandbox != run.Sandbox {
		t.Errorf("expected sandbox %q, got %q", run.Sandbox, statuses[0].Sandbox)
	}
}
