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

## Running Agents

### Prerequisites

1. An OpenShell gateway running (local with podman or on a cluster via Helm)
2. `openshell` CLI installed and connected to the gateway (`openshell status` shows Connected)
3. `ac` binary built (`make build`)

### Example 1: Claude Code via Vertex AI

Claude Code needs Anthropic's API. If you use Vertex AI (GCP), set up the `google-vertex-ai` provider once:

```bash
# One-time: create the Vertex provider from your local gcloud ADC
openshell provider create --type google-vertex-ai --name vertex --from-gcloud-adc

# Run Claude Code in a sandbox
openshell sandbox create --name my-claude \
  --provider vertex \
  --env CLAUDE_CODE_USE_VERTEX=1 \
  --env CLOUD_ML_REGION=us-east5 \
  --env ANTHROPIC_VERTEX_PROJECT_ID=your-project-id \
  --env GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcloud-adc.json \
  --upload ~/.config/gcloud/application_default_credentials.json:/tmp/gcloud-adc.json \
  --auto-providers --no-tty \
  -- claude -p "Say hello" --max-turns 1 --dangerously-skip-permissions
```

You also need to open egress for Vertex and OAuth (the `google-vertex-ai` profile currently misses `oauth2.googleapis.com`):

```bash
openshell policy update my-claude \
  --add-endpoint "us-east5-aiplatform.googleapis.com:443:read-write:rest:enforce" \
  --add-endpoint "oauth2.googleapis.com:443:read-write:rest:enforce" \
  --add-endpoint "statsig.anthropic.com:443:read-write:rest:enforce" \
  --binary /usr/local/bin/claude
```

With agent-compose, this becomes:

```bash
ac run --runtime claude-code-vertex --prompt "Say hello"
```

The engine resolves the runtime profile, attaches the `google-vertex-ai` provider, sets the Vertex env vars, and creates the sandbox. The policy update step is still manual until the upstream profile gap is fixed.

### Example 2: Custom agent against a vLLM endpoint (GPU cluster)

Any agent that uses the OpenAI-compatible API can call models served by vLLM/KServe:

```bash
# Create a sandbox with the inference endpoint as an env var
openshell sandbox create --name my-agent \
  --env OPENAI_BASE_URL=https://qwen3-14b-user-nxu.apps.ocp.cloud.rhai-tmm.dev/v1 \
  --env OPENAI_MODEL=qwen3-14b \
  --no-tty

# Open egress to the endpoint
openshell policy update my-agent \
  --add-endpoint "qwen3-14b-user-nxu.apps.ocp.cloud.rhai-tmm.dev:443:read-write:rest:enforce" \
  --binary /usr/bin/curl

# Run a query from inside the sandbox
openshell sandbox exec --name my-agent -- \
  curl -sk $OPENAI_BASE_URL/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen3-14b","messages":[{"role":"user","content":"Hello"}],"max_tokens":20}'
```

With agent-compose:

```yaml
# ~/.ac/config.yaml
runtimes:
  qwen-agent:
    kind: raw
    image: ghcr.io/nvidia/openshell-community/sandboxes/base:latest
    env-mapping:
      OPENAI_BASE_URL: "${endpoint}"
      OPENAI_MODEL: "${model}"
    entrypoint: ["python3", "agent.py"]

inference:
  gpu-cluster:
    endpoint: https://qwen3-14b-user-nxu.apps.ocp.cloud.rhai-tmm.dev/v1
    provider: ""
    default-model: qwen3-14b

agents:
  my-agent:
    runtime: qwen-agent
    prompt: "You are a helpful assistant"
```

```bash
ac run my-agent
```

### Example 3: ADK / LangGraph agent via Vertex AI

Framework agents use the same provider system. An ADK agent that calls Gemini via Vertex:

```yaml
runtimes:
  support-bot:
    kind: framework
    image: myco/support-bot:v2.1
    env-mapping:
      GOOGLE_GENAI_MODEL: "${model}"
    entrypoint: ["python", "-m", "agent"]
    providers: [google-vertex-ai]

agents:
  support:
    runtime: support-bot
```

```bash
# One-time provider setup
openshell provider create --type google-vertex-ai --name vertex --from-gcloud-adc

# Run
ac run support
```

The `google-vertex-ai` provider handles credentials via OpenShell's GCE metadata emulator. The ADK agent's Python code calls `google.auth.default()` and gets a token transparently.

