package compose

import (
	"context"
	"errors"
	"testing"
)

func TestChainedRuntimeResolver_FallsThrough(t *testing.T) {
	empty := NewConfigRuntimeResolver(&Config{Runtimes: map[string]RuntimeProfile{}})
	withData := NewConfigRuntimeResolver(DefaultConfig())

	chain := NewChainedRuntimeResolver(empty, withData)

	profile, err := chain.Resolve(context.Background(), "claude-code")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if profile.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("image = %q, want ghcr.io/anthropics/claude-code:latest", profile.Image)
	}
}

func TestChainedRuntimeResolver_FirstWins(t *testing.T) {
	override := NewConfigRuntimeResolver(&Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {Kind: "harness", Image: "custom-image:v1"},
		},
	})
	defaults := NewConfigRuntimeResolver(DefaultConfig())

	chain := NewChainedRuntimeResolver(override, defaults)

	profile, err := chain.Resolve(context.Background(), "claude-code")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if profile.Image != "custom-image:v1" {
		t.Errorf("image = %q, want custom-image:v1 (override should win)", profile.Image)
	}
}

func TestChainedRuntimeResolver_AllMiss(t *testing.T) {
	empty1 := NewConfigRuntimeResolver(&Config{Runtimes: map[string]RuntimeProfile{}})
	empty2 := NewConfigRuntimeResolver(&Config{Runtimes: map[string]RuntimeProfile{}})

	chain := NewChainedRuntimeResolver(empty1, empty2)

	_, err := chain.Resolve(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
