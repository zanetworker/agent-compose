# Running Agents

## Prerequisites

1. An OpenShell gateway running (local with podman or on a cluster via Helm)
2. `openshell` CLI installed and connected (`openshell status` shows Connected)
3. `ac` binary built (`make build`)
4. `ac init` run (creates config + auto-detects providers)

## Example 1: Claude Code via Vertex AI

```bash
# ac init already created the google-vertex-ai and google-cloud providers from gcloud ADC
# The engine auto-adds Vertex egress via UpdatePolicy after CreateSandbox

ac run --runtime claude-code-vertex --prompt "Say hello" --skip-permissions
```

**What the engine does:**
1. Creates sandbox with `google-vertex-ai` and `google-cloud` providers (from runtime config)
2. Calls `UpdatePolicy` to add `${region}-aiplatform.googleapis.com:443` (from ResolvedSpec)
3. Waits ~10s for policy propagation
4. Executes: `claude -p "Say hello" --dangerously-skip-permissions`

## Example 2: Custom agent against vLLM (GPU cluster)

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
    endpoint: https://qwen3-14b.apps.cluster.dev/v1
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

## Example 3: ADK / LangGraph agent via Vertex AI

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
ac run support
```
