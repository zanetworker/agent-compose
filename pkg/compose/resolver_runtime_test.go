package compose

import "testing"

func TestConfigRuntimeResolver_Resolve_BuiltIn(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigRuntimeResolver(cfg)
	profile, err := r.Resolve(nil, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Kind != "harness" {
		t.Errorf("expected kind=harness, got %q", profile.Kind)
	}
	if profile.Image == "" {
		t.Error("expected non-empty image")
	}
	if _, ok := profile.EnvMapping["ANTHROPIC_BASE_URL"]; !ok {
		t.Error("expected ANTHROPIC_BASE_URL in N-var env-mapping")
	}
}

func TestConfigRuntimeResolver_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigRuntimeResolver(cfg)
	_, err := r.Resolve(nil, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown runtime")
	}
}

func TestConfigRuntimeResolver_List(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigRuntimeResolver(cfg)
	profiles, err := r.List(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) < 2 {
		t.Errorf("expected at least 2 built-in runtimes, got %d", len(profiles))
	}
}
