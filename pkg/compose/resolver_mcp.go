package compose

import (
	"context"
	"fmt"
)

type MCPResolver interface {
	Resolve(ctx context.Context, name string) (*MCPSpec, error)
	List(ctx context.Context) ([]MCPSpec, error)
}

type ConfigMCPResolver struct {
	config *Config
}

func NewConfigMCPResolver(cfg *Config) *ConfigMCPResolver {
	return &ConfigMCPResolver{config: cfg}
}

func (r *ConfigMCPResolver) Resolve(_ context.Context, name string) (*MCPSpec, error) {
	spec, ok := r.config.MCP[name]
	if !ok {
		return nil, fmt.Errorf("mcp %q: %w", name, ErrNotFound)
	}
	spec.Name = name
	return &spec, nil
}

func (r *ConfigMCPResolver) List(_ context.Context) ([]MCPSpec, error) {
	specs := make([]MCPSpec, 0, len(r.config.MCP))
	for name, s := range r.config.MCP {
		s.Name = name
		specs = append(specs, s)
	}
	return specs, nil
}
