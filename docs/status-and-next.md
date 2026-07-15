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
- [x] Resolved: CONNECT policy denial was misconfiguration, not upstream bug (#2272 CLOSED)
  - Policy `--binary` needs full paths (e.g., `/usr/local/bin/claude`, `/usr/bin/node`)
  - `OPENSHELL_GATEWAY_INSECURE=true` skips mTLS client certs; must be unset for mTLS gateways
  - Added `binaries` field to RuntimeProfile to declare full paths for egress policy rules

## What's Not Done

### P0: UX (blocks real usage)

**1. ~~Stream agent output to user.~~** DONE.
`ExecInSandbox` streams stdout/stderr directly to injected writers. `CLIExecutor` takes `stdin io.Reader, stdout, stderr io.Writer` at construction. `ConnectSandbox` and `run()` also use injected writers. No hardcoded `os.Stdout/Stderr/Stdin` in `pkg/compose/`. SDK-ready.

**2. ~~Progress indicators.~~** DONE.
`Engine` has `progress io.Writer` (via `WithProgress`, defaults to `io.Discard`). `Run()` emits to stderr: "Creating sandbox X..." then "Updating egress policy..." (conditional) then "Running agent...".

**3. ~~Async run + attach.~~** PARTIALLY DONE.
`ac attach <name>` connects to a running sandbox via `openshell sandbox connect`. Validates sandbox exists first (returns ErrNotFound if not running). Async run (splitting Run into CreateAgent+ExecAgent) is not yet implemented.

**4. ~~Clean exit codes.~~** DONE.
`ExitError` type wraps subprocess exit codes. `CLIExecutor.ExecInSandbox` extracts exit codes from `exec.ExitError`. `Engine.Run` propagates `ExitError` through error wrapping. `main.go` extracts the code via `errors.As` and calls `os.Exit(code)` instead of always `os.Exit(1)`.

### P1: Correctness

**5. ~~ExecInSandbox stdout should stream, not buffer.~~** DONE (see #1).

**6. Policy delay should be configurable or eliminated.**
The 12s sleep is a hack. Better: poll `openshell policy get <name>` until the new version is active, with a timeout.

**7. Provider instance vs profile type naming.**
Runtime profiles declare provider instance names (vertex, gcp, claude-code). If the user names their providers differently in ac init, the names won't match. Need either: convention-based naming (always use these names), or a mapping in config.

### P2: Features

**8. ~~ac attach command.~~** DONE (basic).
`ac attach <name>` connects to a running sandbox shell. Validates sandbox exists. Future: store entrypoint in labels to support harness-specific attach (re-enter Claude vs raw shell).

**9. Named agent config files.**
Currently agents are defined in config.yaml's `agents:` section. Support separate files: `agents/security-reviewer.yaml` that ac discovers automatically.

**10. ~~ac run should print the sandbox name.~~** DONE.
Progress message "Creating sandbox X..." prints the sandbox name to stderr immediately after it's computed, before creation starts. User can `ac logs` / `ac stop` from another terminal.

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

1. ~~Stream agent output (P0 #1 + #5) - makes ac run usable~~ DONE
2. ~~Progress indicators (P0 #2) - user knows what's happening~~ DONE
3. ~~Print sandbox name (P2 #10) - user can ac logs in another terminal~~ DONE
4. ~~ac attach (P0 #3 + P2 #8) - basic attach~~ DONE
5. ~~Clean exit codes (P0 #4) - CI/CD usage~~ DONE
6. Async run (split Engine.Run into CreateAgent + ExecAgent)
7. Policy delay elimination (P1 #6)
8. Named agent config files (P2 #9)
9. Crafted-code audit fixes (P3 #12)
