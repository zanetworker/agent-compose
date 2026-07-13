# Running Agents

## Prerequisites

1. An OpenShell gateway running (local with podman or on a cluster via Helm)
2. `openshell` CLI installed and connected (`openshell status` shows Connected)
3. `ac` binary built (`make build`)
4. `ac init` run (creates config + auto-detects providers)

## Example 1: Claude Code via Vertex AI

```bash
# ac init already created the vertex provider from gcloud ADC

# Run (still needs manual policy update for oauth2, see upstream issues)
openshell sandbox create --name my-claude \
  --provider vertex \
  --env CLAUDE_CODE_USE_VERTEX=1 \
  --env CLOUD_ML_REGION=us-east5 \
  --env ANTHROPIC_VERTEX_PROJECT_ID=your-project-id \
  --env GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcloud-adc.json \
  --upload ~/.config/gcloud/application_default_credentials.json:/tmp/gcloud-adc.json \
  --auto-providers --no-tty

openshell policy update my-claude \
  --add-endpoint "us-east5-aiplatform.googleapis.com:443:read-write:rest:enforce" \
  --add-endpoint "oauth2.googleapis.com:443:read-write:rest:enforce" \
  --add-endpoint "statsig.anthropic.com:443:read-write:rest:enforce" \
  --binary /usr/local/bin/claude

openshell sandbox exec --name my-claude --no-tty -- \
  claude -p "Say hello" --max-turns 1 --dangerously-skip-permissions
```

With agent-compose: `ac run --runtime claude-code-vertex --prompt "Say hello"`

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
