# End-to-End Test Results

All tests run against live OpenShell gateways (podman driver local + Kubernetes remote), July 2026.

## Harness: Claude Code via Vertex AI

OpenShell sandbox with Landlock, network proxy, egress policy. Claude Code called Vertex AI.

```
$ claude -p "Say hello in exactly 3 words"
Hello! Let's code.

$ claude -p "What is 2+2? Just the number"
4

$ claude -p "Review this code: def get_user(id): return db.execute(f'SELECT...')"
SQL injection: id parameter directly interpolated via f-string without parameterized query
```

Claude Code also used tools inside the sandbox:
- Created and read files (`/sandbox/test.txt`)
- Called GitHub API via `gh api` (returned OpenShell repo description and star count)
- Read skill reference files mounted via `--upload`

**Configuration required:** `google-vertex-ai` provider, Vertex env vars, ADC file upload, policy update for `oauth2.googleapis.com` + `us-east5-aiplatform.googleapis.com`.

## Framework: Python agent calling GPU cluster (qwen3-14b)

Python/curl inside a sandbox calling a vLLM model on a remote GPU cluster.

```
$ curl -sk $OPENAI_BASE_URL/chat/completions -d '{"model":"qwen3-14b",...}'
def add(a, b):
    return a + b
```

**Configuration required:** `OPENAI_BASE_URL` and `OPENAI_MODEL` env vars, policy update for the GPU cluster endpoint.

## Skills

Resolver assembles combined prompt (agent + skills). Reference files uploaded and readable.

```
$ ac get style-checker --json
{
  "prompt": "Check this code...\n\n# Code Style Review\n\nWhen reviewing...",
  "skill_mounts": [{"Source": ".../style-guide.md", "Target": "/workspace/skills/code-style/"}]
}

$ cat /sandbox/skills/security-review/owasp-top-10.md   (inside sandbox)
# OWASP Top 10 Quick Reference
1. Broken Access Control
...
```

## MCP: GitHub Provider

Provider attaches credentials and egress policy. Binary-level access control verified.

```
$ git ls-remote https://github.com/NVIDIA/OpenShell.git HEAD    (allowed)
94cdd697...  HEAD

$ curl -sv https://api.github.com/user                          (blocked)
CONNECT tunnel failed, response 403

$ curl -sv https://example.com                                  (blocked)
CONNECT tunnel failed, response 403

$ gh api /repos/NVIDIA/OpenShell --jq .stargazers_count          (with GH_TOKEN env var)
7666
```

## CLI

```
$ ac init
Created ~/.ac/config.yaml
Detecting local credentials...
  Google Cloud ADC found       created vertex provider
  GitHub token found           created github provider
Created 2 provider(s).

$ ac doctor
Runtimes: ok | Inference: ok | Agents: ok
OpenShell: gateway reachable, profiles detected

$ ac list
NAME              SANDBOX           STATUS   AGE
hello-1783881595  hello-1783881595  running  0s

$ ac stop hello-1783881595
Agent hello-1783881595 stopped
```

## Framework: ADK Agent calling gemma-3-12b-it (GPU cluster)

Python agent (`examples/adk-agent/agent.py`) inside a sandbox, calling gemma-3-12b-it served by vLLM on the GPU cluster via OpenAI-compatible API.

```
$ openshell sandbox exec --name adk-live -- python3 /sandbox/agent.py

Agent: calling gemma-3-12b-it at https://gemma-3-12b-it-user-nxu.apps.ocp.cloud.rhai-tmm.dev/v1
Prompt: Explain what an agent composition engine does in one sentence....

An agent composition engine automatically combines and orchestrates multiple
individual agents to create a more complex, capable agent capable of handling
more sophisticated tasks.
```

**What was configured:** `OPENAI_BASE_URL` and `OPENAI_MODEL` env vars, agent.py uploaded via `--upload`, prompt written to `/sandbox/prompt.md`, policy update to allow the GPU cluster endpoint for the python binary.

**Upload path gotcha:** `--upload file.py:/sandbox/agent.py` creates a directory `/sandbox/agent.py/agent.py` (OpenShell extracts tar into a directory). Use `--upload file.py:/sandbox/` to place the file directly at `/sandbox/file.py`.

## SDK

```
$ go test ./examples/ -v
--- PASS: TestSDK_ResolveAgent (0.00s)
--- PASS: TestSDK_ResolveWithOverrides (0.00s)
--- PASS: TestSDK_InlineAgent (0.00s)
```

## Known Upstream Gaps

See [upstream-issues/](upstream-issues/) for detailed write-ups with evidence.

1. **GCE Metadata Emulator Not Running** ([001](upstream-issues/001-metadata-emulator-not-running.md)): Vertex provider doesn't start the metadata emulator. GCP SDKs can't discover credentials. Workaround: upload ADC file.

2. **Provider env_vars Not Injected** ([002](upstream-issues/002-provider-env-vars-not-injected.md)): Credentials injected under internal name (`api_token`), not declared env var names (`GITHUB_TOKEN`). CLI tools fail. Workaround: pass `--env GH_TOKEN=...` manually.
