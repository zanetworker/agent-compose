package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

type CLIExecutor struct {
	binary string
}

func NewCLIExecutor(binary string) *CLIExecutor {
	return &CLIExecutor{binary: binary}
}

func (e *CLIExecutor) BinaryPath() string {
	return e.binary
}

func (e *CLIExecutor) CreateSandbox(ctx context.Context, name string, spec *ResolvedSpec) error {
	args := []string{"sandbox", "create", "--name", name, "--from", spec.Image, "--auto-providers", "--no-tty"}
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
	labelKeys := make([]string, 0, len(spec.Labels))
	for k := range spec.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, spec.Labels[k]))
	}
	return e.run(ctx, args...)
}

func (e *CLIExecutor) ExecInSandbox(ctx context.Context, name string, cmd []string) error {
	args := append([]string{"sandbox", "exec", "--name", name, "--"}, cmd...)
	return e.run(ctx, args...)
}

func (e *CLIExecutor) DeleteSandbox(ctx context.Context, name string) error {
	return e.run(ctx, "sandbox", "delete", name)
}

func (e *CLIExecutor) SandboxLogs(ctx context.Context, name string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, e.binary, "logs", name)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("sandbox logs pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("sandbox logs start: %w", err)
	}
	return stdout, nil
}

func (e *CLIExecutor) SandboxStatus(ctx context.Context, name string) (SandboxState, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, e.binary, "sandbox", "list", "--names")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return SandboxUnknown, nil
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(line) == name {
			return SandboxRunning, nil
		}
	}
	return SandboxStopped, nil
}

func (e *CLIExecutor) ListSandboxes(ctx context.Context, labelSelector string) ([]string, error) {
	args := []string{"sandbox", "list", "--names"}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, e.binary, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sandbox list: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var names []string
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			names = append(names, l)
		}
	}
	return names, nil
}

func (e *CLIExecutor) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, e.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w\nstderr: %s", e.binary, strings.Join(args, " "), err, stderr.String())
	}
	return nil
}
