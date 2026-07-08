package compose

import (
	"context"
	"errors"
	"testing"
)

func TestConfigPolicyResolver_Resolve(t *testing.T) {
	r := NewConfigPolicyResolver()

	policy, err := r.Resolve(context.Background(), "restricted")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if policy.Name != "restricted" {
		t.Errorf("name = %q, want restricted", policy.Name)
	}
}

func TestConfigPolicyResolver_EmptyName(t *testing.T) {
	r := NewConfigPolicyResolver()

	_, err := r.Resolve(context.Background(), "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestConfigPolicyResolver_List(t *testing.T) {
	r := NewConfigPolicyResolver()

	policies, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if policies != nil {
		t.Errorf("policies = %v, want nil", policies)
	}
}
