# Personas: Who Sets Up What

Three personas use agent-compose. Each has different prerequisites, responsibilities, and outputs. The work flows left to right: platform engineer sets up infrastructure, team lead defines agents, developer runs them.

```
Platform Engineer          Team Lead              Developer
sets up infra        -->   defines agents   -->   runs agents
(once per cluster)         (once per team)         (every day)
```

## Platform Engineer

**Job:** Make it possible for agents to run. Own the infrastructure: runtimes, inference endpoints, MCP servers, credentials, policies.

**Prerequisites:**
- OpenShell gateway running (local podman or Kubernetes cluster)
- `openshell` CLI installed and connected
- Access to inference endpoints (MaaS, vLLM, Vertex AI)
- Credentials for external services (GitHub PAT, Jira token, cloud auth)

**What they set up:**

```yaml
# ~/.ac/config.yaml (or a shared repo)

# 1. Runtimes: how agents run
runtimes:
  claude-code-vertex:
    kind: harness
    image: ghcr.io/nvidia/openshell-community/sandboxes/base:latest
    env-mapping:
      CLAUDE_CODE_USE_VERTEX: "1"
      CLOUD_ML_REGION: "${region}"
      ANTHROPIC_VERTEX_PROJECT_ID: "${project}"
      ANTHROPIC_DEFAULT_SONNET_MODEL: "${model}"
    entrypoint: ["claude"]
    providers: [google-vertex-ai]

# 2. Inference: where models are
inference:
  vertex:
    endpoint: https://us-east5-aiplatform.googleapis.com/v1
    provider: vertex
    default-model: claude-sonnet-4-20250514

  gpu-cluster:
    endpoint: https://qwen3-14b.apps.cluster.dev/v1
    provider: gpu-vllm
    default-model: qwen3-14b

# 3. MCP servers: what tools agents can access
mcp:
  github:
    provider: github
    egress: [api.github.com:443, github.com:443]
  jira:
    provider: jira
    egress: [redhat.atlassian.net:443]

# 4. Defaults
defaults:
  inference: vertex
  sandbox:
    scope: session
    mode: all
    ttl: 30m
```

**What they run:**

```bash
ac init                    # creates config + auto-detects credentials (Vertex, GitHub)
ac apply --sync-profiles   # pushes custom profiles to OpenShell gateway
ac doctor                  # verifies everything resolves and gateway is healthy
```

**What they hand off:** a `config.yaml` that the team lead and developers consume. Developers never need to know inference endpoints, provider names, or egress rules.

**When GitOps matters:** multiple clusters or environments. Per-environment overlays:
```
infra-agents/
+-- base/config.yaml          # shared runtimes, common MCP servers
+-- overlays/
    +-- dev/config.yaml        # dev inference endpoints, relaxed TTL
    +-- prod/config.yaml       # prod endpoints, strict policy, pinned image digests
```

## Team Lead

**Job:** Define what agents exist and what they can do. Own agent definitions, skills, and team standards.

**Prerequisites:**
- Platform engineer has set up `config.yaml` (runtimes, inference, MCP exist)
- Understanding of what each agent should do (prompt, tools, model)

**What they set up:**

**Skills** (reusable prompt + dependency bundles):

```
~/.ac/skills/
+-- security-review/
|   +-- SKILL.md                    # prompt + MCP/tool dependencies
|   +-- references/
|       +-- owasp-top-10.md         # reference file mounted into sandbox
+-- pr-review/
    +-- SKILL.md
```

```markdown
# ~/.ac/skills/security-review/SKILL.md
---
requires:
  mcp: [github]
  tools: [shell, file-read]
---

# Security Review
When reviewing code, check for:
1. SQL injection
2. XSS
3. Auth bypass
4. Secrets in code
```

**Named agents** (compositions of runtime + inference + MCP + skills + prompt):

```yaml
# in config.yaml or as separate files in agents/

agents:
  security-reviewer:
    runtime: claude-code-vertex
    mcp: [github]
    skills: [security-review]
    prompt: "You are a security-focused code reviewer."

  pr-reviewer:
    runtime: claude-code-vertex
    mcp: [github, jira]
    skills: [pr-review, security-review]
    prompt: "Review PRs end-to-end. Check Jira for context."

  test-writer:
    runtime: claude-code-vertex
    mcp: [github]
    prompt: "Write tests for the changed files. Focus on edge cases."
```

**What they verify:**

```bash
ac get security-reviewer --json   # check resolved spec looks right
ac get pr-reviewer --json         # verify skills merged, MCP deduped
ac doctor                         # verify all references resolve
```

**What they hand off:** named agents that developers run with `ac run <name>`. Developers don't need to know which MCP servers, skills, or model the agent uses.

**When GitOps matters:** always. Agent definitions are reviewed in PRs. Prompt changes are visible in diffs. Rollback is `git revert`.

```
team-agents/
+-- config.yaml
+-- agents/
|   +-- security-reviewer.yaml
|   +-- pr-reviewer.yaml
|   +-- test-writer.yaml
+-- skills/
    +-- security-review/SKILL.md
    +-- pr-review/SKILL.md
```

## Developer

**Job:** Run agents. Focus on the work, not the plumbing.

**Prerequisites:**
- `ac` binary installed
- `ac init` run (one-time, creates config + providers)
- Platform engineer and team lead have set up config + agents

**What they run:**

```bash
# Named agent (team lead defined it)
ac run security-reviewer --workspace ./my-project

# Override model for a single run
ac run security-reviewer --model llama-3.3-70b

# Compose ad hoc (no config entry needed)
ac run --runtime claude-code-vertex \
       --mcp github \
       --skills security-review \
       --prompt "Review PR #42" \
       --workspace ./my-project

# Inspect what was resolved
ac get security-reviewer --json

# Lifecycle
ac list
ac stop security-reviewer
```

**What they don't need to know:**
- Which OpenShell providers are attached
- What env vars the runtime needs
- Which endpoints have egress rules
- How credentials are injected
- What sandbox scope/mode/ttl means

**When GitOps matters:** never. Developers use CLI flags or consume named agents from config.

## Responsibility Matrix

| What | Platform Engineer | Team Lead | Developer |
|---|---|---|---|
| OpenShell gateway | Installs, maintains | | |
| Runtimes (image, entrypoint, providers) | Defines in config.yaml | | |
| Inference providers (endpoints, models) | Defines in config.yaml | | Overrides per run (`--model`) |
| MCP servers (credentials, egress) | Defines in config.yaml | References in agents | References inline (`--mcp`) |
| Skills (SKILL.md, references) | | Creates and maintains | References inline (`--skills`) |
| Named agents (compositions) | | Defines in config.yaml | Runs (`ac run <name>`) |
| Providers (openshell) | Creates via `ac init` | | |
| Profile sync | Runs `ac apply --sync-profiles` | | |
| Validation | Runs `ac doctor` | Runs `ac doctor` | |
| Running agents | | Tests with `ac run --dry-run` | `ac run` |

## Handoff Flow

```
Platform Engineer                    Team Lead                      Developer
                                                                    
1. Install OpenShell gateway                                        
2. Write config.yaml (runtimes,                                     
   inference, MCP, defaults)                                        
3. ac init (create providers)                                       
4. ac apply --sync-profiles                                         
5. ac doctor (verify)                                               
6. Share config.yaml            -->  7. Write skills (SKILL.md)     
                                     8. Define named agents         
                                     9. ac get <agent> --json       
                                        (verify composition)        
                                    10. ac doctor (verify refs)     
                                    11. Share agent repo       -->  12. ac init (one-time)
                                                                    13. ac run <agent>
```
