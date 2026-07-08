package compose

import (
	"context"
	"fmt"
)

type InferenceResolver interface {
	Resolve(ctx context.Context, name string) (*InferenceSpec, error)
	List(ctx context.Context) ([]InferenceSpec, error)
}

type ConfigInferenceResolver struct {
	config *Config
}

func NewConfigInferenceResolver(cfg *Config) *ConfigInferenceResolver {
	return &ConfigInferenceResolver{config: cfg}
}

func (r *ConfigInferenceResolver) Resolve(_ context.Context, name string) (*InferenceSpec, error) {
	spec, ok := r.config.Inference[name]
	if !ok {
		return nil, fmt.Errorf("inference %q: %w", name, ErrNotFound)
	}
	spec.Name = name
	return &spec, nil
}

func (r *ConfigInferenceResolver) List(_ context.Context) ([]InferenceSpec, error) {
	specs := make([]InferenceSpec, 0, len(r.config.Inference))
	for name, s := range r.config.Inference {
		s.Name = name
		specs = append(specs, s)
	}
	return specs, nil
}
