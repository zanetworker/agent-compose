# agent-compose

![agent-compose](docs/logo.png)

Declare what an AI agent needs. One command resolves it into a running, governed sandbox.

```bash
ac run security-reviewer --workspace ./repo
```

The engine picks the runtime, attaches inference credentials, connects MCP servers, injects skill prompts, creates the sandbox, and cleans up when done. You configure once, run everywhere.

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

```go
import "github.com/zanetworker/agent-compose/pkg/compose"

cfg, _ := compose.LoadConfig("config.yaml")
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell", os.Stdin, os.Stdout, os.Stderr)),
    compose.WithProgress(os.Stderr),
)

run, _ := engine.Run(ctx, "security-reviewer", compose.RunOpts{
    Workspace: "./my-repo",
})

// Or start in background
run, _ := engine.Start(ctx, "security-reviewer", compose.RunOpts{
    Prompt: "Review the auth module",
})
```

## Documentation

| Doc | What |
|-----|------|
| [Tutorial](docs/tutorial.md) | Step-by-step walkthrough |
| [Composition Guide](docs/composition.md) | Full config reference |
| [Architecture](docs/architecture.md) | Engine design, resolvers, executor interface |
| [Status](docs/status-and-next.md) | What's done, what's next |
