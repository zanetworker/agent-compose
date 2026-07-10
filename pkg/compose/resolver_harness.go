package compose

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type HarnessResolver interface {
	Resolve(ctx context.Context, name string) (*RuntimeProfile, error)
	List(ctx context.Context) ([]RuntimeProfile, error)
}

type ConfigHarnessResolver struct {
	config *Config
}

func NewConfigHarnessResolver(cfg *Config) *ConfigHarnessResolver {
	return &ConfigHarnessResolver{config: cfg}
}

func (r *ConfigHarnessResolver) Resolve(_ context.Context, name string) (*RuntimeProfile, error) {
	profile, ok := r.config.Runtimes[name]
	if !ok {
		return nil, fmt.Errorf("harness %q: %w", name, ErrNotFound)
	}
	profile.Name = name
	return &profile, nil
}

func (r *ConfigHarnessResolver) List(_ context.Context) ([]RuntimeProfile, error) {
	profiles := make([]RuntimeProfile, 0, len(r.config.Runtimes))
	for name, p := range r.config.Runtimes {
		p.Name = name
		profiles = append(profiles, p)
	}
	return profiles, nil
}
