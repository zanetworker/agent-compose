package compose

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type RuntimeResolver interface {
	Resolve(ctx context.Context, name string) (*RuntimeProfile, error)
	List(ctx context.Context) ([]RuntimeProfile, error)
}

type ConfigRuntimeResolver struct {
	config *Config
}

func NewConfigRuntimeResolver(cfg *Config) *ConfigRuntimeResolver {
	return &ConfigRuntimeResolver{config: cfg}
}

func (r *ConfigRuntimeResolver) Resolve(_ context.Context, name string) (*RuntimeProfile, error) {
	profile, ok := r.config.Runtimes[name]
	if !ok {
		return nil, fmt.Errorf("runtime %q: %w", name, ErrNotFound)
	}
	profile.Name = name
	return &profile, nil
}

func (r *ConfigRuntimeResolver) List(_ context.Context) ([]RuntimeProfile, error) {
	profiles := make([]RuntimeProfile, 0, len(r.config.Runtimes))
	for name, p := range r.config.Runtimes {
		p.Name = name
		profiles = append(profiles, p)
	}
	return profiles, nil
}
