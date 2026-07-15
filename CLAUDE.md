# agent-compose: Project Guide for Claude

## Testing Policy

Every new function, method, or package MUST have corresponding tests before the task is considered complete. Never claim work is done without:
1. Writing tests that cover the new/modified code
2. Running the project's test suite and confirming all tests pass
3. Verifying no regressions in existing tests

## Examples Are Documentation

The `examples/` directory contains runnable use-case tests that serve as living documentation for the Go SDK. They must stay in sync with the codebase.

**Before any commit that touches `pkg/compose/`:**
1. Run `go test ./examples/ -v` and verify all examples pass
2. If you changed a public API (added/removed/renamed a field, method, or type), update the affected examples
3. If you added a new feature, add an example demonstrating it in `examples/usecases_test.go`

**Examples to keep in sync:**
- `examples/sdk_test.go` - core SDK patterns (resolve, run, inline agent)
- `examples/usecases_test.go` - real-world use cases (batch review, MCP, custom runtime, background agents)

## Pre-Push Checklist

Before pushing, always run:
```bash
go test ./... -count=1
```

This covers unit tests, integration tests, AND examples. If any fail, fix before pushing.

## Key Patterns

- **Test-first**: Write failing test, implement, verify
- **Injectable I/O**: Never hardcode os.Stdout/Stderr/Stdin in pkg/compose/. Use constructor fields.
- **Executor interface**: All sandbox operations go through the Executor interface. Never call openshell directly from the engine.
- **DryRunExecutor for tests**: All examples use DryRunExecutor so they run without a gateway.
- **Progress on stderr**: User-facing progress goes to the engine's progress writer. Agent output goes to stdout.

## Building and Testing

```bash
go build -o /tmp/ac ./cmd/ac/    # Build
go test ./... -count=1            # All tests including examples
go test ./examples/ -v            # Just examples
```

## Environment

- `OPENSHELL_GATEWAY_INSECURE` must be UNSET when using the mTLS gateway (0.0.83+). It causes the CLI to skip client certificates.
- MCP config files go to `/sandbox/.ac-mcp.json` (not `.claude.json` which Claude overwrites).
- Policy binary paths must be full paths (`/usr/local/bin/claude`, `/usr/bin/node`), not short names.
