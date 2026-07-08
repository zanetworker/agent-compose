package compose

import (
	"context"
	"errors"
	"testing"
)

func TestChainedHarnessResolver_FallsThrough(t *testing.T) {
	empty := NewConfigHarnessResolver(&Config{Harnesses: map[string]HarnessProfile{}})
	withData := NewConfigHarnessResolver(DefaultConfig())

	chain := NewChainedHarnessResolver(empty, withData)

	profile, err := chain.Resolve(context.Background(), "claude-code")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if profile.Image != "ghcr.io/anthropics/claude-code:latest" {
		t.Errorf("image = %q, want ghcr.io/anthropics/claude-code:latest", profile.Image)
	}
}

func TestChainedHarnessResolver_FirstWins(t *testing.T) {
	override := NewConfigHarnessResolver(&Config{
		Harnesses: map[string]HarnessProfile{
			"claude-code": {Image: "custom-image:v1"},
		},
	})
	defaults := NewConfigHarnessResolver(DefaultConfig())

	chain := NewChainedHarnessResolver(override, defaults)

	profile, err := chain.Resolve(context.Background(), "claude-code")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if profile.Image != "custom-image:v1" {
		t.Errorf("image = %q, want custom-image:v1 (override should win)", profile.Image)
	}
}

func TestChainedHarnessResolver_AllMiss(t *testing.T) {
	empty1 := NewConfigHarnessResolver(&Config{Harnesses: map[string]HarnessProfile{}})
	empty2 := NewConfigHarnessResolver(&Config{Harnesses: map[string]HarnessProfile{}})

	chain := NewChainedHarnessResolver(empty1, empty2)

	_, err := chain.Resolve(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
