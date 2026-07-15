package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'\\$`") {
			quoted[i] = "'" + strings.ReplaceAll(a, "'", "'\\''") + "'"
		} else {
			quoted[i] = a
		}
	}
	return strings.Join(quoted, " ")
}

type RunOpts struct {
	Prompt          string
	Workspace       string
	Inference       string // override inference provider for this run
	Model           string // override model for this run
	SkipPermissions bool   // if true, append --dangerously-skip-permissions for harness agents
	Interactive     bool   // if true, use sandbox connect instead of exec
	Agent           *Agent // inline agent (when name is empty)
}

type Engine struct {
	config            *Config
	resolver          *Resolver
	executor          Executor
	skillsDir         string
	progress          io.Writer
	runtimeOverride   RuntimeResolver
	inferenceOverride InferenceResolver
	mcpOverride       MCPResolver
}

func New(opts ...Option) *Engine {
	e := &Engine{}
	for _, opt := range opts {
		opt(e)
	}
	if e.config == nil {
		e.config = DefaultConfig()
	}
	if e.progress == nil {
		e.progress = io.Discard
	}

	runtimes := RuntimeResolver(NewConfigRuntimeResolver(e.config))
	if e.runtimeOverride != nil {
		runtimes = NewChainedRuntimeResolver(e.runtimeOverride, runtimes)
	}
	inference := InferenceResolver(NewConfigInferenceResolver(e.config))
	if e.inferenceOverride != nil {
		inference = e.inferenceOverride
	}
	mcp := MCPResolver(NewConfigMCPResolver(e.config))
	if e.mcpOverride != nil {
		mcp = e.mcpOverride
	}
	skills := SkillResolver(NewLocalSkillResolver(e.skillsDir))
	policy := PolicyResolver(NewConfigPolicyResolver())

	e.resolver = NewResolver(runtimes, inference, mcp, skills, policy, e.config.Defaults)
	return e
}

func (e *Engine) Resolve(ctx context.Context, name string) (*ResolvedSpec, error) {
	agent, err := e.findAgent(name)
	if err != nil {
		return nil, err
	}
	return e.resolver.Resolve(ctx, *agent)
}

func (e *Engine) setup(ctx context.Context, name string, opts RunOpts) (string, *ResolvedSpec, func(), error) {
	var agent *Agent
	if opts.Agent != nil {
		agent = opts.Agent
		if agent.Name == "" {
			agent.Name = fmt.Sprintf("inline-%d", time.Now().Unix())
		}
		name = agent.Name
	} else {
		found, err := e.findAgent(name)
		if err != nil {
			return "", nil, nil, err
		}
		agent = found
	}

	if opts.Prompt != "" {
		agentCopy := *agent
		if agentCopy.Prompt != "" {
			agentCopy.Prompt = agentCopy.Prompt + "\n\n" + opts.Prompt
		} else {
			agentCopy.Prompt = opts.Prompt
		}
		agent = &agentCopy
	}
	if opts.Workspace != "" {
		agentCopy := *agent
		agentCopy.Workspace = opts.Workspace
		agent = &agentCopy
	}
	if opts.Inference != "" {
		agentCopy := *agent
		agentCopy.Inference = opts.Inference
		agent = &agentCopy
	}
	if opts.Model != "" {
		agentCopy := *agent
		agentCopy.Model = opts.Model
		agent = &agentCopy
	}

	spec, err := e.resolver.Resolve(ctx, *agent)
	if err != nil {
		return "", nil, nil, fmt.Errorf("resolving agent %q: %w", name, err)
	}

	var cleanups []func()
	if spec.Prompt != "" && spec.RuntimeKind != "harness" {
		tmpfile, err := os.CreateTemp("", "ac-prompt-*.md")
		if err != nil {
			return "", nil, nil, fmt.Errorf("creating prompt temp file: %w", err)
		}
		cleanups = append(cleanups, func() { os.Remove(tmpfile.Name()) })
		if _, err := tmpfile.WriteString(spec.Prompt); err != nil {
			tmpfile.Close()
			return "", nil, nil, fmt.Errorf("writing prompt file: %w", err)
		}
		tmpfile.Close()
		spec.SkillMounts = append(spec.SkillMounts, Mount{
			Source: tmpfile.Name(),
			Target: "/sandbox/prompt.md",
		})
	}

	sandboxName := fmt.Sprintf("%s-%d", name, time.Now().Unix())

	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	spec.Labels["agentctl.io/agent"] = name

	fmt.Fprintf(e.progress, "Creating sandbox %s...\n", sandboxName)
	if err := e.executor.CreateSandbox(ctx, sandboxName, spec); err != nil {
		for _, fn := range cleanups {
			fn()
		}
		return "", nil, nil, fmt.Errorf("creating sandbox: %w", err)
	}

	if len(spec.Egress) > 0 {
		fmt.Fprintf(e.progress, "Updating egress policy...\n")
	}
	if err := e.executor.UpdatePolicy(ctx, sandboxName, spec); err != nil {
		e.executor.DeleteSandbox(context.Background(), sandboxName)
		for _, fn := range cleanups {
			fn()
		}
		return "", nil, nil, fmt.Errorf("updating policy: %w", err)
	}

	cleanup := func() {
		for _, fn := range cleanups {
			fn()
		}
	}
	return sandboxName, spec, cleanup, nil
}

