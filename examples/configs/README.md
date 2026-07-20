# Example Configurations

Copy any of these into your `~/.ac/config.yaml` to get started.

## Agents

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
