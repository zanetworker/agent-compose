# agent-compose

![agent-compose](docs/agent-compose-flow.svg)

Declare what an AI agent needs. One command resolves it into a running, governed sandbox.

```bash
ac run security-reviewer --workspace ./repo
```

## Why

- **One command from zero to running agent.** `ac run` handles sandbox creation, credentials, egress policy, execution, and cleanup. One command.
- **Developers think about agents, not infrastructure.** No containers to configure, no policies to write, no credentials to manage. Declare what the agent needs; the engine handles the rest.
- **Every agent runs in a sandbox. No exceptions.** Network access is deny-by-default. Only declared endpoints are reachable. Credentials are injected at the network layer, never visible to agent code.
- **Spin up, tear down, no residue.** Sandboxes auto-delete when agents finish. No orphaned compute, no leaked credentials, no cleanup scripts.
- **Adding a tool or changing a model is a config change, not an infrastructure project.** Swap the model, add an MCP server, change the runtime. The sandbox adapts automatically.
- **The same engine powers CLIs, dashboards, and pipelines.** Import as a Go library. Preview agent composition in a UI, fan out reviews across repos in CI, or reconcile agent CRDs in a controller. Same API everywhere.

## Quick Start

```bash
# Build
make build

# Initialize config + auto-detect credentials
ac init

# Run an agent (blocks, streams output, auto-cleans up)
ac run quick-review --prompt "What is 2+2?"

# Start in background, check later
ac start security-reviewer --workspace ./repo
ac logs <sandbox-name>
ac attach <sandbox-name>
ac stop <sandbox-name>

# Interactive session
ac run quick-review -i

# Preview without executing
ac run security-reviewer --dry-run
ac get security-reviewer --json
```

## How It Works

An agent is a composition of five layers:

```yaml
# ~/.ac/config.yaml
agents:
  security-reviewer:
    runtime: claude-code-vertex        # image, entrypoint, providers
    inference: vertex                  # model endpoint, credentials
    mcp: [github]                      # tools (auto-generates agent-native config)
    skills: [security-review]          # prompt + reference files
    prompt: "Review code for vulnerabilities."
```

`ac run` resolves all layers, generates the right config files for the agent framework, and runs it in an OpenShell sandbox:

```
config.yaml ──resolve──> ResolvedSpec ──execute──> openshell sandbox
                              │
        ┌─────────────────────┼────────────────────┐
        │                     │                    │
   env vars              MCP config           egress policy
   (inference)      (agent-native JSON/YAML)  (per-endpoint)
```

MCP servers are declared once and translated to each agent's native format. Claude gets JSON at `.ac-mcp.json`, Goose gets YAML at `.config/goose/config.yaml`. The runtime profile is the translation layer.

## Commands

```
ac init                     Create config + auto-detect credentials
ac run <name> [flags]       Run agent (blocks, streams output, auto-cleanup)
ac start <name> [flags]     Start agent in background
ac stop <name>              Stop agent + delete sandbox
ac attach <name>            Shell into running sandbox
ac list                     List running agents
ac logs <name>              Show agent output (--system for gateway logs)
ac get <name>               Show resolved spec as JSON
ac doctor                   Validate config + environment
ac apply --sync-profiles    Push provider profiles to gateway
```

**Run/Start flags:** `--prompt`, `--workspace`, `--runtime`, `--inference`, `--model`, `--mcp`, `--skills`, `--skip-permissions`, `-i` (interactive)

**Global flags:** `--dry-run`, `--json`, `--config`, `--skills-dir`

## Prerequisites

- **OpenShell 0.0.83+** with gateway running (`brew install nvidia/openshell/openshell && brew services start openshell`)
- **Go 1.24+** to build
- At least one credential: `gcloud auth application-default login` (Vertex), `gh auth login` (GitHub), or `ANTHROPIC_API_KEY` (direct API)

## Testing It Yourself

