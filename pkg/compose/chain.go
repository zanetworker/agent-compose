package compose

import (
	"context"
	"errors"
	"fmt"
)

type ChainedHarnessResolver struct {
	resolvers []HarnessResolver
}

func NewChainedHarnessResolver(resolvers ...HarnessResolver) *ChainedHarnessResolver {
	return &ChainedHarnessResolver{resolvers: resolvers}
}

func (c *ChainedHarnessResolver) Resolve(ctx context.Context, name string) (*RuntimeProfile, error) {
	for _, r := range c.resolvers {
		result, err := r.Resolve(ctx, name)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("harness %q: %w in any resolver", name, ErrNotFound)
}

func (c *ChainedHarnessResolver) List(ctx context.Context) ([]RuntimeProfile, error) {
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
