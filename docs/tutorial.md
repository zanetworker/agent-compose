# Tutorial: Using agent-compose Step by Step

This tutorial walks through every feature of agent-compose, from zero to running composed agents in sandboxes. Each step builds on the previous one. You'll need a working OpenShell gateway before starting.

## Prerequisites

Verify these before starting:

```bash
openshell status          # should show Connected
go version                # should show 1.24+
```

At least one credential source:

```bash
# For Claude Code via Vertex AI:
gcloud auth application-default login
echo $CLOUD_ML_REGION          # should show your region (e.g., us-east5)

# For GitHub MCP access:
gh auth status                 # should show logged in

# For Claude Code via direct API (alternative to Vertex):
echo $ANTHROPIC_API_KEY        # should show sk-...
```

## Step 1: Build and Initialize

```bash
cd /path/to/agent-compose
make build

# Initialize: creates config + auto-detects credentials
./ac init
```

Expected output:

```
Created ~/.ac/config.yaml
Detecting local credentials...
  Google Cloud ADC found       created vertex provider
  GitHub token found           created github provider
  Anthropic API key            not set (using Vertex? That's fine)
Created 2 provider(s).
```

Verify:

```bash
./ac doctor
```

You should see all config checks pass and the gateway marked as reachable.

## Step 2: Your First Dry Run

Before creating real sandboxes, use `--dry-run` to see exactly what openshell commands agent-compose would generate:

```bash
./ac run --runtime claude-code-vertex --prompt "Say hello" --dry-run
```

Expected output (two openshell commands):

```
openshell sandbox create --name inline-... \
  --from ghcr.io/nvidia/openshell-community/sandboxes/base:latest \
  --auto-providers --no-tty \
  --provider google-vertex-ai \
  --env CLAUDE_CODE_USE_VERTEX=1 \
  --scope session --mode all --ttl 30m \
  --label agent=inline-... --label agentctl.io/agent=inline-...
openshell sandbox exec --name inline-... -- claude -p Say hello
```

This shows: the image, providers attached, env vars set, sandbox opts, and the entrypoint with the prompt delivered via `-p`.

## Step 3: Run Claude Code in a Sandbox (Headless)

This creates a real sandbox and runs Claude Code inside it:

```bash
./ac run --runtime claude-code-vertex \
         --prompt "What is 2+2? Just the number." \
         --skip-permissions
```

If this is your first run, the sandbox takes 30-60 seconds to provision (image pull). The agent responds and the sandbox is cleaned up.

**Note:** If you get a policy error about `oauth2.googleapis.com`, you need to update the sandbox policy (known upstream gap):

```bash
openshell policy update <sandbox-name> \
  --add-endpoint "us-east5-aiplatform.googleapis.com:443:read-write:rest:enforce" \
  --add-endpoint "oauth2.googleapis.com:443:read-write:rest:enforce" \
  --binary /usr/local/bin/claude
```

## Step 4: Run Claude Code Interactively

Instead of a one-shot prompt, open an interactive terminal inside the sandbox:

```bash
./ac run --runtime claude-code-vertex --interactive
```

This creates the sandbox and drops you into `openshell sandbox connect`. You get a shell inside the sandboxed environment where Claude Code is available. Type `claude` to start a session. Press `Ctrl-D` or `exit` to leave.

## Step 5: Configure an MCP Server

MCP servers give agents access to external tools (GitHub, Jira, Slack). Add GitHub to your config:

```yaml
# Add to ~/.ac/config.yaml under the mcp: section
mcp:
  github:
    provider: github
    egress:
      - api.github.com:443
      - github.com:443
```

Compose an agent with GitHub access:

```bash
./ac run --runtime claude-code-vertex \
         --mcp github \
         --prompt "What is the description of the NVIDIA/OpenShell repo?" \
         --skip-permissions \
         --dry-run
```

The dry-run should show `--provider github` in the sandbox create command.

## Step 6: Create a Skill

Skills are reusable prompt bundles with optional reference files. They can declare MCP and tool dependencies that get auto-merged into the agent's config.

Create a security review skill:

```bash
mkdir -p ~/.ac/skills/security-review/references

cat > ~/.ac/skills/security-review/SKILL.md << 'EOF'
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

Reference the OWASP top 10 at /sandbox/skills/security-review/owasp-top-10.md
EOF

cat > ~/.ac/skills/security-review/references/owasp-top-10.md << 'EOF'
# OWASP Top 10 (2025)
1. Broken Access Control
2. Cryptographic Failures
3. Injection
4. Insecure Design
5. Security Misconfiguration
6. Vulnerable and Outdated Components
7. Identification and Authentication Failures
8. Software and Data Integrity Failures
9. Security Logging and Monitoring Failures
10. Server-Side Request Forgery (SSRF)
EOF
```