```bash
# 1. Build
make build

# 2. Initialize (creates ~/.ac/config.yaml + providers)
ac init
ac doctor

# 3. Basic test (no MCP, just inference)
ac run quick-review --prompt "What is 2+2? Reply with just the number."
# Should print: 4

# 4. Test with remote MCP server
# Start the MCP reference test server:
npx -y @modelcontextprotocol/server-everything streamableHttp &

# Add it to ~/.ac/config.yaml under mcp:
#   everything:
#     type: http
#     url: http://host.containers.internal:3001/mcp
#     egress: [host.containers.internal:3001]
#
# Add mcp: [everything] to your agent definition, then:
ac run quick-review --prompt "Use the everything MCP echo tool to echo 'hello'. Print the response."
# Should print: Echo: hello

# 5. Test lifecycle
ac start quick-review --prompt "Explain Go interfaces in one sentence."
ac logs <sandbox-name>    # read agent output
ac attach <sandbox-name>  # shell in
ac stop <sandbox-name>    # clean up
ac list                   # should be empty

# 6. Test interactive
ac run quick-review -i
# Drops into Claude. Type /exit to leave. Sandbox auto-deletes.

# 7. Dry-run (no gateway needed)
ac run security-reviewer --workspace ./repo --dry-run
# Prints exact openshell commands without executing
```

## Configuration

See [docs/composition.md](docs/composition.md) for the full config reference. Key sections:

```yaml
runtimes:       # How to run agents (image, entrypoint, env-mapping, binaries, mcp-config)
inference:      # Where models live (endpoint, provider, default-model)
mcp:            # Tool servers (type, command/url, env, provider, egress)
agents:         # Named compositions (runtime + inference + mcp + skills + prompt)
defaults:       # Fallback inference, policy, sandbox opts
```

## Go SDK

agent-compose is library-first. The `pkg/compose` package embeds into dashboards, CI pipelines, platform controllers, or any Go service that needs to compose and run agents.

```go
import "github.com/zanetworker/agent-compose/pkg/compose"

cfg, _ := compose.LoadConfig("config.yaml")
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell", os.Stdin, os.Stdout, os.Stderr)),
    compose.WithProgress(os.Stderr),
)
```

**Preview before launch** (dashboard pattern):

```go
// Resolve shows the full spec without creating anything
spec, _ := engine.Resolve(ctx, "security-reviewer")
// spec.Image, spec.Providers, spec.Env, spec.MCPServers, spec.Prompt
// Display in UI, let user confirm, then:
engine.Run(ctx, "security-reviewer", compose.RunOpts{Workspace: "./repo"})
```

**Fan out across repos** (CI pattern):

```go
repos := []string{"./api", "./web", "./auth"}
for _, repo := range repos {
    run, _ := engine.Start(ctx, "reviewer", compose.RunOpts{
        Workspace: repo,
        Prompt:    "Review for security issues",
    })
    fmt.Printf("Started %s in %s\n", repo, run.Sandbox)
}
// Later: collect results
for _, run := range runs {
    output, _ := engine.AgentOutput(ctx, run.Sandbox)
    engine.Stop(ctx, run.Sandbox)
}
```

**Custom runtime** (bring your own agent):

```go
cfg.Runtimes["my-agent"] = compose.RuntimeProfile{
    Kind:       "framework",
    Image:      "my-registry.com/my-agent:latest",
    EnvMapping: map[string]string{"LLM_ENDPOINT": "${endpoint}", "LLM_MODEL": "${model}"},
    Entrypoint: []string{"python3", "/app/agent.py"},
}
engine.Run(ctx, "my-agent", compose.RunOpts{Workspace: "./data"})
```

**Compose with MCP** (tools for agents):

```go
cfg.MCP["sentry"] = compose.MCPSpec{
    Type: "http", URL: "https://mcp.sentry.dev/mcp",
    Egress: []string{"mcp.sentry.dev:443"},
}
cfg.Agents["debugger"] = compose.Agent{
    Runtime: "claude-code", MCP: []string{"github", "sentry"},
    Prompt:  "Debug this issue using GitHub and Sentry.",
}
// MCP config is auto-generated in the agent's native format
```

**Run the examples** (no gateway needed, uses DryRunExecutor):

```bash
go test ./examples/ -v
```

See [examples/](examples/) for all runnable use cases.

## Documentation

| Doc | What |
|-----|------|
| [Tutorial](docs/tutorial.md) | Step-by-step walkthrough |
| [Composition Guide](docs/composition.md) | Full config reference |
| [Architecture](docs/architecture.md) | Engine design, resolvers, executor interface |
| [Status](docs/status-and-next.md) | What's done, what's next |
