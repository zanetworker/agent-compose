package compose

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

type DryRunExecutor struct {
	out io.Writer
}

func NewDryRunExecutor(out io.Writer) *DryRunExecutor {
	return &DryRunExecutor{out: out}
}

func (e *DryRunExecutor) CreateSandbox(_ context.Context, name string, spec *ResolvedSpec) error {
	args := []string{"openshell", "sandbox", "create", "--name", name, "--from", spec.Image}
	for _, p := range spec.Providers {
		args = append(args, "--provider", p)
	}
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--env", fmt.Sprintf("%s=%s", k, spec.Env[k]))
	}
	if spec.Policy != "" {
		args = append(args, "--policy", spec.Policy)
	}
	for _, m := range spec.SkillMounts {
		args = append(args, "--upload", fmt.Sprintf("%s:%s", m.Source, m.Target))
	}
	if spec.Workspace != "" {
		args = append(args, "--upload", spec.Workspace)
	}
	if spec.Sandbox.Scope != "" {
		args = append(args, "--scope", spec.Sandbox.Scope)
	}
	if spec.Sandbox.Mode != "" {
		args = append(args, "--mode", spec.Sandbox.Mode)
	}
	if spec.Sandbox.TTL != "" {
		args = append(args, "--ttl", spec.Sandbox.TTL)
	}
	// Labels sorted for determinism
	labelKeys := make([]string, 0, len(spec.Labels))
	for k := range spec.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, spec.Labels[k]))
	}
	fmt.Fprintln(e.out, strings.Join(args, " "))
	return nil
}

func (e *DryRunExecutor) UpdatePolicy(_ context.Context, name string, spec *ResolvedSpec) error {
	if len(spec.Egress) == 0 {
		return nil
	}
	args := []string{"openshell", "policy", "update", name}
	for _, endpoint := range spec.Egress {
		args = append(args, "--add-endpoint", endpoint+":read-write:rest:enforce")
	}
	if len(spec.Entrypoint) > 0 {
		args = append(args, "--binary", spec.Entrypoint[0])
	}
	fmt.Fprintln(e.out, strings.Join(args, " "))
	return nil
}

func (e *DryRunExecutor) ExecInSandbox(_ context.Context, name string, cmd []string) error {
	args := append([]string{"openshell", "sandbox", "exec", "--name", name, "--"}, cmd...)
	fmt.Fprintln(e.out, strings.Join(args, " "))
	return nil
}

func (e *DryRunExecutor) ConnectSandbox(_ context.Context, name string) error {
	fmt.Fprintf(e.out, "openshell sandbox connect %s\n", name)
	return nil
}

func (e *DryRunExecutor) DeleteSandbox(_ context.Context, name string) error {
	fmt.Fprintf(e.out, "openshell sandbox delete %s\n", name)
	return nil
}

func (e *DryRunExecutor) SandboxLogs(_ context.Context, name string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(fmt.Sprintf("[dry-run] logs for %s\n", name))), nil
}

func (e *DryRunExecutor) SandboxStatus(_ context.Context, name string) (SandboxState, error) {
	return SandboxUnknown, nil
}

func (e *DryRunExecutor) ListSandboxes(_ context.Context, labelSelector string) ([]string, error) {
	return nil, nil
}
