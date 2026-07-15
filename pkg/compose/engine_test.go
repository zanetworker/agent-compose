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

	engine.Start(context.Background(), "reviewer", RunOpts{})

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
	execErr   error
}

func (m *mockExecutorWithList) CreateSandbox(_ context.Context, name string, spec *ResolvedSpec) error {
	m.sandboxes = append(m.sandboxes, name)
	if m.buf != nil {
		fmt.Fprintf(m.buf, "created sandbox %s\n", name)
	}
	return nil
}

func (m *mockExecutorWithList) UpdatePolicy(_ context.Context, name string, spec *ResolvedSpec) error {
	return nil
}

func (m *mockExecutorWithList) ExecInSandbox(_ context.Context, name string, cmd []string) error {
	if m.buf != nil {
		fmt.Fprintf(m.buf, "exec in %s: %v\n", name, cmd)
	}
	return m.execErr
}

func (m *mockExecutorWithList) ConnectSandbox(_ context.Context, name string) error {
	if m.buf != nil {
		fmt.Fprintf(m.buf, "connect to %s\n", name)
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

func TestEngine_Run_AutoCleanup(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{Name: "reviewer", Runtime: "claude-code", Prompt: "test"}

	exec := &mockExecutorWithList{buf: &bytes.Buffer{}}
	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	run, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// After Run completes, sandbox should be auto-deleted
	if len(exec.sandboxes) != 0 {
		t.Errorf("expected 0 sandboxes after Run, got %d: %v", len(exec.sandboxes), exec.sandboxes)
	}
	output := exec.buf.String()
	if !strings.Contains(output, "deleted sandbox "+run.Sandbox) {
		t.Errorf("expected DeleteSandbox call in output: %s", output)
	}
}

func TestEngine_Run_AutoCleanup_OnError(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{Name: "reviewer", Runtime: "claude-code", Prompt: "test"}

	exec := &mockExecutorWithList{buf: &bytes.Buffer{}}
	exec.execErr = &ExitError{Code: 1, Err: fmt.Errorf("agent failed")}
	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	_, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err == nil {
		t.Fatal("expected error")
	}

	if len(exec.sandboxes) != 0 {
		t.Errorf("expected 0 sandboxes after failed Run, got %d", len(exec.sandboxes))
	}
}

func TestEngine_Start(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{Name: "reviewer", Runtime: "claude-code", Prompt: "Review."}

	exec := &mockExecutorWithList{buf: &bytes.Buffer{}}
	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	run, err := engine.Start(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if run.Agent != "reviewer" {
		t.Errorf("run.Agent = %q", run.Agent)
	}

	// After Start, sandbox should NOT be deleted (persists)
	if len(exec.sandboxes) != 1 {
		t.Fatalf("expected 1 sandbox after Start, got %d", len(exec.sandboxes))
	}

	// Exec should have been called (background agent)
	output := exec.buf.String()
	if !strings.Contains(output, "exec in") {
		t.Errorf("expected ExecInSandbox call: %s", output)
	}
}

func TestEngine_List_FromExecutor(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{Name: "reviewer", Runtime: "claude-code", Prompt: "test"}

	var buf bytes.Buffer
	exec := &mockExecutorWithList{buf: &buf}

	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	// Start (not Run) so the sandbox persists for listing
	run, err := engine.Start(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

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

func TestEngine_Run_Progress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtimes["bare"] = RuntimeProfile{
		Kind:       "harness",
		Image:      "test:latest",
		EnvMapping: map[string]string{},
		Entrypoint: []string{"agent"},
	}
	cfg.Agents = map[string]Agent{
		"simple": {Name: "simple", Runtime: "bare", Prompt: "Hello."},
	}
	cfg.Defaults.Inference = ""

	var dryBuf, progressBuf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&dryBuf)),
		WithProgress(&progressBuf),
	)

	_, err := engine.Run(context.Background(), "simple", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	progress := progressBuf.String()
	if !strings.Contains(progress, "Creating sandbox") {
		t.Errorf("missing 'Creating sandbox' in progress: %s", progress)
	}
	if !strings.Contains(progress, "Running agent") {
		t.Errorf("missing 'Running agent' in progress: %s", progress)
	}
	if strings.Contains(progress, "Updating egress policy") {
		t.Errorf("should not show egress message when no egress rules: %s", progress)
	}
}

func TestEngine_Run_Progress_WithEgress(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		Prompt:  "Review code.",
		MCP:     []string{"github"},
	}

	var dryBuf, progressBuf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&dryBuf)),
		WithSkillsDir(t.TempDir()),
		WithProgress(&progressBuf),
	)

	_, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	progress := progressBuf.String()
	if !strings.Contains(progress, "Updating egress policy") {
		t.Errorf("missing 'Updating egress policy' in progress: %s", progress)
	}

	createIdx := strings.Index(progress, "Creating sandbox")
	egressIdx := strings.Index(progress, "Updating egress policy")
	runIdx := strings.Index(progress, "Running agent")
	if createIdx >= egressIdx || egressIdx >= runIdx {
		t.Errorf("progress messages out of order: create=%d, egress=%d, run=%d", createIdx, egressIdx, runIdx)
	}
}

func TestEngine_Run_PropagatesExitError(t *testing.T) {
	cfg := testConfig()
	cfg.Agents["reviewer"] = Agent{
		Name:    "reviewer",
		Runtime: "claude-code",
		Prompt:  "Review code.",
	}

	exec := &mockExecutorWithList{}
	exec.execErr = &ExitError{Code: 2, Err: fmt.Errorf("process exited with code 2")}

	engine := New(
		WithConfig(cfg),
		WithExecutor(exec),
		WithSkillsDir(t.TempDir()),
	)

	_, err := engine.Run(context.Background(), "reviewer", RunOpts{})
	if err == nil {
		t.Fatal("expected error")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.Code)
	}
}

func TestEngine_Attach(t *testing.T) {
	cfg := testConfig()
	exec := &mockExecutorWithList{buf: &bytes.Buffer{}}
	exec.sandboxes = []string{"reviewer-123"}

	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	err := engine.Attach(context.Background(), "reviewer-123", AttachOpts{Shell: true})
	if err != nil {
		t.Fatalf("Attach failed: %v", err)
	}

	output := exec.buf.String()
	if !strings.Contains(output, "connect to reviewer-123") {
		t.Errorf("expected ConnectSandbox call, got: %s", output)
	}
}

func TestEngine_Attach_NotFound(t *testing.T) {
	cfg := testConfig()
	exec := &mockExecutorWithList{}

	engine := New(WithConfig(cfg), WithExecutor(exec), WithSkillsDir(t.TempDir()))

	err := engine.Attach(context.Background(), "nonexistent", AttachOpts{})
	if err == nil {
		t.Fatal("expected error for unknown sandbox")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestEngine_Run_FrameworkAgent_PromptUploaded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtimes["test-framework"] = RuntimeProfile{
		Kind:       "framework",
		Image:      "python:3.12-slim",
		EnvMapping: map[string]string{},
		Entrypoint: []string{"python", "-m", "agent"},
	}
	cfg.Agents["fw-agent"] = Agent{
		Runtime: "test-framework",
		Prompt:  "You are a helpful assistant",
	}
	cfg.Defaults.Inference = ""

	var buf bytes.Buffer
	engine := New(
		WithConfig(cfg),
		WithExecutor(NewDryRunExecutor(&buf)),
	)

	_, err := engine.Run(context.Background(), "fw-agent", RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--upload") || !strings.Contains(output, "prompt.md") {
		t.Errorf("expected prompt file upload for framework agent, got: %s", output)
	}
}
