# agent-compose

Compose agents with the right model, MCP servers, skills, and credentials, and run them securely in OpenShell sandboxes. Zero plumbing.

## The Problem

Running an agent in an OpenShell sandbox requires 8+ manual configuration steps: choosing the right image, creating providers, figuring out framework-specific env vars, configuring egress, assembling prompts. Every framework (Claude Code, Codex, ADK, LangGraph) uses different env var names for the same concepts.

## The Solution

A composition engine that resolves declarative agent configs into raw `openshell` commands. Catalogs provide the menu (what's available). Agent config is the order (what I want). The engine connects them into a running sandbox.

```
Catalogs (what's available)    Agent Config (what I want)    Engine (connect)    OpenShell (run)
models, MCP, skills, policies  harness + inference + mcp     resolve             sandbox create
```

## Quick Start

```bash
# Build
make build

# Initialize config
./ac init
# Edit ~/.agentctl/config.yaml to add your inference providers and MCP servers

# Run an agent (zero config, dry-run)
./ac run --harness claude-code --inference maas --prompt "Review this code" --dry-run

# Run a named agent
./ac run code-reviewer --workspace ./my-project --dry-run

# Inspect resolved config
./ac inspect code-reviewer

# List running agents
./ac ps

# Stop an agent
./ac stop code-reviewer
```

## Config

One file for infrastructure (`~/.agentctl/config.yaml`), one entry per agent. Three friction tiers:

**Zero files (CLI flags):**
```bash
ac run --harness claude-code --inference maas --mcp github --prompt "Review this"
```

**Named agent (config entry):**
```yaml
# ~/.agentctl/config.yaml
agents:
  code-reviewer:
    harness: claude-code
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
├── config.yaml
├── agents/
└── skills/
```

## Agent Types

| Type | Declaration | Examples |
|---|---|---|
| Harness agent | `harness: claude-code` | Claude Code, Codex, Goose |
| Framework agent | `image:` + `env-mapping:` | ADK, LangGraph, CrewAI |
| Custom agent | `image:` + `entrypoint:` | Any container |

Harness profiles translate between "inference endpoint" (generic) and framework-specific env vars (`ANTHROPIC_BASE_URL` vs `OPENAI_BASE_URL` vs `GOOGLE_GENAI_BASE_URL`).

## Skills

Skills are reusable bundles of prompt instructions + tool/MCP requirements.

```
~/.agentctl/skills/security-review/
├── SKILL.md           # prompt (appended to agent's prompt)
└── references/        # mounted at /workspace/skills/<name>/
    └── owasp-top-10.md
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

The core is a Go library (`pkg/compose/`). CLI and API are thin interfaces.

```
┌─────────┐   ┌─────────┐   ┌─────────┐
│   CLI   │   │   API   │   │   SDK   │
│  (ac)   │   │ (HTTP)  │   │(Go pkg) │
└────┬────┘   └────┬────┘   └────┬────┘
     └──────────┬──┘──────────┘
                ▼
         Core Engine
     Resolver → ResolvedSpec
                ▼
         openshell CLI
```

**Resolver interfaces** are pluggable. V1 reads from config.yaml. V2 can discover models from KServe, MCP servers from MCP Gateway, skills from OCI registries.

```go
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell")),
    compose.WithSkillsDir("~/.agentctl/skills"),
)

// Resolve only (for harnesses/frameworks that create their own sandboxes)
spec, _ := engine.Resolve(ctx, "code-reviewer")

// Or resolve + create + run
run, _ := engine.Run(ctx, "code-reviewer", compose.RunOpts{Prompt: "Check PR #42"})
```

## Built-in Harness Profiles

| Harness | Image | Env Vars |
|---|---|---|
| claude-code | ghcr.io/anthropics/claude-code:latest | ANTHROPIC_BASE_URL, ANTHROPIC_API_KEY, ANTHROPIC_DEFAULT_SONNET_MODEL |
| codex | ghcr.io/openai/codex:latest | OPENAI_BASE_URL, OPENAI_API_KEY, OPENAI_MODEL |
| goose | ghcr.io/block/goose:latest | OPENAI_BASE_URL, OPENAI_API_KEY, GOOSE_MODEL |
| adk | python:3.12-slim | GOOGLE_GENAI_BASE_URL, GOOGLE_API_KEY, GOOGLE_GENAI_MODEL |

Add custom profiles in `config.yaml` under `harnesses:`.

## Commands

```
ac init                Create ~/.agentctl/ with default config
ac run <name> [flags]  Resolve + create sandbox + start agent
ac stop <name>         Stop agent + delete sandbox
ac ps                  List running agents
ac logs <name>         Stream sandbox output
ac inspect <name>      Show fully resolved spec as JSON
```

## Development

```bash
make build      # Build binary
make test       # Run tests
make test-race  # Run tests with race detector
make install    # Install to $GOPATH/bin
```
