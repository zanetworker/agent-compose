# Personas and When GitOps Makes Sense

Three personas use agent-compose differently. The config surface is designed so each works independently.

## The Developer

Runs agents on a laptop. Cares about getting a working agent fast.

```bash
ac init                                                    # one-time
ac run --runtime claude-code-vertex --prompt "Review this"  # inline
ac run security-reviewer --workspace ./repo                 # named agent
```

**Config tier:** zero files (CLI flags) or one file (`~/.ac/config.yaml`). No GitOps needed.

## The Platform Engineer

Owns `config.yaml`: runtimes, inference providers, MCP servers, defaults.

```bash
vim config.yaml             # define infrastructure
ac apply --sync-profiles    # push to OpenShell gateway
ac doctor                   # verify everything resolves
```

**GitOps when:** managing multiple clusters or environments (dev/staging/prod):

```
infra-agents/
+-- base/
|   +-- config.yaml         # shared runtimes, skills
+-- overlays/
    +-- dev/config.yaml      # dev endpoints, relaxed policy
    +-- prod/config.yaml     # prod endpoints, strict policy, pinned digests
```

## The Team Lead

Defines named agents as separate YAML files. Each agent is a reviewed, versioned composition.

```
team-agents/
+-- config.yaml
+-- agents/
|   +-- code-reviewer.yaml
|   +-- test-runner.yaml
+-- skills/
    +-- security-review/SKILL.md
```

GitOps is essential here: agent definitions are reviewed in PRs, rollback is `git revert`.

## Summary

| Persona | Config tier | GitOps? | What they own |
|---|---|---|---|
| Developer | CLI flags or `~/.ac/config.yaml` | No | Their agents, their prompts |
| Platform Engineer | Shared `config.yaml` in a repo | Yes, for multi-env | Runtimes, inference, MCP, policies |
| Team Lead | Separate agent files in a team repo | Yes, always | Agent definitions, skills, team standards |