### Example 4: Composing an agent with MCP servers, skills, and prompt

This is where agent-compose shines. Instead of wiring 8 things manually, you declare what the agent needs and the engine resolves everything.

**Step 1: Platform engineer configures the infrastructure (once)**

```yaml
# ~/.ac/config.yaml

runtimes:
  claude-code:
    kind: harness
    image: ghcr.io/anthropics/claude-code:latest
    env-mapping:
      ANTHROPIC_BASE_URL: "${endpoint}"
      ANTHROPIC_DEFAULT_SONNET_MODEL: "${model}"
    entrypoint: ["claude", "--prompt-file", "/workspace/prompt.md"]
    tools: [shell, file-read, file-write, bundle-mcp]
    providers: [claude-code]

inference:
  maas:
    endpoint: https://maas.apps.cluster.com/v1
    provider: maas-anthropic
    default-model: granite-3.3-8b-instruct
    models:
      opus: granite-3.3-8b-instruct
      haiku: granite-3.3-2b-instruct

mcp:
  github:
    provider: github
    egress: [api.github.com:443, github.com:443]

  jira:
    provider: jira
    egress: [redhat.atlassian.net:443]

  slack:
    provider: slack
    egress: [slack.com:443, api.slack.com:443]

defaults:
  inference: maas
  sandbox:
    scope: session
    mode: all
    ttl: 30m
```

This defines: which runtimes exist, which inference endpoints are available, which MCP servers are reachable, and the defaults. Developers never touch this.

**Step 2: Team lead creates skills (reusable prompt + dependency bundles)**

```
~/.ac/skills/
+-- security-review/
|   +-- SKILL.md
|   +-- references/
|       +-- owasp-top-10.md
|       +-- cwe-patterns.md
+-- pr-review/
|   +-- SKILL.md
+-- test-writer/
    +-- SKILL.md
```

Each SKILL.md is a prompt that can declare its own MCP and tool dependencies:

```markdown
# ~/.ac/skills/security-review/SKILL.md
---
requires:
  mcp: [github]
  tools: [shell, file-read]
---

# Security Review

When reviewing code, check for:
1. SQL injection (parameterized queries vs string concat)
2. XSS (output encoding, CSP headers)
3. Auth bypass (missing checks, IDOR)
4. Secrets in code (API keys, passwords, tokens)

Reference the OWASP top 10 at /workspace/skills/security-review/owasp-top-10.md
```

```markdown
# ~/.ac/skills/pr-review/SKILL.md
---
requires:
  mcp: [github, jira]
  tools: [shell, file-read]
---

# PR Review

1. Read the PR diff via GitHub MCP
2. Check for related Jira tickets
3. Review for correctness, security, and style
4. Post review comments via GitHub MCP
```

Skills compose: if two skills both require `github`, it's deduplicated. If a skill requires an MCP server the agent didn't list, it gets added automatically.

**Step 3: Team lead defines named agents (the compositions)**

```yaml
# in config.yaml or as separate files

agents:
  # Security-focused code reviewer
  security-reviewer:
    runtime: claude-code
    inference: maas
    mcp: [github]
    skills: [security-review]
    prompt: "You are a security-focused code reviewer. Be thorough but concise."

  # Full PR reviewer with Jira integration
  pr-reviewer:
    runtime: claude-code
    mcp: [github, jira]
    skills: [pr-review, security-review]
    prompt: "Review PRs end-to-end. Check Jira for context. Post inline comments."

  # Test writer
  test-writer:
    runtime: claude-code
    mcp: [github]
    skills: [test-writer]
    prompt: "Write tests for the changed files. Focus on edge cases."
```

Each agent is a composition of: runtime (how to run it) + inference (which model) + MCP servers (what tools it can access) + skills (what instructions and references it gets) + prompt (what to do).

**Step 4: Developer runs agents**

```bash
# Run the security reviewer on a repo
ac run security-reviewer --workspace ./my-project

# Run the PR reviewer (has both github + jira MCP, two skills)
ac run pr-reviewer --workspace ./my-project

# Run the test writer with a different model
ac run test-writer --model llama-3.3-70b --workspace ./my-project

# Compose ad hoc: pick a runtime, add MCP servers and skills inline
ac run --runtime claude-code \
       --mcp github \
       --mcp jira \
       --skills security-review \
       --skills pr-review \
       --prompt "Review PR #42, cross-reference Jira tickets" \
       --workspace ./my-project

# See exactly what the engine resolved
ac get pr-reviewer --json
```

