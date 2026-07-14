"""Minimal ADK agent for testing with agent-compose.

This agent uses google-genai (not the full ADK framework) to keep
dependencies minimal. It reads a prompt from /sandbox/prompt.md if
present, otherwise uses a default. It calls the model once and prints
the response.

Works with any OpenAI-compatible endpoint (vLLM, MaaS) or Google
Vertex AI, depending on which env vars are set.
"""

import os
import json
import urllib.request
import ssl

def call_openai_compatible(endpoint, model, prompt):
    """Call an OpenAI-compatible chat completions endpoint."""
    data = json.dumps({
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": 200,
    }).encode()
    req = urllib.request.Request(
        f"{endpoint}/chat/completions",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    resp = urllib.request.urlopen(req, context=ctx, timeout=30)
    result = json.loads(resp.read())
    return result["choices"][0]["message"]["content"]

def main():
    # Read prompt from file (uploaded by agent-compose for framework agents)
    prompt_file = "/sandbox/prompt.md"
    if os.path.exists(prompt_file):
        with open(prompt_file) as f:
            prompt = f.read().strip()
    else:
        prompt = os.environ.get("AGENT_PROMPT", "Hello, what can you help me with?")

    # Determine which endpoint to use
    endpoint = os.environ.get("OPENAI_BASE_URL", "")
    model = os.environ.get("OPENAI_MODEL", "")

    if not endpoint or not model:
        print("Error: OPENAI_BASE_URL and OPENAI_MODEL must be set")
        print("Set these in your agent-compose config under inference:")
        exit(1)

    print(f"Agent: calling {model} at {endpoint}")
    print(f"Prompt: {prompt[:100]}...")
    print()

    response = call_openai_compatible(endpoint, model, prompt)
    print(response)

if __name__ == "__main__":
    main()