This skill declares `requires.mcp: [github]`, which is why we configured the GitHub MCP server in Step 5 first. When the agent uses this skill, the GitHub provider is attached automatically.

Verify the skill is found:

```bash
./ac run --runtime claude-code-vertex \
         --skills security-review \
         --prompt "Review this: def login(u,p): return db.execute(f\"SELECT * FROM users WHERE name='{u}'\")" \
         --dry-run
```

The dry-run output should show:
- `--provider github` (auto-added from the skill's `requires.mcp`)
- `--upload .../owasp-top-10.md:/workspace/skills/security-review/` (reference file)
- The exec command includes the agent prompt AND the skill prompt assembled together

## Step 7: Configure an Inference Provider

Add a custom inference endpoint (e.g., a vLLM model on a GPU cluster):

```bash
cat >> ~/.ac/config.yaml << 'EOF'

inference:
  gpu-cluster:
    endpoint: https://qwen3-14b.apps.your-cluster.com/v1
    provider: gpu-vllm
    default-model: qwen3-14b
    egress:
      - qwen3-14b.apps.your-cluster.com:443
EOF
```

Test with a model override:

```bash
./ac run --runtime claude-code-vertex \
         --inference gpu-cluster \
         --model qwen3-14b \
         --prompt "Hello" \
         --dry-run
```

The dry-run should show the inference endpoint in the env vars and `--provider gpu-vllm` attached.

## Step 8: Define a Named Agent

Instead of passing flags every time, define a reusable agent in config:

```bash
cat >> ~/.ac/config.yaml << 'EOF'

agents:
  security-reviewer:
    runtime: claude-code-vertex
    mcp: [github]
    skills: [security-review]
    prompt: "You are a security-focused code reviewer. Be thorough but concise."

  quick-review:
    runtime: claude-code-vertex
    prompt: "Quick code review, no deep security scan."
EOF
```

Now run by name:

```bash
# See the full resolved spec
./ac get security-reviewer

# Dry-run
./ac run security-reviewer --workspace ./my-project --dry-run

# Run for real
./ac run security-reviewer --workspace ./my-project --skip-permissions

# Override model for this run only
./ac run security-reviewer --model llama-3.3-70b --dry-run
```

## Step 9: Compose Ad Hoc

You don't need a named agent. Compose everything inline:

```bash
./ac run --runtime claude-code-vertex \
         --mcp github \
         --skills security-review \
         --prompt "Review PR #42 for auth bypass vulnerabilities" \
         --workspace ./my-project \
         --skip-permissions
```

This picks from the menu the platform engineer defined (runtimes, inference, MCP servers) and composes on the fly.

## Step 10: Lifecycle Management

```bash
# List running sandboxes
./ac list

# View logs
./ac logs <sandbox-name>

# Stop and delete
./ac stop <sandbox-name>

# Validate everything
./ac doctor
```

## Step 11: Use the Go SDK

Everything the CLI does is available programmatically:

```go
package main

import (
    "context"
    "fmt"
    "github.com/zanetworker/agent-compose/pkg/compose"
)

func main() {
    cfg, _ := compose.LoadConfig("~/.ac/config.yaml")
    engine := compose.New(
        compose.WithConfig(cfg),
        compose.WithExecutor(compose.NewCLIExecutor("openshell")),
        compose.WithSkillsDir("~/.ac/skills"),
    )

    // Preview what would be created
    spec, _ := engine.Resolve(context.Background(), "security-reviewer")
    fmt.Printf("Image: %s\n", spec.Image)
    fmt.Printf("Providers: %v\n", spec.Providers)
    fmt.Printf("Prompt: %s\n", spec.Prompt[:50])

    // Or run it
    run, _ := engine.Run(context.Background(), "security-reviewer", compose.RunOpts{
        Workspace:       "./my-project",
        SkipPermissions: true,
    })
    fmt.Printf("Running in sandbox: %s\n", run.Sandbox)
}
```

Run the included SDK tests to see more examples:

```bash
go test ./examples/ -v
```

## Step 12: Run a Framework Agent (ADK Example)

Harness agents (Claude Code, Codex, Goose) are pre-installed in the OpenShell base image. Framework agents are different: you bring your own code, and agent-compose handles the infrastructure (inference endpoint, credentials, prompt, sandbox).

A working example is included at `examples/adk-agent/agent.py`. It's a minimal Python agent that reads a prompt from `/sandbox/prompt.md`, calls an OpenAI-compatible inference endpoint, and prints the response.

**Step 12a: Add the framework runtime and inference to your config**

```bash
cat >> ~/.ac/config.yaml << 'EOF'

runtimes:
  adk-agent:
    kind: framework
    image: ghcr.io/nvidia/openshell-community/sandboxes/base:latest
    env-mapping:
      OPENAI_BASE_URL: "${endpoint}"
      OPENAI_MODEL: "${model}"
    entrypoint: ["python3", "/sandbox/agent.py"]

inference:
  gpu-cluster:
    endpoint: https://qwen3-14b.apps.your-cluster.com/v1
    provider: ""
    default-model: qwen3-14b
    egress:
      - qwen3-14b.apps.your-cluster.com:443

agents:
  my-adk-agent:
    runtime: adk-agent
    inference: gpu-cluster
    prompt: "Explain what an agent composition engine does in two sentences."
EOF
```

Replace the inference endpoint with a real one you have access to. Any OpenAI-compatible endpoint works (vLLM, MaaS, Ollama, etc.).

**Step 12b: Dry-run to see the composition**

```bash
./ac run my-adk-agent --workspace ./examples/adk-agent --dry-run
```

Expected output shows:
- `--from ghcr.io/nvidia/openshell-community/sandboxes/base:latest` (base image with Python)
- `--env OPENAI_BASE_URL=https://...` (inference endpoint from config)
- `--env OPENAI_MODEL=qwen3-14b` (model from config)
- `--upload .../agent.py:...` (your agent code uploaded from workspace)
- `--upload .../prompt.md:...` (prompt written and uploaded by agent-compose)
- `openshell sandbox exec ... -- python3 /sandbox/agent.py` (entrypoint)

**Step 12c: Inspect the resolved spec**

```bash
./ac get my-adk-agent
```

```json
{
  "runtime_kind": "framework",
  "image": "ghcr.io/nvidia/openshell-community/sandboxes/base:latest",
  "entrypoint": ["python3", "/sandbox/agent.py"],
  "env": {
    "OPENAI_BASE_URL": "https://qwen3-14b.apps.your-cluster.com/v1",
    "OPENAI_MODEL": "qwen3-14b"
  },
  "prompt": "Explain what an agent composition engine does in two sentences.",
  "sandbox": {"scope": "session", "mode": "all", "ttl": "30m"}
}
```

**Step 12d: Run it**

```bash
./ac run my-adk-agent --workspace ./examples/adk-agent
```

The agent code is uploaded into the sandbox, the prompt is written to `/sandbox/prompt.md`, the inference env vars are set, and `python3 /sandbox/agent.py` runs inside the sandboxed environment.

**Step 12e: Override the model**

```bash
./ac run my-adk-agent --workspace ./examples/adk-agent --model llama-3.3-70b
```

Same agent, different model. The `OPENAI_MODEL` env var changes; everything else stays the same.

**How framework agents differ from harness agents:**

| | Harness (claude-code) | Framework (adk-agent) |
|---|---|---|
| Image | Base image (agent pre-installed) | Base image or custom (your code uploaded) |
| Prompt delivery | `-p` flag on the entrypoint | Uploaded as `/sandbox/prompt.md` |
| Code | Already in the image | Uploaded via `--workspace` |
| Entrypoint | `claude`, `codex`, `goose` | Your command (`python3 agent.py`) |
| Inference | Anthropic/OpenAI API via provider | Any OpenAI-compatible endpoint |

## What Each Command Does

| Command | What it does |
|---|---|
| `ac init` | Creates `~/.ac/config.yaml`, auto-detects credentials, creates OpenShell providers |
| `ac doctor` | Validates config references, checks gateway reachability, verifies providers exist |
| `ac run <name>` | Resolves agent config, creates sandbox, executes agent |
| `ac run --dry-run` | Shows the exact openshell commands without executing |
| `ac run -i` | Creates sandbox and opens interactive terminal |
| `ac get <name>` | Shows the fully resolved spec as JSON (providers, env, prompt, mounts) |
| `ac list` | Lists running sandboxes |
| `ac stop <name>` | Deletes a sandbox |
| `ac logs <name>` | Streams sandbox output |
| `ac apply --sync-profiles` | Pushes custom inference/MCP profiles to the OpenShell gateway |

## Troubleshooting

**`ac init` fails to create providers:**
The gateway must be running and connected. Check `openshell status`.

**Sandbox creation takes 60+ seconds:**
First run pulls the base image (~3 GB). Subsequent runs are faster.

**`policy_denied` errors from inside the sandbox:**
The sandbox's egress policy blocks requests. Either attach the right provider (`--provider github`) or update the policy manually with `openshell policy update`.

**Claude Code says "model not found":**
The `ANTHROPIC_DEFAULT_SONNET_MODEL` env var may have an invalid model name for your auth method (API vs Vertex). Remove the model env var or set it to a valid value.

**`gh` CLI says "run gh auth login" inside sandbox:**
OpenShell injects credentials at the proxy level, but `gh` checks for `GH_TOKEN` locally. Pass it explicitly: `--env GH_TOKEN=$(gh auth token)` on the openshell sandbox exec. This is a known upstream gap.
