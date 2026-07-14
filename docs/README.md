# agent-compose Documentation

## Getting Started

| Doc | Description |
|---|---|
| [../README.md](../README.md) | Quick start, commands, built-in runtimes |
| [tutorial.md](tutorial.md) | Step-by-step tutorial: build, init, dry-run, skills, MCP, inference, named agents, SDK |
| [running-agents.md](running-agents.md) | Examples: Claude Code via Vertex, custom agents against vLLM |

## Concepts

| Doc | Description |
|---|---|
| [composition.md](composition.md) | How agents are composed: config structure, skills, MCP servers, N-var env-mapping, resolution pipeline |
| [architecture.md](architecture.md) | Engine design, resolver interfaces, executor, Go SDK, sandbox lifecycle |
| [personas.md](personas.md) | Developer, platform engineer, team lead workflows and when GitOps makes sense |

## Diagrams

| File | Description |
|---|---|
| [architecture.png](architecture.png) | System architecture: transports, engine, resolvers, executor, catalogs |
| [resolution-pipeline.png](resolution-pipeline.png) | 12-step resolution from agent config to openshell commands |

Source files (`.drawio`) are alongside the PNGs.

## Test Evidence

| Doc | Description |
|---|---|
| [test-results.md](test-results.md) | Live test output: Claude Code, GPU cluster inference, skills, MCP, CLI, SDK |

## Upstream Issues

Validated gaps in OpenShell with evidence and workarounds.

| Issue | Summary | Impact |
|---|---|---|
| [001](upstream-issues/001-metadata-emulator-not-running.md) | GCE metadata emulator not running with vertex provider | All GCP SDK consumers can't get credentials inside sandboxes |
| [002](upstream-issues/002-provider-env-vars-not-injected.md) | Provider `env_vars` not injected as sandbox env vars | CLI tools that check auth locally (gh, aws, gcloud) fail |
