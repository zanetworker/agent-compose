# Provider env_vars Field Not Injected as Sandbox Environment Variables

**Component:** openshell-sandbox, openshell-server
**Severity:** Medium (blocks CLI tools that check auth locally)

## Summary

Provider profiles declare an `env_vars` field that maps credentials to environment variable names (e.g., `env_vars: [GITHUB_TOKEN, GH_TOKEN]`). However, credentials are injected under the credential's internal name (e.g., `api_token`), not the declared env var names. CLI tools that check for specific env vars before making network requests fail because they don't see the expected variable.

## Evidence

```bash
# GitHub provider profile declares:
#   credentials:
#     - name: api_token
#       env_vars: [GITHUB_TOKEN, GH_TOKEN]

# Create provider and sandbox
openshell provider create --type github --name github --credential "api_token=ghp_..."
openshell sandbox create --name test --provider github --no-tty

# Check what's injected
openshell sandbox exec --name test -- env | grep -i github
# api_token=openshell:resolve:env:v..._api_token

# Expected: GITHUB_TOKEN=openshell:resolve:env:... or GH_TOKEN=openshell:resolve:env:...
# Got: api_token=openshell:resolve:env:...

# gh CLI fails (checks GH_TOKEN locally before making requests)
openshell sandbox exec --name test -- gh api /user --jq .login
# Error: To get started with GitHub CLI, please run: gh auth login
# Alternatively, populate the GH_TOKEN environment variable...

# Workaround: pass env var manually
openshell sandbox exec --name test --env "GH_TOKEN=ghp_..." -- gh api /user --jq .login
# zanetworker  (works)
```

## Expected Behavior

When a provider is attached to a sandbox, credentials should be injected under ALL env var names declared in the profile's `env_vars` field, not just the credential's internal name.

For the `github` profile with `env_vars: [GITHUB_TOKEN, GH_TOKEN]`:
- `GITHUB_TOKEN` should be set (primary)
- `GH_TOKEN` should be set (alias)
- `api_token` can optionally be set (internal name)

The proxy-level header injection (Authorization header on outbound requests) works correctly. The gap is env var injection for tools that check auth locally.

## Impact

Any CLI tool that checks for an auth env var before making network requests:
- `gh` (requires `GH_TOKEN` or `GITHUB_TOKEN`)
- `aws` CLI (requires `AWS_ACCESS_KEY_ID`)
- `gcloud` (requires various `GOOGLE_*` env vars or ADC file)
- `npm` (requires `NPM_TOKEN` for private registries)

The proxy handles auth transparently for tools that don't check locally (like `curl`, `git` for public repos). But most CLI tools validate auth before making requests.

## Current Workaround

Pass the credential as an env var manually:

```bash
GH_TOKEN=$(gh auth token)
openshell sandbox exec --name test --env "GH_TOKEN=$GH_TOKEN" -- gh api /user
```

This works but defeats the provider's purpose (the user must know the credential value and pass it explicitly).

## Design Note

The `env_vars` field already exists on every provider profile and is populated. The credential value is already stored in the provider. The fix is to inject credentials under their declared `env_vars` names in addition to proxy-level header injection.

The proxy-level injection should remain (it handles non-CLI network requests). The env var injection is additive.

## Environment

- OpenShell CLI: 0.0.66
- Gateway: 0.0.77-dev.1 (podman driver, macOS)
- Tested: 2026-07-13
