# GCE Metadata Emulator Not Running with google-vertex-ai Provider

**Component:** openshell-sandbox
**Severity:** High (blocks all GCP SDK consumers inside sandboxes)

## Summary

When a `google-vertex-ai` provider is attached to a sandbox, the GCE metadata emulator does not start. GCP SDKs (Python `google.auth`, Node.js `google-auth-library`, Go `cloud.google.com/go`) cannot obtain credentials via the standard Application Default Credentials (ADC) discovery flow.

## Evidence

```bash
# Create sandbox with vertex provider
openshell sandbox create --name test --provider vertex --no-tty

# Check env: GCE_METADATA_HOST is NOT set
openshell sandbox exec --name test -- env | grep GCE_METADATA
# (empty)

# Metadata endpoint returns policy_denied
openshell sandbox exec --name test -- \
  curl -s -H "Metadata-Flavor: Google" \
  http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token
# {"detail":"GET 169.254.169.254:80/... not permitted by policy","error":"policy_denied"}
```

## Expected Behavior

When a `google-vertex-ai` or `google-cloud` provider is attached:
1. The metadata emulator should start as a loopback HTTP server inside the sandbox
2. `GCE_METADATA_HOST` should be set to the loopback address
3. GCP SDKs calling `google.auth.default()` should discover the emulator and obtain tokens

The code for this exists in `crates/openshell-sandbox/src/google_cloud_metadata.rs` and `metadata_server.rs` in upstream OpenShell, but it is not present in the current build (0.0.77-dev.1).

## Impact

All agents using GCP SDKs inside sandboxes are affected:
- **Claude Code via Vertex**: uses Node.js GCP SDK for auth, falls back to ADC file. Without the metadata emulator, requires manual `--upload` of the ADC file.
- **ADK / LangGraph agents**: use Python `google.auth.default()`. Would work transparently if the metadata emulator ran.
- **Any Go/Python/Node service calling Vertex AI**: standard GCP SDK auth fails.

## Current Workaround

Upload the ADC credentials file manually:

```bash
openshell sandbox create --name my-agent \
  --provider vertex \
  --env GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcloud-adc.json \
  --upload ~/.config/gcloud/application_default_credentials.json:/tmp/gcloud-adc.json
```

This leaks credentials as a file inside the sandbox, bypasses token refresh, and defeats the provider's credential isolation model.

## Additional Finding: Missing oauth2.googleapis.com in Vertex Profile

The `google-vertex-ai` provider profile declares endpoints for `*-aiplatform.googleapis.com` but not `oauth2.googleapis.com`. When agents refresh ADC tokens inside the sandbox, the egress policy blocks the token refresh request:

```
NET:OPEN [MED] DENIED /usr/local/bin/claude(43) -> oauth2.googleapis.com:443
[reason:endpoint oauth2.googleapis.com:443 is not allowed by any policy]
```

The workaround is a manual policy update:

```bash
openshell policy update my-sandbox \
  --add-endpoint "oauth2.googleapis.com:443:read-write:rest:enforce" \
  --binary /usr/local/bin/claude
```

The `google-vertex-ai` profile should include `oauth2.googleapis.com:443` in its endpoints since token refresh is required for any GCP auth flow.

## Environment

- OpenShell CLI: 0.0.66
- Gateway: 0.0.77-dev.1 (podman driver, macOS)
- Tested: 2026-07-12/13
