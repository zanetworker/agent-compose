package compose

import (
	"context"
	"fmt"
	"io"
	"time"
)

type RunOpts struct {
	Prompt    string
	Workspace string
	Agent     *Agent // inline agent (when name is empty)
}

type Engine struct {
	config            *Config
	resolver          *Resolver
	executor          Executor
	skillsDir         string
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

func (e *Engine) Run(ctx context.Context, name string, opts RunOpts) (*Run, error) {
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
			return nil, err
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

	spec, err := e.resolver.Resolve(ctx, *agent)
	if err != nil {
		return nil, fmt.Errorf("resolving agent %q: %w", name, err)
	}

	sandboxName := fmt.Sprintf("%s-%d", name, time.Now().Unix())

	// Ensure labels map exists and add agent label for tracking
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	spec.Labels["agentctl.io/agent"] = name

	if err := e.executor.CreateSandbox(ctx, sandboxName, spec); err != nil {
		return nil, fmt.Errorf("creating sandbox: %w", err)
	}

	if err := e.executor.ExecInSandbox(ctx, sandboxName, spec.Entrypoint); err != nil {
		e.executor.DeleteSandbox(ctx, sandboxName)
		return nil, fmt.Errorf("executing entrypoint: %w", err)
	}

	run := &Run{
		ID:        sandboxName,
		Agent:     name,
		Sandbox:   sandboxName,
		StartedAt: time.Now().Unix(),
		Status:    SandboxRunning,
	}

	return run, nil
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

func (e *Engine) Inspect(ctx context.Context, name string) (*ResolvedSpec, error) {
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
