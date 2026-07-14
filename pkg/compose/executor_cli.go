package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
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
	for _, m := range spec.SkillMounts {
		args = append(args, "--upload", fmt.Sprintf("%s:%s", m.Source, m.Target))
	}
	if spec.Workspace != "" {
		args = append(args, "--upload", spec.Workspace)
	}
	labelKeys := make([]string, 0, len(spec.Labels))
	for k := range spec.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, spec.Labels[k]))
	}
	args = append(args, "--", "true")
	return e.run(ctx, args...)
}

func (e *CLIExecutor) UpdatePolicy(ctx context.Context, name string, spec *ResolvedSpec) error {
	if len(spec.Egress) == 0 {
		return nil
	}
	args := []string{"policy", "update", name}
	for _, endpoint := range spec.Egress {
		args = append(args, "--add-endpoint", endpoint+":read-write:rest:enforce")
	}
	if len(spec.Entrypoint) > 0 {
		args = append(args, "--binary", spec.Entrypoint[0])
	}
	if err := e.run(ctx, args...); err != nil {
		return err
	}
	// Policy propagation delay: OpenShell needs ~10s to enforce new egress rules.
	time.Sleep(12 * time.Second)
	return nil
}

func (e *CLIExecutor) ExecInSandbox(ctx context.Context, name string, cmd []string) error {
	args := append([]string{"sandbox", "exec", "--name", name, "--"}, cmd...)
	return e.run(ctx, args...)
}

func (e *CLIExecutor) ConnectSandbox(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, e.binary, "sandbox", "connect", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
	return &waitCloser{ReadCloser: stdout, cmd: cmd}, nil
}

type waitCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (w *waitCloser) Close() error {
	err := w.ReadCloser.Close()
	w.cmd.Wait()
	return err
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
