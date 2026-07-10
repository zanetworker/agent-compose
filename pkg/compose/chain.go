package compose

import (
	"context"
	"errors"
	"fmt"
)

type ChainedRuntimeResolver struct {
	resolvers []RuntimeResolver
}

func NewChainedRuntimeResolver(resolvers ...RuntimeResolver) *ChainedRuntimeResolver {
	return &ChainedRuntimeResolver{resolvers: resolvers}
}

func (c *ChainedRuntimeResolver) Resolve(ctx context.Context, name string) (*RuntimeProfile, error) {
	for _, r := range c.resolvers {
		result, err := r.Resolve(ctx, name)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("runtime %q: %w in any resolver", name, ErrNotFound)
}

func (c *ChainedRuntimeResolver) List(ctx context.Context) ([]RuntimeProfile, error) {
	seen := make(map[string]bool)
	var all []RuntimeProfile
	for _, r := range c.resolvers {
		items, err := r.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !seen[item.Name] {
				seen[item.Name] = true
				all = append(all, item)
			}
		}
	}
	return all, nil
}
