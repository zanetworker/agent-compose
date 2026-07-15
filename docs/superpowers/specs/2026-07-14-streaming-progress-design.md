# Streaming Output, Progress Indicators, Early Sandbox Name

## Problem

`ac run` is a black box. The user sees nothing during the 45s lifecycle (sandbox creation, policy update, agent execution). Agent output is buffered and only shown on success or included in error messages. The sandbox name is printed only after everything finishes, preventing concurrent `ac logs` / `ac stop` from another terminal.

Additionally, `CLIExecutor` hardcodes `os.Stdout`, `os.Stderr`, and `os.Stdin`, creating coupling that will break when switching to the upcoming Go SDK executor (`openshell-sdk-go`).

## Design

### Injectable I/O on CLIExecutor

Add `stdin io.Reader`, `stdout io.Writer`, `stderr io.Writer` fields to `CLIExecutor`. All methods that interact with the terminal use these instead of the `os` package directly.

```go
type CLIExecutor struct {
    binary string
    stdin  io.Reader
    stdout io.Writer
    stderr io.Writer
}

func NewCLIExecutor(binary string, stdin io.Reader, stdout, stderr io.Writer) *CLIExecutor
```

Affected methods:
- `ExecInSandbox`: streams to `e.stdout`/`e.stderr` directly (no longer uses `run()`)
- `ConnectSandbox`: uses `e.stdin`/`e.stdout`/`e.stderr`
- `run()`: uses `e.stdout` for success output instead of `fmt.Println`

The `Executor` interface is unchanged. The streaming destination is a construction concern, not an interface concern. Future `SDKExecutor` follows the same pattern.

### Progress Writer on Engine

Add `progress io.Writer` to `Engine`, defaulting to `io.Discard`. New option: `WithProgress(w io.Writer)`.

`Engine.Run` emits progress to stderr:
```
Creating sandbox reviewer-1752000000...
Updating egress policy...
Running agent...
```

"Updating egress policy..." is conditional (only when egress rules exist). The 12s policy propagation delay is covered by this message.

### CLI Wiring

`buildEngine()` passes:
- `NewCLIExecutor(binary, os.Stdin, os.Stdout, os.Stderr)`
- `WithProgress(os.Stderr)`

The post-execution print in `run.go` ("Agent X running in sandbox Y") is removed. The progress message "Creating sandbox X..." already communicates the sandbox name, and it appears immediately rather than after execution finishes.

### Output Separation

Progress goes to stderr, agent output goes to stdout. Users can pipe agent output cleanly:

```
ac run reviewer --prompt "Review auth" 2>/dev/null | jq
```

## Files Changed

| File | Change |
|------|--------|
| `pkg/compose/executor_cli.go` | Add stdin/stdout/stderr fields. Update ExecInSandbox, ConnectSandbox, run(). |
| `pkg/compose/engine.go` | Add progress field. Emit progress in Run(). |
| `pkg/compose/options.go` | Add WithProgress option. |
| `cmd/ac/main.go` | Update NewCLIExecutor call, add WithProgress(os.Stderr). |
| `cmd/ac/run.go` | Remove post-exec print. |

## Tests

- `TestCLIExecutor_ExecInSandbox_StreamsOutput`: runs a real subprocess (`echo`), verifies output appears on the injected stdout writer, not os.Stdout.
- `TestEngine_Run_Progress`: uses DryRunExecutor with a progress writer, verifies messages appear in order: "Creating sandbox", "Running agent".
- `TestEngine_Run_Progress_WithEgress`: same but with egress rules, verifies "Updating egress policy" appears between create and run.
- Existing tests pass unchanged (DryRunExecutor is unaffected, progress defaults to Discard).

## What Does Not Change

- `Executor` interface (narrow, SDK-ready)
- `DryRunExecutor` (already uses injected `io.Writer`)
- `run()` helper continues to buffer for CreateSandbox/UpdatePolicy/DeleteSandbox
- 12s policy sleep stays in CLIExecutor.UpdatePolicy
- All existing tests
