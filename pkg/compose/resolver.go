package compose

import (
	"context"
	"fmt"
	"strings"
)

type Resolver struct {
	runtimes  RuntimeResolver
	inference InferenceResolver
	mcp       MCPResolver
	skills    SkillResolver
	policy    PolicyResolver
	defaults  Defaults
}

func NewResolver(
	runtimes RuntimeResolver,
	inference InferenceResolver,
	mcp MCPResolver,
	skills SkillResolver,
	policy PolicyResolver,
	defaults Defaults,
) *Resolver {
	return &Resolver{
		runtimes:  runtimes,
		inference: inference,
		mcp:       mcp,
		skills:    skills,
		policy:    policy,
		defaults:  defaults,
	}
}

func (r *Resolver) Resolve(ctx context.Context, agent Agent) (*ResolvedSpec, error) {
	spec := &ResolvedSpec{
		Name:   agent.Name,
		Env:    make(map[string]string),
		Labels: map[string]string{"agent": agent.Name},
	}

	// Apply defaults
	if agent.Inference == "" {
		agent.Inference = r.defaults.Inference
	}
	if agent.Policy == "" {
		agent.Policy = r.defaults.Policy
	}

	// Resolve runtime or use direct image
	var envMapping map[string]string
	if agent.Runtime != "" {
		profile, err := r.runtimes.Resolve(ctx, agent.Runtime)
		if err != nil {
			return nil, fmt.Errorf("resolving runtime: %w", err)
		}
		spec.Image = profile.Image
		spec.Entrypoint = profile.Entrypoint
		spec.Tools = profile.Tools
		spec.RuntimeKind = profile.Kind
		envMapping = profile.EnvMapping
	} else if agent.Image != "" {
		spec.Image = agent.Image
		spec.Entrypoint = agent.Entrypoint
		spec.RuntimeKind = "raw"
		if agent.EnvMapping != nil {
			envMapping = agent.EnvMapping
		}
	} else {
		return nil, fmt.Errorf("agent %q: must specify runtime or image", agent.Name)
	}

	// Resolve inference
	if agent.Inference != "" {
		infSpec, err := r.inference.Resolve(ctx, agent.Inference)
		if err != nil {
			return nil, fmt.Errorf("resolving inference: %w", err)
		}
		// TODO(Task 5): Use ExpandEnvMapping to populate spec.Env from N-var template map
		// For now, stub this out. The old 3-field EnvMapping logic is removed.
		// Task 5 will wire the N-var expansion function here with bindings:
		// {"endpoint": infSpec.Endpoint, "key": "<from policy>", "model": model}
		_ = envMapping
		_ = infSpec

		model := infSpec.DefaultModel
		if agent.Model != "" {
			model = agent.Model
		}
		_ = model

		spec.Providers = appendUnique(spec.Providers, infSpec.Provider)
		spec.Egress = appendUnique(spec.Egress, infSpec.Egress...)
	}

	// Collect MCP from agent + skill dependencies
	mcpNames := make([]string, len(agent.MCP))
	copy(mcpNames, agent.MCP)

	// Resolve skills (may add MCP dependencies)
	var skillPrompts []string
	for _, skillName := range agent.Skills {
		skill, err := r.skills.Resolve(ctx, skillName)
		if err != nil {
			return nil, fmt.Errorf("resolving skill %q: %w", skillName, err)
		}
		if skill.Prompt != "" {
			skillPrompts = append(skillPrompts, skill.Prompt)
		}
		mcpNames = appendUnique(mcpNames, skill.RequiresMCP...)
		spec.Tools = appendUnique(spec.Tools, skill.RequiresTools...)
		for _, ref := range skill.References {
			spec.SkillMounts = append(spec.SkillMounts, Mount{
				Source: ref,
				Target: fmt.Sprintf("/workspace/skills/%s/", skillName),
			})
		}
	}

	// Resolve each MCP
	for _, mcpName := range mcpNames {
		mcpSpec, err := r.mcp.Resolve(ctx, mcpName)
		if err != nil {
			return nil, fmt.Errorf("resolving mcp %q: %w", mcpName, err)
		}
		if mcpSpec.Provider != "" {
			spec.Providers = appendUnique(spec.Providers, mcpSpec.Provider)
		}
		spec.Egress = appendUnique(spec.Egress, mcpSpec.Egress...)
	}

	// Assemble prompt
	parts := []string{}
	if agent.Prompt != "" {
		parts = append(parts, agent.Prompt)
	}
	parts = append(parts, skillPrompts...)
	spec.Prompt = strings.Join(parts, "\n\n")

	// Resolve policy
	if agent.Policy != "" {
		pol, err := r.policy.Resolve(ctx, agent.Policy)
		if err != nil {
			return nil, fmt.Errorf("resolving policy: %w", err)
		}
		spec.Policy = pol.Name
	}

	// Apply agent env overrides (cannot override system env)
	for k, v := range agent.Env {
		if _, exists := spec.Env[k]; !exists {
			spec.Env[k] = v
		}
	}

	// Apply agent tool overrides
	if len(agent.Tools) > 0 {
		spec.Tools = agent.Tools
	}

	spec.Workspace = agent.Workspace

	return spec, nil
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			slice = append(slice, item)
		}
	}
	return slice
}
