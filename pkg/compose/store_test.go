package compose

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStore_SaveAndGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &Run{
		ID:        "test-sandbox-123",
		Agent:     "reviewer",
		Sandbox:   "reviewer-123",
		StartedAt: time.Now().Unix(),
		Status:    SandboxRunning,
	}

	if err := store.Save(ctx, run); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Get(ctx, "reviewer")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Agent != "reviewer" {
		t.Errorf("agent = %q, want reviewer", got.Agent)
	}
	if got.Sandbox != "reviewer-123" {
		t.Errorf("sandbox = %q, want reviewer-123", got.Sandbox)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run1 := &Run{Agent: "agent1", Sandbox: "sandbox1", Status: SandboxRunning}
	run2 := &Run{Agent: "agent2", Sandbox: "sandbox2", Status: SandboxRunning}

	store.Save(ctx, run1)
	store.Save(ctx, run2)

	runs, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("len(runs) = %d, want 2", len(runs))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &Run{Agent: "test", Sandbox: "sandbox"}
	store.Save(ctx, run)

	if err := store.Delete(ctx, "test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(ctx, "test")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, Get should return ErrNotFound, got %v", err)
	}
}