**What the engine does for `ac run pr-reviewer`:**

```
1. Load agent "pr-reviewer" from config
2. Resolve runtime "claude-code"
   -> image, entrypoint, provider: claude-code
3. Resolve inference "maas" (from defaults)
   -> expand env-mapping: ANTHROPIC_BASE_URL, ANTHROPIC_DEFAULT_SONNET_MODEL
4. Resolve MCP "github"
   -> provider: github, egress: api.github.com:443
5. Resolve MCP "jira"
   -> provider: jira, egress: redhat.atlassian.net:443
6. Resolve skill "pr-review"
   -> appends PR review prompt
   -> merges requires.mcp: [github, jira] (already present, deduped)
   -> merges requires.tools: [shell, file-read]
7. Resolve skill "security-review"
   -> appends security review prompt
   -> merges requires.mcp: [github] (already present)
   -> mounts references/owasp-top-10.md at /workspace/skills/security-review/
8. Assemble final prompt: agent prompt + pr-review prompt + security-review prompt
9. Merge egress: [maas:443, api.github.com:443, github.com:443, redhat.atlassian.net:443]
10. Collect providers: [claude-code, maas-anthropic, github, jira]
11. Apply sandbox opts: scope=session, ttl=30m
12. Return ResolvedSpec
```

The developer typed one command. The engine resolved 4 providers, 2 MCP servers, 2 skills (with deduped dependencies), assembled the prompt, and produced the sandbox spec. Without agent-compose, that's 8+ manual `openshell` commands with 20+ flags to get right every time.

## What the Engine Produces

```bash
$ ac run --runtime claude-code --prompt "test" --dry-run

openshell sandbox create --name inline-1234 \
  --from ghcr.io/anthropics/claude-code:latest \
  --auto-providers --no-tty \
  --provider claude-code \
  --env ANTHROPIC_DEFAULT_SONNET_MODEL=granite-3.3-8b-instruct \
  --scope session --mode all --ttl 30m \
  --label agentctl.io/agent=inline-1234

openshell sandbox exec --name inline-1234 -- claude --prompt-file /workspace/prompt.md
```

Credentials are handled by the `claude-code` provider (OpenShell injects `ANTHROPIC_API_KEY`). Only non-credential env vars (model name) go through `--env`. All 8 manual steps collapsed into one command.

## Go SDK

The CLI is a thin wrapper. Everything is available programmatically via `pkg/compose`:

```go
import "github.com/zanetworker/agent-compose/pkg/compose"

// Build the engine from config
engine := compose.New(
    compose.WithConfig(cfg),
    compose.WithExecutor(compose.NewCLIExecutor("openshell")),
    compose.WithSkillsDir("~/.ac/skills"),
)

// Resolve only (for harnesses/frameworks that create their own sandboxes)
spec, err := engine.Resolve(ctx, "code-reviewer")
// spec.Image, spec.Env, spec.Providers, spec.Egress, spec.Prompt, spec.Sandbox, ...

// Resolve + create + run (P1 pattern)
run, err := engine.Run(ctx, "code-reviewer", compose.RunOpts{
    Workspace: "./repo",
})

// Override inference and model per run
run, err = engine.Run(ctx, "code-reviewer", compose.RunOpts{
    Inference: "local-vllm",
    Model:     "llama-3.3-70b",
    Prompt:    "Focus on auth bypass",
})

// Compose an inline agent (no config entry needed)
run, err = engine.Run(ctx, "", compose.RunOpts{
    Agent: &compose.Agent{
        Runtime: "claude-code",
        MCP:     []string{"github"},
        Prompt:  "Review this code",
    },
    Inference: "maas",
})

// Lifecycle
statuses, _ := engine.List(ctx)
engine.Stop(ctx, "code-reviewer-1234")
logs, _ := engine.Logs(ctx, "code-reviewer-1234")

// Introspection
spec, _ = engine.Get(ctx, "code-reviewer")   // resolved spec as JSON
results := compose.Doctor(cfg, skillsDir, "openshell")

// Profile sync
ids, _ := engine.SyncProfiles(ctx)
```

SDK tests are in `examples/sdk_test.go` and cover: resolving a named agent, overriding inference/model, and composing inline agents. Run with `go test ./examples/ -v`.

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

