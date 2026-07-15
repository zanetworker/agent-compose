package compose

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIExecutor_BinaryNotFound(t *testing.T) {
	ex := NewCLIExecutor("/nonexistent/openshell", os.Stdin, os.Stdout, os.Stderr)
	spec := &ResolvedSpec{Image: "test:latest"}

	err := ex.CreateSandbox(context.Background(), "test", spec)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestCLIExecutor_ExecInSandbox_StreamsOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ex := NewCLIExecutor("echo", os.Stdin, &stdout, &stderr)

	err := ex.ExecInSandbox(context.Background(), "test", []string{"hello", "world"})
	if err != nil {
		t.Fatalf("ExecInSandbox failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected stdout to contain 'hello world', got: %q", out)
	}
}

func TestCLIExecutor_ExecInSandbox_PropagatesExitCode(t *testing.T) {
	script := filepath.Join(t.TempDir(), "exit42.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 42\n"), 0755)

	var stdout, stderr bytes.Buffer
	ex := NewCLIExecutor(script, os.Stdin, &stdout, &stderr)

	err := ex.ExecInSandbox(context.Background(), "test", []string{"ignored"})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("exit code = %d, want 42", exitErr.Code)
	}
}

func TestCLIExecutor_CreateSandbox_OutputUsesInjectedWriter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ex := NewCLIExecutor("echo", os.Stdin, &stdout, &stderr)

	spec := &ResolvedSpec{Image: "test:latest", Env: map[string]string{}}
	err := ex.CreateSandbox(context.Background(), "test", spec)
	if err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "sandbox create") {
		t.Errorf("expected injected stdout to receive output, got: %q", out)
	}
}
