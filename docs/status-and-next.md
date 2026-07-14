# agent-compose: Status and What's Next

## What's Done

### Core Engine (pkg/compose)
- [x] RuntimeResolver, InferenceResolver, MCPResolver, SkillResolver, PolicyResolver
- [x] N-var env-mapping template expansion (${endpoint}, ${model}, ${model.opus})
- [x] Skill prompt assembly with dependency dedup and reference file upload
- [x] ResolvedSpec with SandboxOpts (scope/mode/ttl)
- [x] OpenShell provider-based credential flow (no raw env vars for secrets)
- [x] UpdatePolicy after CreateSandbox (Vertex egress workaround for upstream #896)
- [x] Prompt delivery: -p flag for harness agents, /sandbox/prompt.md upload for framework agents
- [x] --inference and --model per-run overrides
- [x] Executor interface: CLIExecutor, DryRunExecutor, future SDKExecutor
- [x] No local run database (OpenShell labels as source of truth)

### CLI (cmd/ac)
- [x] ac init (auto-creates vertex + gcp + github providers from local credentials)
- [x] ac run (resolve + create + policy + exec)
- [x] ac run --dry-run (shows exact openshell commands)
- [x] ac run --interactive (sandbox connect)
- [x] ac run --skip-permissions (opt-in, not hardcoded)
- [x] ac run --workspace (uploads local directory)
- [x] ac run --model / --inference (per-run overrides)
- [x] ac get (resolved spec as JSON)
- [x] ac list / ac stop / ac logs
- [x] ac doctor (config + live gateway checks)
- [x] ac apply --sync-profiles

### Go SDK
- [x] engine.Resolve(), engine.Run(), engine.List(), engine.Stop(), engine.Get()
- [x] examples/sdk_test.go (3 tests pass)

### Testing
- [x] Unit tests: all pass (50+)
- [x] Live: Claude Code via Vertex (responded "4", security review, GitHub API)
- [x] Live: ADK agent via gemma-3-12b-it (GPU cluster)
- [x] Live: Skills (prompt assembly + reference file mount)
- [x] Live: MCP GitHub provider (egress policy, binary access control)
- [x] Live: ac init, ac doctor, ac list, ac stop

### Upstream
- [x] Rebuilt gateway to 0.0.83 (metadata emulator now works)
- [x] Identified: #896 (provider endpoint composition) OPEN, workaround in place
- [x] Identified: #1740 (upload creates directory) OPEN
- [x] Identified: #1706 (metadata emulator) CLOSED/FIXED

## What's Not Done

### P0: UX (blocks real usage)

**1. Stream agent output to user.**
`ac run` captures stdout but doesn't print it. The user sees "Created sandbox" and "Deleted sandbox" but never sees what the agent said. Fix: `ExecInSandbox` should stream stdout/stderr to the terminal in real-time, not capture into a buffer.

Files: `pkg/compose/executor_cli.go` (ExecInSandbox should use cmd.Stdout = os.Stdout, cmd.Stderr = os.Stderr instead of capturing), `cmd/ac/run.go` (print agent output).

**2. Progress indicators.**
Currently silent during: sandbox provisioning (30-60s), policy update (instant), policy propagation wait (12s), agent execution. User sees nothing.

Print: "Creating sandbox..." → "Updating egress policy..." → "Waiting for policy propagation (12s)..." → "Running agent..." → output → "Done."

Files: `pkg/compose/executor_cli.go` (add fmt.Fprintf to stderr for progress), `pkg/compose/engine.go` (add progress callbacks or just print to stderr).

**3. Async run + attach.**
`ac run` blocks for the entire lifecycle (create + policy + wait + exec). Should return the sandbox name immediately and let the user attach later.

```
ac run security-reviewer --workspace ./repo
# → Created sandbox security-reviewer-1784045591. Use ac attach to connect.

ac attach security-reviewer-1784045591          # connects to harness (claude)
ac attach security-reviewer-1784045591 --shell  # raw shell for debugging
ac logs security-reviewer-1784045591            # stream output
ac stop security-reviewer-1784045591            # cleanup
```

Files: new `cmd/ac/attach.go`, modify `cmd/ac/run.go` (default to async, add --wait for synchronous), modify `pkg/compose/engine.go` (split Run into CreateAgent + ExecAgent).

**4. Clean exit codes.**
Agent success/failure should propagate as exit codes. Currently all errors are exit 1 with cobra error text.

### P1: Correctness

**5. ExecInSandbox stdout should stream, not buffer.**
Currently `e.run()` captures all output into bytes.Buffer and only shows it on error. For agent execution, output should stream in real-time. The `run()` helper needs a streaming variant.

**6. Policy delay should be configurable or eliminated.**
The 12s sleep is a hack. Better: poll `openshell policy get <name>` until the new version is active, with a timeout.

**7. Provider instance vs profile type naming.**
Runtime profiles declare provider instance names (vertex, gcp, claude-code). If the user names their providers differently in ac init, the names won't match. Need either: convention-based naming (always use these names), or a mapping in config.

### P2: Features

**8. ac attach command.**
`ac attach <name>` calls `openshell sandbox exec --name <name> -- <entrypoint>` (connects to the harness). `ac attach <name> --shell` calls `openshell sandbox connect <name>` (raw shell). The engine stores the entrypoint in sandbox labels so attach knows what to run.

**9. Named agent config files.**
Currently agents are defined in config.yaml's `agents:` section. Support separate files: `agents/security-reviewer.yaml` that ac discovers automatically.

**10. ac run should print the sandbox name.**
Even in sync mode, print the sandbox name so the user can ac logs / ac attach / ac stop in another terminal.

**11. Governance basics.**
Add a `policy:` field to agent config for custom sandbox policies beyond the auto-generated egress rules.

### P3: Quality

**12. Crafted-code audit fixes.**
From the earlier audit:
- No test for CreateSandbox/ExecInSandbox error paths
- No test for agent env override invariant (agent.Env cannot override system env)
- Credential material for gcp provider still passes secrets as CLI args to `openshell provider refresh configure --material`
- `ListSandboxes` ignores `labelSelector` parameter
- `get` command has unused `--output` flag

**13. Naming decision.**
chaitin/agent-compose exists (200 stars, AGPL). If this goes public, rename.

**14. Documentation.**
- Tutorial Step 3 needs updating for the streaming output UX
- README Quick Start examples need to show actual agent output
- Architecture doc needs UpdatePolicy + policy delay section

## Recommended Priority for Next Session

1. Stream agent output (P0 #1 + #5) - makes ac run usable
2. Progress indicators (P0 #2) - user knows what's happening  
3. Print sandbox name (P2 #10) - user can ac logs in another terminal
4. ac attach (P0 #3 + P2 #8) - async workflow
5. Clean exit codes (P0 #4) - CI/CD usage
