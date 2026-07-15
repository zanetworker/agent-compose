# Examples

Runnable examples showing how to use agent-compose as a Go library.

## Running

```bash
# All examples (uses DryRunExecutor, no gateway needed)
go test ./examples/ -v

# Specific example
go test ./examples/ -v -run TestSDK_BatchReviewMultipleRepos
```

## Examples

| Test | What it shows |
|------|---------------|
| `TestSDK_ResolveAgent` | Resolve a named agent to see its full spec (image, env, providers, prompt) before running |
| `TestSDK_ResolveWithOverrides` | Override inference provider and model at run time |
| `TestSDK_InlineAgent` | Compose an agent on the fly without a config.yaml entry |
| `TestSDK_BatchReviewMultipleRepos` | Fan out the same agent across multiple repos |
| `TestSDK_PreviewBeforeLaunch` | Dashboard pattern: resolve spec, show to user, then run |
| `TestSDK_ComposeWithMCP` | Declare MCP servers and verify the generated config |
| `TestSDK_CustomRuntime` | Define a framework agent runtime with custom image and entrypoint |
| `TestSDK_BackgroundAgent` | Start an agent in the background, check output later |

## Use Cases

**CI pipeline**: Resolve an agent spec, validate it, run it headless, check the exit code.

**Dashboard backend**: Call `engine.Resolve()` to preview what an agent will look like (providers, env, prompt). Show it in the UI. When the user clicks "Run", call `engine.Run()` or `engine.Start()`.

**Multi-repo review**: Loop over repos, call `engine.Start()` for each with a different workspace. Poll `engine.AgentOutput()` to collect results.

**Platform controller**: Load config from a CRD, construct an `Agent` struct, call `engine.Run()`. The executor interface (`CLIExecutor` today, `SDKExecutor` tomorrow) is the only thing that changes.
