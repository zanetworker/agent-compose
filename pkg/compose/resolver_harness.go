package compose

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type HarnessResolver interface {
	Resolve(ctx context.Context, name string) (*HarnessProfile, error)
	List(ctx context.Context) ([]HarnessProfile, error)
}

type ConfigHarnessResolver struct {
	config *Config
}

func NewConfigHarnessResolver(cfg *Config) *ConfigHarnessResolver {
	return &ConfigHarnessResolver{config: cfg}
}

func (r *ConfigHarnessResolver) Resolve(_ context.Context, name string) (*HarnessProfile, error) {
	profile, ok := r.config.Harnesses[name]
	if !ok {
		return nil, fmt.Errorf("harness %q: %w", name, ErrNotFound)
	}
	profile.Name = name
	return &profile, nil
}

func (r *ConfigHarnessResolver) List(_ context.Context) ([]HarnessProfile, error) {
	profiles := make([]HarnessProfile, 0, len(r.config.Harnesses))
	for name, p := range r.config.Harnesses {
		p.Name = name
		profiles = append(profiles, p)
	}
	return profiles, nil
}
