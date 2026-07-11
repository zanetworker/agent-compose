# agent-compose

Agent composition engine for OpenShell. Resolves declarative agent configs into sandbox commands. One declaration replaces 8 manual steps.

## The Problem

Running an agent in an OpenShell sandbox requires 8+ manual steps: choosing the right image, creating providers for credentials, figuring out framework-specific env vars (`ANTHROPIC_BASE_URL` vs `OPENAI_BASE_URL` vs `GOOGLE_GENAI_BASE_URL`), setting those vars, configuring egress, assembling prompts. Every framework uses different names for the same concepts.

## The Solution

A composition engine that connects catalogs (what's available) to sandboxes (where agents run). You declare what the agent needs; the engine resolves it into raw `openshell` commands.

![Architecture](docs/architecture.png)

```
Catalogs (config.yaml)     Agent Config (what I want)     Engine (resolve)     OpenShell (run)
models, MCP, skills        runtime + inference + mcp      ResolvedSpec         sandbox create
```

## Quick Start

```bash
# Build
make build

# Initialize config
ac init
# Edit ~/.ac/config.yaml with your inference providers and MCP servers

# Validate your setup
ac doctor

# Run an agent with inline flags (zero config needed)
ac run --runtime claude-code --inference maas --prompt "Review this code" --dry-run

# Run a named agent from config
ac run code-reviewer --workspace ./my-project

# Override inference or model at run time
ac run code-reviewer --inference local-vllm --model llama-3.3-70b

# Show fully resolved spec as JSON
ac get code-reviewer --json

# List running agents
ac list

# Stop an agent
ac stop code-reviewer

# Sync provider profiles to OpenShell gateway
ac apply --sync-profiles
```

## What the Engine Produces

```bash
$ ac run --runtime claude-code --prompt "test" --dry-run

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

One file (`~/.ac/config.yaml`). Three friction tiers:

**Zero files (CLI flags):**
```bash
ac run --runtime claude-code --inference maas --mcp github --prompt "Review this"
```

**Named agent (config entry):**
```yaml
# ~/.ac/config.yaml
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

## Resolution Pipeline

The engine resolves agent configs in 12 steps, producing a `ResolvedSpec` that contains everything OpenShell needs.

![Resolution Pipeline](docs/resolution-pipeline.png)

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

Override at run time:

```bash
# different provider
ac run code-reviewer --inference vertex --model gemini-2.5-pro

# same provider, different model
ac run code-reviewer --model llama-3.3-70b
```

## Skills

Reusable bundles of prompt instructions + tool/MCP requirements:

```
~/.ac/skills/security-review/
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

When you pass `--skills security-review`, the resolver appends the skill's prompt, merges its MCP and tool requirements, and mounts its references into the sandbox.

## OpenShell Profile Integration

OpenShell has built-in provider profiles for known agents (claude-code, codex, copilot, cursor) and inference providers. agent-compose can push custom profiles to the OpenShell gateway so credentials and egress are handled natively:

```bash
# Push inference and MCP profiles to the gateway
ac apply --sync-profiles

# Profiles appear alongside built-in ones
openshell provider list-profiles
```

**Who owns what:**

| Layer | Owner |
|---|---|
| Credentials, network policy, token refresh | OpenShell (provider profiles) |
| Image, entrypoint, env-mapping, skills, prompt, sandbox lifecycle | agent-compose |

The engine generates OpenShell provider profile YAML from your config.yaml inference and MCP entries. `ac apply --sync-profiles` imports them into the gateway. After that, sandboxes get credential injection and egress policy automatically from OpenShell's policy composition pipeline.

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
    compose.WithSkillsDir("~/.ac/skills"),
)

// Resolve only (for harnesses that create their own sandboxes)
spec, _ := engine.Resolve(ctx, "code-reviewer")

// Or resolve + create + run
run, _ := engine.Run(ctx, "code-reviewer", compose.RunOpts{Workspace: "./repo"})

// Override inference and model per run
run, _ = engine.Run(ctx, "code-reviewer", compose.RunOpts{
    Inference: "local-vllm",
    Model:     "llama-3.3-70b",
})
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
ac init                          Create ~/.ac/ with default config
ac run <name> [flags]            Resolve + create sandbox + start agent
ac stop <name>                   Stop agent + delete sandbox
ac list                          List running agents
ac get <name>                    Show fully resolved spec as JSON
ac logs <name> [--follow]        Stream sandbox output
ac apply --sync-profiles         Push provider profiles to OpenShell gateway
ac doctor                        Validate config and check environment readiness
```

**Run flags:** `--runtime`, `--inference`, `--model`, `--mcp`, `--skills`, `--prompt`, `--workspace`

**Global flags:** `--json`, `--dry-run`, `--config <path>`, `--skills-dir <path>`

## Doctor

`ac doctor` validates both config integrity and the live OpenShell environment:

```
$ ac doctor

Config
  runtimes           claude-code, codex          ok
  inference          maas                        ok
  agents             code-reviewer               ok

OpenShell
  gateway            reachable                   ok
  profile: maas      imported                    ok
  profile: github    imported                    FAIL (run: ac apply --sync-profiles)
  provider: maas-anthropic   exists              ok

Inference
  maas               endpoint reachable          ok
  maas               model granite-3.3-8b        ok

1 issue to fix
```

If the gateway isn't reachable, it reports that and skips checks that depend on it.

## Built-in Runtime Profiles

| Runtime | Kind | Image | Key Env Vars |
|---|---|---|---|
| claude-code | harness | ghcr.io/anthropics/claude-code | ANTHROPIC_BASE_URL, ANTHROPIC_API_KEY, ANTHROPIC_DEFAULT_SONNET_MODEL |
| codex | harness | ghcr.io/openai/codex | OPENAI_BASE_URL, OPENAI_API_KEY, OPENAI_MODEL |
| goose | harness | ghcr.io/block/goose | OPENAI_BASE_URL, OPENAI_API_KEY, GOOSE_MODEL |
| adk | framework | python:3.12-slim | GOOGLE_GENAI_BASE_URL, GOOGLE_API_KEY, GOOGLE_GENAI_MODEL |

Add custom profiles under `runtimes:` in config.yaml.

## Personas and When GitOps Makes Sense

Three personas use agent-compose differently. The config surface is designed so each can work independently without blocking the others.

### The Developer (runs agents on a laptop)

Uses CLI flags or named agents from config.yaml. Cares about getting a working agent fast, not about infrastructure:

```bash
ac init
ac run --runtime claude-code --prompt "Review this PR" --workspace .
ac run code-reviewer --workspace ./repo
```

**Config tier:** zero files (CLI flags) or one file (`~/.ac/config.yaml`). No GitOps needed.

### The Platform Engineer (manages infrastructure config)

Owns `config.yaml`: runtimes, inference providers, MCP servers, policies, and defaults. Ensures the right credentials, egress rules, and security policies are in place.

```bash
vim config.yaml                    # define runtimes, inference, MCP
ac apply --sync-profiles           # push to OpenShell gateway
ac doctor                          # verify everything resolves
```

**Config tier:** one file, version-controlled. GitOps matters when managing multiple clusters/environments:

```
infra-agents/
+-- base/
|   +-- config.yaml         # shared runtimes, skills
+-- overlays/
    +-- dev/
    |   +-- config.yaml      # dev endpoints, relaxed policy
    +-- prod/
        +-- config.yaml      # prod endpoints, strict policy, pinned digests
```

### The Team Lead (manages agent definitions for a team)

Defines named agents as separate YAML files. Each agent is a composition of runtime + inference + MCP + skills + prompt, reviewed and versioned like code:

```
team-agents/
+-- config.yaml
+-- agents/
|   +-- code-reviewer.yaml
|   +-- test-runner.yaml
+-- skills/
    +-- security-review/
        +-- SKILL.md
```

GitOps is essential here: agent definitions are reviewed in PRs, rollback is `git revert`.

### Summary

| Persona | Config tier | GitOps? | What they own |
|---|---|---|---|
| Developer | CLI flags or `~/.ac/config.yaml` | No | Their agents, their prompts |
| Platform Engineer | Shared `config.yaml` in a repo | Yes, for multi-env | Runtimes, inference, MCP, policies |
| Team Lead | Separate agent files in a team repo | Yes, always | Agent definitions, skills, team standards |

## Current Status

The composition engine and CLI are complete. The resolution pipeline, all CLI commands, and dry-run mode have been functionally tested. Live execution against an OpenShell gateway (`ac run` without `--dry-run`, `ac apply --sync-profiles`, `ac doctor` live checks) requires a running gateway and has not been end-to-end tested yet.

## Development

```bash
make build      # Build binary
make test       # Run tests
make test-race  # Run tests with race detector
make install    # Install to $GOPATH/bin
```