See the [Go SDK](#go-sdk) section for programmatic usage.

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

| Runtime | Kind | Image | OpenShell Provider | Env Vars (non-credential) |
|---|---|---|---|---|
| claude-code | harness | ghcr.io/anthropics/claude-code | claude-code | ANTHROPIC_BASE_URL, ANTHROPIC_DEFAULT_SONNET_MODEL |
| claude-code-vertex | harness | ghcr.io/anthropics/claude-code | google-vertex-ai | CLAUDE_CODE_USE_VERTEX, CLOUD_ML_REGION, ANTHROPIC_VERTEX_PROJECT_ID |
| codex | harness | ghcr.io/openai/codex | codex | OPENAI_BASE_URL, OPENAI_MODEL |
| goose | harness | ghcr.io/block/goose | (none) | OPENAI_BASE_URL, GOOSE_MODEL |
| adk | framework | python:3.12-slim | google-vertex-ai | GOOGLE_GENAI_MODEL |

Credentials (API keys, tokens) are handled by OpenShell providers, not env vars. Only framework-specific vars (model names, endpoint overrides) go through the env-mapping.

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

## Tested End-to-End

All tests run against a live OpenShell gateway (podman driver, macOS, July 2026).

### Harness: Claude Code via Vertex AI

OpenShell sandbox with Landlock, network proxy, egress policy. Claude Code called Vertex AI and responded.

```
$ openshell sandbox exec --name claude-v2 --no-tty -- \
    claude -p "Say hello in exactly 3 words" --max-turns 1 --dangerously-skip-permissions

Hello! Let's code.
```

```
$ openshell sandbox exec --name claude-v2 --no-tty -- \
    claude -p "What is 2+2? Just the number" --max-turns 1 --dangerously-skip-permissions

4
```

**What was configured:** `google-vertex-ai` provider (created via `--from-gcloud-adc`), Vertex env vars (`CLAUDE_CODE_USE_VERTEX`, `CLOUD_ML_REGION`, `ANTHROPIC_VERTEX_PROJECT_ID`), ADC file uploaded, and a policy update to allow `oauth2.googleapis.com` + `us-east5-aiplatform.googleapis.com` for `/usr/local/bin/claude`.

### Framework: Python agent calling GPU cluster (qwen3-14b)

Python script using `urllib` inside a sandbox, calling a vLLM model on a remote GPU cluster.

```
$ openshell sandbox exec --name adk-test --no-tty -- \
    bash -c 'curl -sk $OPENAI_BASE_URL/chat/completions \
    -H "Content-Type: application/json" \
    -d "{...qwen3-14b...}" | python3 -c "..."'

def add(a, b):
    return a + b
```

**What was configured:** `OPENAI_BASE_URL` and `OPENAI_MODEL` env vars, policy update to allow `qwen3-14b-user-nxu.apps.ocp.cloud.rhai-tmm.dev:443` for `/usr/bin/curl`.

### Skills: prompt assembly and reference file mounting

Created a `code-style` skill with a SKILL.md and a reference file. Verified the resolver assembles the combined prompt and the files are accessible inside the sandbox.

```
$ ac get style-checker --skills-dir /tmp/ac-test-skills

{
  "prompt": "Check this code for style violations.\n\n# Code Style Review\n\n
    When reviewing code, enforce these rules:\n1. No functions longer than 50 lines...",
  "skill_mounts": [
    {"Source": "/tmp/ac-test-skills/code-style/references/style-guide.md",
     "Target": "/workspace/skills/code-style/"}
  ]
}
```

```
$ openshell sandbox create --name skills-test \
    --upload "/tmp/test-prompt.md:/sandbox/prompt.md" \
    --upload ".../style-guide.md:/sandbox/skills/code-style/style-guide.md" \
    --no-tty \
    -- bash -c "cat /sandbox/prompt.md && cat /sandbox/skills/code-style/style-guide.md"

Check this code for style violations.
# Code Style Review
When reviewing code, enforce these rules:
1. No functions longer than 50 lines
...
# Style Guide Reference
This is a reference file mounted from the skill.
```

**What was verified:** Skill prompt appended to agent prompt, skill tool/MCP dependencies merged (deduped), reference files uploaded via `--upload` and readable inside the sandbox.

### MCP: GitHub provider with egress policy

Created a `github` provider from a local GitHub token. Verified the sandbox can access GitHub (via allowed binaries) and blocks everything else.

```
$ openshell provider create --type github --name github --credential "api_token=$(gh auth token)"
Created provider github

$ openshell sandbox create --name mcp-test --provider github --no-tty

# git works (binary allowed by github profile)
$ openshell sandbox exec --name mcp-test -- git ls-remote https://github.com/NVIDIA/OpenShell.git HEAD
94cdd697c55aedb571f177ec13cfa54a8e213919  HEAD

# curl blocked (binary not in github profile's allowed list)
$ openshell sandbox exec --name mcp-test -- curl -sv https://api.github.com/user
CONNECT tunnel failed, response 403

# unrelated endpoints blocked
$ openshell sandbox exec --name mcp-test -- curl -sv https://example.com
CONNECT tunnel failed, response 403
```

**What was verified:** Provider attaches credentials and egress policy. Only binaries declared in the provider profile (`/usr/bin/git`, `/usr/bin/gh`) can access GitHub endpoints. Other binaries and other endpoints are blocked. The `gh` CLI requires `GH_TOKEN` as an env var (not just proxy-level injection), which is a known OpenShell limitation for CLI tools that check auth locally before making network requests.

### CLI: dry-run, overrides, composition

```
$ ac run --runtime claude-code --prompt "Say hello" --dry-run

openshell sandbox create --name inline-1234 \
  --from ghcr.io/nvidia/openshell-community/sandboxes/base:latest \
  --auto-providers --no-tty \
  --provider claude-code \
  --scope session --mode all --ttl 30m \
  --label agentctl.io/agent=inline-1234
openshell sandbox exec --name inline-1234 -- claude -p Say hello --dangerously-skip-permissions
```

```
$ ac run code-reviewer --inference local-vllm --model custom-7b --dry-run

openshell sandbox create ... --provider claude-code --provider vllm-local \
  --env ANTHROPIC_BASE_URL=https://vllm.internal:8000/v1 \
  --env ANTHROPIC_DEFAULT_SONNET_MODEL=custom-7b ...
```

### Lifecycle: list, stop, doctor

```
$ ac list
NAME              SANDBOX           STATUS   AGE
hello-1783881595  hello-1783881595  running  0s

$ ac stop hello-1783881595
Agent hello-1783881595 stopped

$ ac doctor
Runtimes
  claude-code          image specified                ok
  claude-code          entrypoint specified           ok
  ...
OpenShell
  gateway              reachable                      ok
  maas                 profile synced                 FAIL (run: ac apply --sync-profiles)
  github               profile synced                 ok
```

### SDK: programmatic usage

```
$ go test ./examples/ -v

=== RUN   TestSDK_ResolveAgent
{
  "runtime_kind": "harness",
  "image": "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
  "providers": ["claude-code", "maas-anthropic", "github-pat"],
  "env": {
    "ANTHROPIC_BASE_URL": "https://maas.apps.cluster.com/v1",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "granite-3.3-8b-instruct"
  },
  "sandbox": {"scope": "session", "mode": "all", "ttl": "30m"},
  ...
}
--- PASS: TestSDK_ResolveAgent
=== RUN   TestSDK_ResolveWithOverrides
--- PASS: TestSDK_ResolveWithOverrides
=== RUN   TestSDK_InlineAgent
--- PASS: TestSDK_InlineAgent
```

### Known upstream gaps

- OpenShell's `google-vertex-ai` provider profile is missing `oauth2.googleapis.com` in its endpoints, requiring a manual `openshell policy update` for Vertex auth token refresh
- Claude Code's Vertex integration uses file-based ADC (`GOOGLE_APPLICATION_CREDENTIALS`), not OpenShell's metadata emulator. Workaround: `--upload` the ADC file. Proper fix is upstream (metadata emulator support for Claude Code's auth path)
- OpenShell's `--auto-providers` doesn't support `--from-existing` discovery for `google-vertex-ai`; use `openshell provider create --from-gcloud-adc` explicitly
- CLI tools that check auth locally before making requests (e.g., `gh` requires `GH_TOKEN` env var) don't work with OpenShell's proxy-level credential injection. The proxy injects auth headers at the network layer, but the CLI refuses to make the request without a local token. Workaround: pass the token as `--env GH_TOKEN=...` in addition to the provider
- The sandbox's writable directory is `/sandbox` (not `/workspace`). Skill reference mounts and prompt files should target `/sandbox/` paths

## Development

```bash
make build      # Build binary
make test       # Run tests
make test-race  # Run tests with race detector
make install    # Install to $GOPATH/bin
```
