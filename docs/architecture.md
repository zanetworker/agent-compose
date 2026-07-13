# Architecture

The core is a Go library (`pkg/compose/`). CLI and API are thin wrappers.

![Architecture Diagram](architecture.png)

## Engine

The `Engine` struct orchestrates resolution and execution:

```go
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell")),
    compose.WithSkillsDir("~/.ac/skills"),
)

spec, _ := engine.Resolve(ctx, "code-reviewer")   // resolve only (P2/P3)
run, _ := engine.Run(ctx, "code-reviewer", opts)   // resolve + create + exec (P1)
engine.List(ctx)                                    // list running sandboxes
engine.Stop(ctx, "code-reviewer-1234")              // delete sandbox
engine.Get(ctx, "code-reviewer")                    // resolved spec as JSON
engine.SyncProfiles(ctx)                            // push profiles to gateway
```

## Resolvers

Each asset type has its own resolver interface. V1 ships config-file implementations. Future versions add catalog-backed implementations (KServe, MCP Gateway, OCI registry).

```
RuntimeResolver    -> image, entrypoint, env-mapping, providers
InferenceResolver  -> endpoint, provider, model, tiers, egress
MCPResolver        -> provider, egress
SkillResolver      -> prompt, required MCP, required tools, references
PolicyResolver     -> policy file
```

Resolution chain: local config (user overrides) -> catalog API (platform) -> error.

## Executor

Pluggable interface for sandbox operations:

| Implementation | When to use |
|---|---|
| CLIExecutor | Shells out to `openshell` binary (v1) |
| SDKExecutor | OpenShell Go SDK (future) |
| DryRunExecutor | Prints commands without executing |

## No Local State

OpenShell sandbox labels (`agentctl.io/agent`) are the source of truth for run state. `list`/`stop`/`logs` query the executor, never a local database.

## Go SDK

Everything the CLI does is available programmatically. See `examples/sdk_test.go` for usage:

```go
// Resolve a named agent
spec, _ := engine.Resolve(ctx, "code-reviewer")

// Override inference and model per run
run, _ := engine.Run(ctx, "code-reviewer", compose.RunOpts{
    Inference: "local-vllm",
    Model:     "llama-3.3-70b",
})

// Compose inline (no config entry needed)
run, _ = engine.Run(ctx, "", compose.RunOpts{
    Agent: &compose.Agent{
        Runtime: "claude-code",
        MCP:     []string{"github"},
        Prompt:  "Review this code",
    },
})
```

## Sandbox Lifecycle

Configurable via defaults or per-agent:

```yaml
defaults:
  sandbox:
    scope: session    # session | agent | shared
    mode: all         # all | non-main | off
    ttl: 30m          # idle timeout before reaping
```
