# Example Configurations

Copy any of these into your `~/.ac/config.yaml` to get started.

## Config Examples

| Example | What it shows |
|---------|---------------|
| [minimal-agent.yaml](minimal-agent.yaml) | Simplest possible agent: runtime + prompt |
| [agent-with-mcp.yaml](agent-with-mcp.yaml) | Agent with MCP servers (GitHub + Sentry) |
| [agent-with-skills.yaml](agent-with-skills.yaml) | Agent with reusable skill (prompt + reference files) |
| [agent-with-everything.yaml](agent-with-everything.yaml) | Full composition: runtime + inference + MCP + skills + workspace |
| [framework-agent.yaml](framework-agent.yaml) | Custom Python agent (not a harness) |
| [multi-agent.yaml](multi-agent.yaml) | Multiple agents sharing the same infrastructure |

## How to use

```bash
# Copy an example into your config
cp examples/configs/agent-with-mcp.yaml ~/.ac/config.yaml

# Initialize providers
ac init

# Preview what gets created
ac get security-reviewer --json

# Dry-run (no sandbox created)
ac run security-reviewer --dry-run

# Run for real
ac run security-reviewer --workspace ./my-repo
```

## CLI Examples

### Inline agents (no config entry needed)

```bash
# Simplest: pick a runtime, give a prompt
ac run --runtime claude-code-vertex --prompt "What is 2+2?"

# With workspace
ac run --runtime claude-code-vertex \
  --prompt "Review this code for bugs" \
  --workspace ./my-repo

# With MCP servers
ac run --runtime claude-code-vertex \
  --mcp github \
  --prompt "What are the open PRs on this repo?" \
  --skip-permissions

# With skills
ac run --runtime claude-code-vertex \
  --mcp github \
  --skills security-review \
  --prompt "Review this PR for auth bypass" \
  --workspace ./my-repo \
  --skip-permissions

# Override the model
ac run --runtime claude-code-vertex \
  --inference gpu-vllm \
  --model qwen3-14b \
  --prompt "Hello"
```

### Named agents (from config)

```bash
# Run (blocks, streams output, auto-cleans up)
ac run security-reviewer --workspace ./my-repo

# Override the model for this run
ac run security-reviewer --model llama-3.3-70b

# Override the prompt
ac run security-reviewer --prompt "Focus only on SQL injection"

# Interactive (drops into Claude session)
ac run security-reviewer -i

# Dry-run (shows openshell commands)
ac run security-reviewer --workspace ./my-repo --dry-run
```

### Background agents

```bash
# Start in background (returns immediately)
ac start security-reviewer --workspace ./my-repo --skip-permissions

# Check on it
ac list
ac logs <sandbox-name>

# Shell into the sandbox
ac attach <sandbox-name>

# Stop and clean up
ac stop <sandbox-name>
```

### Framework agents (custom code)

```bash
# Upload your code + run it
ac run my-adk-agent --workspace ./examples/adk-agent

# Same agent, different model
ac run my-adk-agent --workspace ./examples/adk-agent --model llama-3.3-70b
```

### Inspect and debug

```bash
# See the fully resolved spec (image, env, providers, egress, prompt, MCP)
ac get security-reviewer --json

# Validate config + check gateway + verify providers
ac doctor

# System logs (gateway/supervisor, not agent output)
ac logs <sandbox-name> --system
```
