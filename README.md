# agent-compose

Agent composition engine for OpenShell. Resolves declarative agent configs into sandbox commands. One declaration replaces 8 manual steps.

## The Problem

Running an agent in an OpenShell sandbox requires 8+ manual steps: choosing the right image, creating providers for credentials, figuring out framework-specific env vars (`ANTHROPIC_BASE_URL` vs `OPENAI_BASE_URL` vs `GOOGLE_GENAI_BASE_URL`), setting those vars, configuring egress, assembling prompts. Every framework uses different names for the same concepts.

## The Solution

A composition engine that connects catalogs (what's available) to sandboxes (where agents run). You declare what the agent needs; the engine resolves it into raw `openshell` commands.

```
Catalogs (config.yaml)     Agent Config (what I want)     Engine (resolve)     OpenShell (run)
models, MCP, skills        runtime + inference + mcp      ResolvedSpec         sandbox create
```

## Quick Start

```bash
# Build
make build

# Initialize config
./ac init
# Edit ~/.agentctl/config.yaml with your inference providers and MCP servers

# Run an agent with inline flags (zero config needed)
./ac run --runtime claude-code --inference maas --prompt "Review this code" --dry-run

# Run a named agent from config
./ac run code-reviewer --workspace ./my-project --dry-run

# Show fully resolved spec as JSON
./ac get code-reviewer --json

# List running agents
./ac list

# Stop an agent
./ac stop code-reviewer
```

## What the Engine Produces

```bash
$ ./ac run --runtime claude-code --prompt "test" --dry-run

openshell sandbox create --name inline-1234 \
  --image ghcr.io/anthropics/claude-code:latest \
  --provider maas-anthropic \
  --env ANTHROPIC_BASE_URL=https://maas.apps.cluster.com/v1 \
  --env ANTHROPIC_DEFAULT_SONNET_MODEL=granite-3.3-8b-instruct \
  --policy restricted \
  --scope session --mode all --ttl 30m \
  --label agentctl.io/agent=inline-1234

openshell sandbox exec inline-1234 -- claude --prompt-file /workspace/prompt.md
```

All 8 manual steps collapsed into one command. No new OpenShell primitives.

## Config

One file (`~/.agentctl/config.yaml`). Three friction tiers:

**Zero files (CLI flags):**
```bash
ac run --runtime claude-code --inference maas --mcp github --prompt "Review this"
```

**Named agent (config entry):**
```yaml
# ~/.agentctl/config.yaml
agents:
  code-reviewer:
    runtime: claude-code
    inference: maas
    mcp: [github]
    skills: [security-review]
    prompt: "Review code for security issues."
```
```bash
ac run code-reviewer
```

**Separate files (GitOps):**
```
my-agents-repo/
+-- config.yaml
+-- agents/
+-- skills/
```

## Agent Types

One field, `runtime:`, discriminates three kinds:

| `runtime.kind` | Declaration | Examples |
|---|---|---|
| **harness** | `runtime: claude-code` | Claude Code, Codex, Goose |
| **framework** | `image:` + `env-mapping:` | ADK, LangGraph, CrewAI |
| **raw** | `image:` + `entrypoint:` | Any container |

## N-var Env-Mapping

Runtime profiles use template maps (not fixed slots) to handle any framework's env var conventions, including multi-tier models and auth toggles:

```yaml
runtimes:
  claude-code:
    kind: harness
    image: ghcr.io/anthropics/claude-code@sha256:...
    env-mapping:
      ANTHROPIC_BASE_URL:             "${endpoint}"
      ANTHROPIC_API_KEY:              "${key}"
      ANTHROPIC_DEFAULT_SONNET_MODEL: "${model}"
      ANTHROPIC_DEFAULT_OPUS_MODEL:   "${model.opus}"
      ANTHROPIC_DEFAULT_HAIKU_MODEL:  "${model.haiku}"
```

Inference providers define the values and optional tier overrides:

```yaml
inference:
  maas:
    endpoint: https://maas.apps.cluster.com/v1
    provider: maas-anthropic
    default-model: granite-3.3-8b-instruct
    models:
      opus: granite-3.3-8b-instruct
      haiku: granite-3.3-2b-instruct
```

Override at run time with `--inference` and `--model`:

```bash
ac run code-reviewer --inference vertex --model gemini-2.5-pro
```

## Skills

Reusable bundles of prompt instructions + tool/MCP requirements:

```
~/.agentctl/skills/security-review/
+-- SKILL.md           # prompt (appended to agent's prompt)
+-- references/        # mounted at /workspace/skills/<name>/
    +-- owasp-top-10.md
```

SKILL.md can declare dependencies:
```markdown
---
requires:
  mcp: [github]
  tools: [shell, file-read]
---

# Security Review
When reviewing code, check for SQL injection, XSS, auth bypass...
```

## Architecture

The core is a Go library (`pkg/compose/`). CLI is a thin cobra wrapper.

```
+----------+   +----------+   +----------+
|   CLI    |   |   API    |   |   SDK    |
|   (ac)   |   |  (v2)    |   | (Go pkg) |
+----+-----+   +----+-----+   +----+-----+
     +--------+-----+---------+
              v
        Core Engine
    Resolvers -> ResolvedSpec
              v
     Executor (pluggable)
       |            |
  CLIExecutor   SDKExecutor (future)
  (openshell)   (OpenShell Go SDK)
```

**Resolver interfaces** are pluggable. V1 reads from config.yaml. Future versions discover models from KServe, MCP servers from MCP Gateway, skills from OCI registries.

**Executor interface** is pluggable. V1 shells out to the `openshell` binary. When the OpenShell SDK ships, swap in `SDKExecutor` with one line.

**No local run database.** OpenShell sandbox labels (`agentctl.io/agent`) are the source of truth for run state. `list`/`stop`/`logs` query the executor, never a local store.

```go
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell")),
    compose.WithSkillsDir("~/.agentctl/skills"),
)

// Resolve only (for harnesses that create their own sandboxes)
spec, _ := engine.Resolve(ctx, "code-reviewer", compose.ResolveOpts{})

// Or resolve + create + run
run, _ := engine.Run(ctx, "code-reviewer", compose.RunOpts{Workspace: "./repo"})
```

## Sandbox Lifecycle

Sandboxes have scope, mode, and TTL:

```yaml
defaults:
  sandbox:
    scope: session    # session | agent | shared
    mode: all         # all | non-main | off
    ttl: 30m          # idle timeout before reaping
```

## Commands

```
ac init                 Create ~/.agentctl/ with default config
ac run <name> [flags]   Resolve + create sandbox + start agent
ac stop <name>          Stop agent + delete sandbox
ac list                 List running agents
ac get <name>           Show fully resolved spec as JSON
ac logs <name>          Stream sandbox output
```

Global flags: `--json`, `--dry-run`, `--config <path>`, `--skills-dir <path>`

## Built-in Runtime Profiles

| Runtime | Kind | Image | Key Env Vars |
|---|---|---|---|
| claude-code | harness | ghcr.io/anthropics/claude-code | ANTHROPIC_BASE_URL, ANTHROPIC_API_KEY, ANTHROPIC_DEFAULT_SONNET_MODEL |
| codex | harness | ghcr.io/openai/codex | OPENAI_BASE_URL, OPENAI_API_KEY, OPENAI_MODEL |
| goose | harness | ghcr.io/block/goose | OPENAI_BASE_URL, OPENAI_API_KEY, GOOSE_MODEL |
| adk | framework | python:3.12-slim | GOOGLE_GENAI_BASE_URL, GOOGLE_API_KEY, GOOGLE_GENAI_MODEL |

Add custom profiles under `runtimes:` in config.yaml.

## Development

```bash
make build      # Build binary
make test       # Run tests
make test-race  # Run tests with race detector
make install    # Install to $GOPATH/bin
```
