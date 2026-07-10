package compose

import (
	"encoding/json"
	"testing"
)

func TestSandboxOpts_TTLField(t *testing.T) {
	opts := SandboxOpts{
		Scope: "session",
		Mode:  "all",
		TTL:   "30m",
	}

	if opts.TTL != "30m" {
		t.Errorf("expected TTL to be '30m', got %q", opts.TTL)
	}
}

func TestSandboxOpts_JSONMarshaling(t *testing.T) {
	opts := SandboxOpts{
		Scope: "agent",
		Mode:  "non-main",
		TTL:   "1h",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal SandboxOpts: %v", err)
	}

	var decoded SandboxOpts
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SandboxOpts: %v", err)
	}

	if decoded.Scope != opts.Scope || decoded.Mode != opts.Mode || decoded.TTL != opts.TTL {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, opts)
	}
}

func TestResolvedSpec_RuntimeKindField(t *testing.T) {
	spec := ResolvedSpec{
		Name:        "test-agent",
		RuntimeKind: "harness",
		Image:       "example.com/agent:latest",
	}

	if spec.RuntimeKind != "harness" {
		t.Errorf("expected RuntimeKind to be 'harness', got %q", spec.RuntimeKind)
	}
}

func TestResolvedSpec_SandboxField(t *testing.T) {
	spec := ResolvedSpec{
		Name:  "test-agent",
		Image: "example.com/agent:latest",
		Sandbox: SandboxOpts{
			Scope: "session",
			Mode:  "all",
			TTL:   "45m",
		},
	}

	if spec.Sandbox.Scope != "session" {
		t.Errorf("expected Sandbox.Scope to be 'session', got %q", spec.Sandbox.Scope)
	}
	if spec.Sandbox.TTL != "45m" {
		t.Errorf("expected Sandbox.TTL to be '45m', got %q", spec.Sandbox.TTL)
	}
}

func TestResolvedSpec_JSONMarshaling(t *testing.T) {
	spec := ResolvedSpec{
		Name:        "test-agent",
		Labels:      map[string]string{"env": "test"},
		RuntimeKind: "framework",
		Image:       "example.com/agent:v1",
		Entrypoint:  []string{"/bin/agent"},
		Providers:   []string{"provider1"},
		Env:         map[string]string{"KEY": "value"},
		Egress:      []string{"example.com:443"},
		Policy:      "restricted",
		Tools:       []string{"tool1"},
		Sandbox: SandboxOpts{
			Scope: "shared",
			Mode:  "non-main",
			TTL:   "2h",
		},
		Prompt:      "test prompt",
		SkillMounts: []Mount{{Source: "/src", Target: "/dst"}},
		Workspace:   "/workspace",
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("failed to marshal ResolvedSpec: %v", err)
	}

	var decoded ResolvedSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ResolvedSpec: %v", err)
	}

	// Check critical fields
	if decoded.Name != spec.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, spec.Name)
	}
	if decoded.RuntimeKind != spec.RuntimeKind {
		t.Errorf("RuntimeKind mismatch: got %q, want %q", decoded.RuntimeKind, spec.RuntimeKind)
	}
	if decoded.Sandbox.Scope != spec.Sandbox.Scope {
		t.Errorf("Sandbox.Scope mismatch: got %q, want %q", decoded.Sandbox.Scope, spec.Sandbox.Scope)
	}
	if decoded.Sandbox.TTL != spec.Sandbox.TTL {
		t.Errorf("Sandbox.TTL mismatch: got %q, want %q", decoded.Sandbox.TTL, spec.Sandbox.TTL)
	}
}
