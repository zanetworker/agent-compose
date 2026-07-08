package compose

import (
	"context"
	"fmt"
)

type PolicyResolver interface {
	Resolve(ctx context.Context, name string) (*Policy, error)
	List(ctx context.Context) ([]Policy, error)
}

type ConfigPolicyResolver struct{}

func NewConfigPolicyResolver() *ConfigPolicyResolver {
	return &ConfigPolicyResolver{}
}

func (r *ConfigPolicyResolver) Resolve(_ context.Context, name string) (*Policy, error) {
	if name == "" {
		return nil, fmt.Errorf("policy: empty name: %w", ErrNotFound)
	}
	return &Policy{Name: name}, nil
}

func (r *ConfigPolicyResolver) List(_ context.Context) ([]Policy, error) {
	return nil, nil
}
