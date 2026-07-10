package compose

import (
	"context"
	"errors"
	"testing"
)

func TestConfigHarnessResolver_Resolve_BuiltIn(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigHarnessResolver(cfg)

	profile, err := r.Resolve(context.Background(), "claude-code")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if profile.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("image = %q, want ghcr.io/anthropics/claude-code:latest", profile.Image)
	}
	if profile.EnvMapping["ANTHROPIC_BASE_URL"] != "${endpoint}" {
		t.Errorf("env-mapping[ANTHROPIC_BASE_URL] = %q, want ${endpoint}", profile.EnvMapping["ANTHROPIC_BASE_URL"])
	}
}

func TestConfigHarnessResolver_Resolve_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigHarnessResolver(cfg)

	_, err := r.Resolve(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestConfigHarnessResolver_List(t *testing.T) {
	cfg := DefaultConfig()
	r := NewConfigHarnessResolver(cfg)

	profiles, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(profiles) != 4 {
		t.Errorf("len = %d, want 4", len(profiles))
	}
}