func (e *Engine) buildCmd(spec *ResolvedSpec, opts RunOpts) []string {
	cmd := append([]string{}, spec.Entrypoint...)
	if spec.MCPConfigPath != "" {
		cmd = append(cmd, "--mcp-config", spec.MCPConfigPath)
	}
	if spec.Prompt != "" && spec.RuntimeKind == "harness" {
		cmd = append(cmd, "-p", spec.Prompt)
		if opts.SkipPermissions {
			cmd = append(cmd, "--dangerously-skip-permissions")
		}
	}
	return cmd
}

func (e *Engine) Run(ctx context.Context, name string, opts RunOpts) (*Run, error) {
	sandboxName, spec, cleanup, err := e.setup(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	defer e.executor.DeleteSandbox(context.Background(), sandboxName)

	if opts.Interactive {
		cmd := append([]string{}, spec.Entrypoint...)
		if spec.MCPConfigPath != "" {
			cmd = append(cmd, "--mcp-config", spec.MCPConfigPath)
		}
		if opts.SkipPermissions && spec.RuntimeKind == "harness" {
			cmd = append(cmd, "--dangerously-skip-permissions")
		}
		fmt.Fprintf(e.progress, "Running agent...\n")
		if err := e.executor.ExecInSandbox(ctx, sandboxName, cmd); err != nil {
			return nil, fmt.Errorf("executing entrypoint: %w", err)
		}
	} else {
		cmd := e.buildCmd(spec, opts)
		fmt.Fprintf(e.progress, "Running agent...\n")
		if err := e.executor.ExecInSandbox(ctx, sandboxName, cmd); err != nil {
			return nil, fmt.Errorf("executing entrypoint: %w", err)
		}
	}

	return &Run{
		ID:        sandboxName,
		Agent:     name,
		Sandbox:   sandboxName,
		StartedAt: time.Now().Unix(),
		Status:    SandboxStopped,
	}, nil
}

func (e *Engine) Start(ctx context.Context, name string, opts RunOpts) (*Run, error) {
	sandboxName, spec, cleanup, err := e.setup(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cmd := e.buildCmd(spec, opts)
	fmt.Fprintf(e.progress, "Starting agent in background...\n")
	bgCmd := []string{"sh", "-c", "setsid " + shellJoin(cmd) + " > /tmp/agent.log 2>&1 &"}
	if err := e.executor.ExecInSandbox(ctx, sandboxName, bgCmd); err != nil {
		e.executor.DeleteSandbox(context.Background(), sandboxName)
		return nil, fmt.Errorf("starting agent: %w", err)
	}

	return &Run{
		ID:        sandboxName,
		Agent:     name,
		Sandbox:   sandboxName,
		StartedAt: time.Now().Unix(),
		Status:    SandboxRunning,
	}, nil
}

func (e *Engine) Attach(ctx context.Context, name string, opts AttachOpts) error {
	state, err := e.executor.SandboxStatus(ctx, name)
	if err != nil {
		return fmt.Errorf("checking sandbox status: %w", err)
	}
	if state != SandboxRunning {
		return fmt.Errorf("sandbox %q: %w", name, ErrNotFound)
	}
	return e.executor.ConnectSandbox(ctx, name)
}

func (e *Engine) Stop(ctx context.Context, name string) error {
	if err := e.executor.DeleteSandbox(ctx, name); err != nil {
		return fmt.Errorf("deleting sandbox: %w", err)
	}
	return nil
}

func (e *Engine) List(ctx context.Context) ([]AgentStatus, error) {
	sandboxes, err := e.executor.ListSandboxes(ctx, "agentctl.io/agent")
	if err != nil {
		return nil, err
	}
	statuses := make([]AgentStatus, len(sandboxes))
	for i, s := range sandboxes {
		status, _ := e.executor.SandboxStatus(ctx, s)
		statuses[i] = AgentStatus{
			Name:    s,
			Sandbox: s,
			Status:  status,
		}
	}
	return statuses, nil
}

func (e *Engine) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	return e.executor.SandboxLogs(ctx, name)
}

func (e *Engine) AgentOutput(ctx context.Context, name string) (string, error) {
	openshellBin := "openshell"
	if cliExec, ok := e.executor.(*CLIExecutor); ok {
		openshellBin = cliExec.BinaryPath()
	}
	var buf bytes.Buffer
	c := exec.CommandContext(ctx, openshellBin, "sandbox", "exec", "--name", name, "--", "cat", "/tmp/agent.log")
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("reading agent output: %w", err)
	}
	return buf.String(), nil
}

func (e *Engine) Get(ctx context.Context, name string) (*ResolvedSpec, error) {
	return e.Resolve(ctx, name)
}

func (e *Engine) Validate(_ context.Context, path string) ([]ValidationError, error) {
	_, err := LoadConfig(path)
	if err != nil {
		return []ValidationError{{Path: path, Message: err.Error()}}, nil
	}
	return nil, nil
}

type ValidationError struct {
	Path    string
	Field   string
	Message string
}

func (e *Engine) SyncProfiles(ctx context.Context) ([]string, error) {
	openshellBin := "openshell"
	if cliExec, ok := e.executor.(*CLIExecutor); ok {
		openshellBin = cliExec.BinaryPath()
	}
	return SyncProfiles(ctx, e.config, openshellBin)
}

func (e *Engine) findAgent(name string) (*Agent, error) {
	agent, ok := e.config.Agents[name]
	if !ok {
		return nil, fmt.Errorf("agent %q: %w (not in config.yaml agents section)", name, ErrNotFound)
	}
	if agent.Name == "" {
		agent.Name = name
	}
	return &agent, nil
}
