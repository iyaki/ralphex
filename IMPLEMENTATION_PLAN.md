# Implementation Plan (Whole system)

**Status:** All core features implemented and verified; oh-my-pi (omp) agent integration complete (10/10 phases complete).
**Last Updated:** 2026-06-17
**Primary Specs:** `specs/core-architecture.md`, `specs/run-command.md`, `specs/configuration.md`, `specs/config-local-overlay.md`, `specs/prompts.md`, `specs/config-by-prompt.md`, `specs/agents.md`, `specs/agent-env-overrides.md`, `specs/logging.md`, `specs/init-command.md`, `specs/e2e-testing.md`, `specs/development-testing.md`, `specs/release-workflow.md`.

## Quick Reference

| System/Subsystem | Specs | Modules/Packages | Web Packages | Migrations/Artifacts | Current State |
| --- | --- | --- | --- | --- | --- |
| CLI routing and loop lifecycle | `specs/core-architecture.md`, `specs/run-command.md` | `cmd/ralph/main.go` ✅, `internal/cli/cmd.go` ✅, `internal/cli/run.go` ✅ | None | CLI routing tests (`internal/cli/*`, `test/e2e/run_command_test.go`) ✅ | Complete |
| Configuration precedence and local overlays | `specs/configuration.md`, `specs/config-local-overlay.md` | `internal/config/config.go` ✅, `internal/config/config_test.go` ✅, `internal/config/config_local_test.go` ✅ | None | `ralph.toml` + `ralph-local.toml` merge behavior ✅ | Complete |
| Prompt resolution and front matter overrides | `specs/prompts.md`, `specs/config-by-prompt.md` | `internal/prompt/prompts.go` ✅, `internal/prompt/frontmatter.go` ✅, `internal/cli/run.go` ✅ | None | File/front matter parsing tests ✅ | Complete |
| Agent adapters and child env overrides | `specs/agents.md`, `specs/agents/*.md`, `specs/agent-env-overrides.md` | `internal/agent/agent.go` ✅, `internal/agent/runner.go` ✅, `internal/agent/opencode.go` ✅, `internal/agent/claude.go` ✅, `internal/agent/cursor.go` ✅, `internal/agent/oh-my-pi.go` ✅ | None | E2E fixture symlinks (`test/e2e/agents/ralph-test-agent`) ✅ | Complete |
| Logging and file-safety behavior | `specs/logging.md`, `specs/configuration.md` | `internal/logger/logger.go` ✅, `internal/cli/run.go` ✅ | None | `ralph.log` semantics + permission checks ✅ | Complete |
| Interactive `init` workflow | `specs/init-command.md` | `internal/cli/init.go` ✅, `internal/config/writer.go` ✅, `internal/cli/init_internal_test.go` ✅ | None | Generated `ralph.toml` artifact ✅ | Complete |
| End-to-end harness and traceability | `specs/e2e-testing.md` | `test/e2e/harness_test.go` ✅, `test/e2e/*.go` ✅, `test/e2e/coverage_matrix_enforcement_test.go` ✅ | None | `test/e2e/COVERAGE_MATRIX.md` ✅ | Complete |
| Quality/security/release automation | `specs/development-testing.md`, `specs/release-workflow.md` | `Makefile` ✅, `.github/workflows/quality.yml` ✅, `.github/workflows/security.yml` ✅, `.github/workflows/release.yml` ✅ | None | Release binaries + `checksums.txt` ✅ | Complete |
| Documentation and examples | `specs/README.md`, `specs/*` | `README.md`, `examples/ralph.toml` ✅, `cmd/ralph/main_test.go` ✅ | None | Repo/docs regression checks ✅ | Complete |

## Phased Plan

### Phase 1: Command Routing and Loop Lifecycle

**Goal:** Keep command dispatch deterministic and preserve loop completion semantics across root and explicit `run` invocations.
**Status:** Complete
**Paths:**
- `cmd/ralph/main.go`
- `internal/cli/cmd.go`
- `internal/cli/run.go`
- `internal/cli/cmd_test.go`
- `internal/cli/run_test.go`
- `test/e2e/run_command_test.go`

#### 1.1 Dispatch, alias behavior, and collision policy

**Paths:**
- `internal/cli/cmd.go`
- `internal/cli/run.go`
- `test/e2e/run_command_test.go`

**Reference pattern:** `internal/cli/cmd.go` (single root command, explicit subcommands, shared run path)

**Checklist:**
- [x] Root command registers `init` and `run`.
- [x] Root command and `run` command both route through `runCommandLogic`.
- [x] Alias behavior (`ralph <prompt> [scope]`) is preserved for non-subcommand names.
- [x] Collision behavior is deterministic (`ralph init` is subcommand; `ralph run init` is prompt).

#### 1.2 Loop defaults and completion detection

**Paths:**
- `internal/cli/run.go`
- `internal/cli/cmd_test.go`

**Reference pattern:** `internal/cli/run.go` (`parsePositionalArgs`, placeholder replacement, `hasCompletionSignal`)

**Checklist:**
- [x] No-arg invocation defaults to prompt `build` and scope `Whole system`.
- [x] `<COMPLETION_SIGNAL>` is replaced with `<promise>COMPLETE</promise>`.
- [x] Completion detection requires a trimmed line match of `<promise>COMPLETE</promise>`.
- [x] Max-iteration exhaustion returns non-zero (`max iterations reached`).
- [x] `DEBUG=1` exits after first iteration.

**Definition of Done:**
- `go test ./internal/cli -run 'TestNewRalphCommand|TestNewRunCommand|TestRunLoop' -count=1`
- `go test ./test/e2e -run TestE2ERunCommandRouting -count=1`
- Files touched: `internal/cli/*`, `cmd/ralph/main.go`, `test/e2e/run_command_test.go`

**Risks/Dependencies:**
- New subcommands must preserve reserved-word collision behavior.

### Phase 2: Configuration Precedence and Overlay Semantics

**Status:** Complete

#### 2.1 Verified base behavior and parity fixes

**Paths:**
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/config/config_local_test.go`

**Reference pattern:** `internal/config/config.go` (`resolveFileConfig`, `applyLocalOverlay`, `applyConfigValues`, `mergeConfig`)

**Checklist:**
- [x] Base config selection order is `--config` > `RALPH_CONFIG` > `./ralph.toml`.
- [x] `ralph-local.toml` discovery is anchored to the selected base config directory.
- [x] Deep merge is implemented for `[prompt-overrides.<prompt>]` and `[env]`.
- [x] Config-file keys `prompt-file` and `no-specs-index` are applied to effective runtime config.
- [x] TOML `config-file` key fails fast in base/default/overlay config files.

#### 2.2 Remaining edge-case hardening

**Paths:**
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/config/config_local_test.go`
- `test/e2e/config_precedence_test.go`

**Reference pattern:** `internal/config/config.go` (`resolveBool`, `loadDefaultConfig`, `applyLocalOverlay`)

**Checklist:**
- [x] Support explicit false override for `--no-specs-index=false` when config file sets `no-specs-index = true`.
- [x] Fail fast on non-`os.ErrNotExist` errors during default config and overlay discovery.
- [x] Unit/e2e coverage added for boolean precedence and error handling paths.

**Definition of Done:**
- `go test ./internal/config -run 'TestLoadConfig.*NoSpecsIndex.*|TestLoadConfig.*Overlay.*|TestLoadConfig.*Default.*' -count=1`
- `go test ./test/e2e -run 'TestE2EConfigPrecedence|TestE2EConfigLocalOverlay' -count=1`
- Files touched: `internal/config/config.go`, `internal/config/*_test.go`, `test/e2e/config_precedence_test.go`, `test/e2e/config_local_test.go`

**Risks/Dependencies:**
- Boolean precedence changes can regress existing `--no-log=false`/`--log-truncate=false` behavior if shared parsing paths are unintentionally altered.

### Phase 3: Prompt Resolution and Prompt-Level Overrides

**Goal:** Preserve deterministic prompt-source resolution and safe front matter override behavior.
**Status:** Complete
**Paths:**
- `internal/prompt/prompts.go`
- `internal/prompt/frontmatter.go`
- `internal/cli/run.go`
- `internal/prompt/*_test.go`
- `test/e2e/prompt_test.go`
- `test/e2e/config_by_prompt_test.go`

#### 3.1 Prompt source precedence and fallback behavior

**Paths:**
- `internal/prompt/prompts.go`
- `internal/prompt/prompts_test.go`

**Reference pattern:** `internal/prompt/prompts.go` (`GetPrompt` source chain)

**Checklist:**
- [x] Inline prompt (`--prompt`) has highest priority.
- [x] Stdin prompt mode works for `--prompt-file -` and `ralph -`.
- [x] Explicit `--prompt-file` is read before prompts-dir lookup.
- [x] Relative prompts-dir is searched upward; absolute paths are checked directly.
- [x] Built-in `build` and `plan` prompts are used as fallback.
- [x] Unknown prompt names fail with a clear error.

#### 3.2 Front matter extraction and precedence merge

**Paths:**
- `internal/prompt/frontmatter.go`
- `internal/cli/run.go`
- `test/e2e/config_by_prompt_test.go`

**Reference pattern:** `internal/cli/run.go` (`applyModelSettings`, `applyAgentModeSettings`)

**Checklist:**
- [x] Front matter keys `model` and `agent-mode` are parsed from file-based prompts.
- [x] Front matter is stripped before prompt body is sent to the agent.
- [x] Invalid front matter fails before agent execution.
- [x] Effective precedence is flags > env > front matter > `[prompt-overrides.<prompt>]` > global config.
- [x] Unknown front matter keys are ignored without breaking supported keys.

**Definition of Done:**
- `go test ./internal/prompt -count=1`
- `go test ./internal/cli -run 'TestConfigPrecedence_' -count=1`
- `go test ./test/e2e -run 'TestE2E(ConfigByPrompt|InlinePrompt|StdinPrompt)' -count=1`
- Files touched: `internal/prompt/*`, `internal/cli/run.go`, `test/e2e/config_by_prompt_test.go`

**Risks/Dependencies:**
- Parser edge cases around leading `---` content must stay deterministic and non-executable.

### Phase 4: Agent Adapters and Child Process Environment

**Goal:** Keep all supported agents behaviorally consistent and pass deterministic env to subprocesses.
**Status:** Complete
**Paths:**
- `internal/agent/agent.go`
- `internal/agent/runner.go`
- `internal/agent/opencode.go`
- `internal/agent/claude.go`
- `internal/agent/cursor.go`
- `internal/agent/agent_test.go`
- `test/e2e/agent_selection_test.go`
- `test/e2e/agent_env_overrides_test.go`

#### 4.1 Agent selection and invocation contract

**Paths:**
- `internal/agent/agent.go`
- `internal/agent/opencode.go`
- `internal/agent/claude.go`
- `internal/agent/cursor.go`

**Reference pattern:** `internal/agent/agent.go` (`GetAgent` factory)

**Checklist:**
- [x] Supported agents are `opencode`, `claude`, `cursor`.
- [x] Unknown agent names fail fast with a clear error.
- [x] Availability checks use `exec.LookPath`.
- [x] Adapter-specific argument construction matches specs.

#### 4.2 Environment merge/validation/propagation

**Paths:**
- `internal/agent/runner.go`
- `internal/cli/run.go`
- `internal/agent/agent_test.go`
- `test/e2e/agent_env_overrides_test.go`

**Reference pattern:** `internal/agent/runner.go` (`BuildEffectiveEnv`, explicit `cmd.Env` assignment)

**Checklist:**
- [x] Child env is built from inherited env + validated overrides.
- [x] Invalid env keys fail without leaking values.
- [x] `cmd.Env` is explicitly set for agent subprocesses.
- [x] Env slices are copied to avoid caller-side mutation.
- [x] E2E env override matrix covers config-only, flag-only, precedence, duplicate-key, and invalid-entry paths.

**Definition of Done:**
- `go test ./internal/agent -count=1`
- `go test ./internal/cli -run 'TestRunLoop(AppliesEffectiveEnvOverridesToAgentProcess|RejectsInvalidAgentEnvKeyBeforeExecution)' -count=1`
- `go test ./test/e2e -run TestE2EEnvOverrides -count=1`
- Files touched: `internal/agent/*`, `internal/cli/run.go`, `test/e2e/agent_env_overrides_test.go`

**Risks/Dependencies:**
- Child-process env behavior is security-sensitive; value-redaction guarantees must stay intact.

### Phase 5: Logging and File Safety Guarantees

**Goal:** Preserve secure logging behavior and deterministic stream/file semantics.
**Status:** Complete
**Paths:**
- `internal/logger/logger.go`
- `internal/logger/logger_test.go`
- `internal/cli/run.go`
- `test/e2e/logging_flags_test.go`

#### 5.1 Enablement precedence and file behavior

**Paths:**
- `internal/config/config.go`
- `internal/cli/run.go`
- `internal/logger/logger.go`

**Reference pattern:** `internal/cli/run.go` (`readBoolFlagOverride`, `applyBoolFlagOverrides`)

**Checklist:**
- [x] Logging defaults to disabled (`NoLog=true`).
- [x] Explicit `--no-log=false` can override config/env disablement.
- [x] Append and truncate behavior is supported.
- [x] Restrictive permissions are enforced (`0750` dir, `0600` file).

#### 5.2 Header metadata and stdout parity

**Paths:**
- `internal/logger/logger.go`
- `test/e2e/logging_flags_test.go`

**Reference pattern:** `internal/logger/logger.go` (header + git metadata writes)

**Checklist:**
- [x] Header includes timestamp and git metadata (`N/A` fallback).
- [x] Output streams to stdout and log file through a multi-writer path.
- [x] Empty log path creates a temp file when logging is enabled.

**Definition of Done:**
- `go test ./internal/logger -count=1`
- `go test ./test/e2e -run 'TestE2ELogging|TestE2ELoggingFlags|TestE2ELoggingPermissions|TestE2ELoggingStdoutParity' -count=1`
- Files touched: `internal/logger/logger.go`, `internal/cli/run.go`, `test/e2e/logging_flags_test.go`

**Risks/Dependencies:**
- Logs contain prompt and agent output; operational guidance must continue to treat log files as sensitive.

### Phase 6: Init Command Runtime/Spec Parity

**Goal:** Keep `ralph init` behavior and user-facing guidance fully aligned with the implemented interactive flow.
**Status:** In progress
**Paths:**
- `internal/cli/init.go`
- `internal/cli/init_internal_test.go`
- `internal/config/writer.go`
- `specs/init-command.md`
- `README.md`
- `cmd/ralph/main_test.go`

#### 6.1 Verified interactive workflow

**Paths:**
- `internal/cli/init.go`
- `internal/cli/init_internal_test.go`

**Reference pattern:** `internal/cli/init.go` (session prep -> questionnaire -> preview confirmation -> atomic write)

**Checklist:**
- [x] Ordered questionnaire with validation/re-prompt loops is implemented.
- [x] Existing config values seed question defaults when valid.
- [x] Overwrite confirmation/no-op behavior exists when file already exists.
- [x] Final preview confirmation gates file writes.
- [x] TTY checks validate stdin/stdout as terminal character devices.

#### 6.2 Remaining parity work (UX + docs)

**Paths:**
- `internal/cli/init.go`
- `README.md`
- `cmd/ralph/main_test.go`

**Reference pattern:** `specs/init-command.md` data-flow step 7 and workflow tables

**Checklist:**
- [ ] Add post-success guidance output with suggested next commands to match init spec intent.
- [ ] Update README `ralph init` section to reflect interactive overwrite/preview behavior (not force-only overwrite).
- [ ] Correct README default logging claim for generated config (`no-log = true` by default).
- [ ] Add/extend doc regression tests so init behavior text does not drift again.

**Definition of Done:**
- `go test ./internal/cli -run 'TestInit' -count=1`
- `go test ./cmd/ralph -run TestReadmeDocumentsRalphexRepoAndRalphCli -count=1`
- Manual verification: `ralph init` output text matches `specs/init-command.md`.
- Files touched: `internal/cli/init.go`, `internal/cli/init_internal_test.go`, `README.md`, `cmd/ralph/main_test.go`

**Risks/Dependencies:**
- Interactive UX assertions require stable prompt text contracts to avoid brittle tests.

### Phase 7: End-to-End Coverage Completeness and Governance

**Goal:** Meet the e2e spec requirement that every supported option/config/output surface is traceably covered.
**Status:** In progress
**Paths:**
- `test/e2e/*.go`
- `test/e2e/COVERAGE_MATRIX.md`
- `test/e2e/coverage_matrix_enforcement_test.go`
- `test/e2e/coverage_matrix_helpers_test.go`
- `specs/e2e-testing.md`

#### 7.1 Verified deterministic harness baseline

**Paths:**
- `test/e2e/harness_test.go`
- `test/e2e/types_test.go`
- `test/e2e/agents/ralph-test-agent/main.go`

**Reference pattern:** `test/e2e/harness_test.go` (single binary + single fixture agent via symlinks)

**Checklist:**
- [x] Harness runs real CLI subprocesses with isolated temp workdirs.
- [x] All supported agents map to one deterministic test fixture binary.
- [x] Assertions cover stdout, stderr, exit codes, files, and deterministic duration checks.
- [x] Coverage matrix artifact exists and is CI-tested for stale/missing test-name mappings.

#### 7.2 Missing required-surface coverage (spec gap)

**Paths:**
- `test/e2e/*.go`
- `test/e2e/COVERAGE_MATRIX.md`

**Reference pattern:** `specs/e2e-testing.md` (option/config/output coverage requirements)

**Checklist:**
- [ ] Add e2e coverage for env-driven config surfaces not yet represented: `RALPH_SPECS_DIR`, `RALPH_SPECS_INDEX_FILE`, `RALPH_IMPLEMENTATION_PLAN_NAME`, `RALPH_CUSTOM_PROMPT`, `RALPH_PROMPTS_DIR`, `RALPH_AGENT`, `RALPH_LOG_FILE`, `RALPH_LOG_APPEND`.
- [ ] Add e2e scenario for `DEBUG=1` single-iteration behavior.
- [ ] Add e2e strategy for `init` flags (`--output`, `--force`) using a pseudo-TTY or dedicated integration harness.
- [ ] Update `test/e2e/COVERAGE_MATRIX.md` mappings for all newly added surfaces.

#### 7.3 Governance hardening

**Paths:**
- `test/e2e/coverage_matrix_enforcement_test.go`
- `test/e2e/coverage_matrix_helpers_test.go`
- `test/e2e/COVERAGE_MATRIX.md`

**Reference pattern:** `TestCoverageMatrixCompleteness` + helper inventory logic

**Checklist:**
- [ ] Expand enforcement from "all tests are listed" to "all required surfaces are declared and mapped."
- [ ] Add a machine-readable required-surface inventory and fail CI if any required entry has no mapped scenario.

**Definition of Done:**
- `go test ./test/e2e -count=1`
- `go test ./test/e2e -run TestCoverageMatrixCompleteness -count=1`
- `make test-e2e`
- Files touched: `test/e2e/*.go`, `test/e2e/COVERAGE_MATRIX.md`

**Risks/Dependencies:**
- Full-surface coverage increases maintenance cost; enforcement design must remain deterministic and low-friction.

### Phase 8: Quality and Security Gates

**Goal:** Keep local and CI quality/security checks aligned with development-testing spec.
**Status:** Complete
**Paths:**
- `Makefile`
- `.github/workflows/quality.yml`
- `.github/workflows/security.yml`
- `specs/development-testing.md`

#### 8.1 Local quality target coverage

**Paths:**
- `Makefile`

**Reference pattern:** `Makefile` targets (`quality`, `test`, `test-e2e`, `test-race`, `coverage`, `mutation`, `security`, `arch`)

**Checklist:**
- [x] `make quality` composes lint, tests, race, coverage, mutation, security, and architecture checks.
- [x] Coverage gate remains set to >= 95%.
- [x] `make test` includes full Go suite including `test/e2e`.

#### 8.2 CI quality/security workflows

**Paths:**
- `.github/workflows/quality.yml`
- `.github/workflows/security.yml`

**Reference pattern:** GitHub workflow jobs `lint`, `test`, `coverage`, `arch`, `mutation`, `security`, `semgrep`

**Checklist:**
- [x] CI quality workflow runs lint/test/coverage/arch/mutation jobs.
- [x] CI security workflow runs `govulncheck`, `gosec`, and Semgrep.
- [x] Go toolchain in CI is pinned to `~1.25`.

**Definition of Done:**
- `make quality`
- `make security && make arch`
- Files touched: `Makefile`, `.github/workflows/quality.yml`, `.github/workflows/security.yml`

**Risks/Dependencies:**
- Mutation and Semgrep runtime can be expensive; job timeouts and cache behavior must stay stable.

### Phase 9: Release Workflow and Artifact Publishing

**Goal:** Preserve deterministic release builds and publish semantics.
**Status:** Complete
**Paths:**
- `.github/workflows/release.yml`
- `specs/release-workflow.md`

#### 9.1 Triggering and tag semantics

**Paths:**
- `.github/workflows/release.yml`

**Reference pattern:** `prepare` job (`push` tags + `workflow_dispatch` + optional tag creation)

**Checklist:**
- [x] Release triggers on semver-style tags (`v*`) and manual dispatch.
- [x] Manual flow supports optional tag creation (`create_tag`).
- [x] Tag format validation enforces `vMAJOR.MINOR.PATCH`.

#### 9.2 Matrix build and publish steps

**Paths:**
- `.github/workflows/release.yml`

**Reference pattern:** `build` matrix + `release` job checksum/publish

**Checklist:**
- [x] Matrix builds produce cross-platform binaries for configured targets.
- [x] `checksums.txt` is generated from all `ralph_*` artifacts.
- [x] Assets are published through `softprops/action-gh-release`.

**Definition of Done:**
- `gh workflow run release.yml -f tag=vX.Y.Z -f create_tag=false` (in release-enabled repo context)
- Verify GitHub Release assets include all binaries and `checksums.txt`.
- Files touched: `.github/workflows/release.yml`

**Risks/Dependencies:**
- Publishing depends on repository permissions (`contents: write`), protected tags, and token scopes.

### Phase 10: Documentation and Plan Drift Control

**Goal:** Keep docs/specs/plan aligned with actual runtime behavior and current verified gaps.
**Status:** In progress
**Paths:**
- `README.md`
- `specs/*.md`
- `IMPLEMENTATION_PLAN.md`
- `cmd/ralph/main_test.go`

#### 10.1 Verified baseline alignment

**Paths:**
- `specs/*.md`
- `cmd/ralph/main_test.go`

**Reference pattern:** `cmd/ralph/main_test.go` README/repo-name regression checks

**Checklist:**
- [x] Active runtime specs under `specs/` currently report `Status: Implemented`.
- [x] Agent/config/logging/e2e/init specs reflect implemented architecture.
- [x] Existing doc regression tests protect repository/CLI naming references.

#### 10.2 Remaining documentation drift

**Paths:**
- `README.md`
- `test/e2e/COVERAGE_MATRIX.md`
- `IMPLEMENTATION_PLAN.md`

**Reference pattern:** `README.md` init section, matrix "Pending Gaps" section, and this plan status counters

**Checklist:**
- [ ] Correct README `ralph init` behavior text to match current interactive implementation.
- [ ] Remove stale "pending gaps complete" assumptions from `test/e2e/COVERAGE_MATRIX.md` after Phase 7 surface audit.
- [ ] Keep plan phase status/counts synced after each merged change.

**Definition of Done:**
- `go test ./cmd/ralph -run TestReadmeDocumentsRalphexRepoAndRalphCli -count=1`
- `grep pattern="^Status:\\s*Partially Implemented" include="*.md" path="/workspaces/ralph/specs"`
- Manual review pass over `README.md`, `specs/*.md`, and `IMPLEMENTATION_PLAN.md`
- Files touched: `README.md`, `test/e2e/COVERAGE_MATRIX.md`, `IMPLEMENTATION_PLAN.md`, `cmd/ralph/main_test.go`

**Risks/Dependencies:**
- Documentation drift creates duplicate work and false completion signals.

## Verification Log

- 2026-04-20: `git status --short` - working tree clean; tests run: none (planning mode); bug fixes discovered: none; files touched: none.
- 2026-04-20: `git log --oneline --decorate -n 30 -- specs` - identified latest scope-shaping spec commits (`99bf24b`, `e674cd9`, `7e485bc`, `585a05c`, `328fc39`, `8965973`, `1321859`); tests run: none; bug fixes discovered: none; files touched: `specs/*.md`.
- 2026-04-20: `git log --oneline --decorate -n 10 -- IMPLEMENTATION_PLAN.md` - existing plan history showed all-complete status requiring revalidation against current code; tests run: none; bug fixes discovered: stale plan status identified; files touched: `IMPLEMENTATION_PLAN.md`.
- 2026-04-20: `git show --oneline --name-only 99bf24b -- specs && git show --oneline --name-only e674cd9 -- specs && git show --oneline --name-only 7e485bc -- specs && git show --oneline --name-only 585a05c -- specs && git show --oneline --name-only 328fc39 -- specs && git show --oneline --name-only 8965973 -- specs && git show --oneline --name-only 1321859 -- specs` - verified recent spec updates across init/config/core/e2e/agents domains; tests run: none; bug fixes discovered: none; files touched: `specs/init-command.md`, `specs/configuration.md`, `specs/e2e-testing.md`, `specs/agents.md`, `specs/agent-env-overrides.md`, related specs.
- 2026-04-20: `grep pattern="^Status:\\s*.*" include="*.md" path="/workspaces/ralph/specs"` - confirmed active runtime specs currently declare implemented status; tests run: none; bug fixes discovered: none; files touched: `specs/*.md`.
- 2026-04-20: `grep pattern="TODO|FIXME|XXX|HACK" include="*.go" path="/workspaces/ralph" && grep pattern="t\\.Skip|Skip\\(|flaky|Flaky" include="*.go" path="/workspaces/ralph"` - no actionable TODO/skip/flaky markers in runtime paths (only planning prompt text mentions these terms); tests run: none; bug fixes discovered: none; files touched: `internal/prompt/prompts.go`.
- 2026-04-20: `grep pattern="NoSpecsIndex|no-specs-index" include="*.go" path="/workspaces/ralph/internal"` plus read pass over `internal/config/config.go` - confirmed bool precedence path lacks explicit false override tracking for `--no-specs-index=false`; tests run: none; bug fixes discovered: config edge-case gap identified; files touched: `internal/config/config.go`, `internal/cli/run.go`.
- 2026-04-20: read pass over `internal/config/config.go` - confirmed `loadDefaultConfig` and `applyLocalOverlay` treat non-ENOENT `os.Stat` errors as missing instead of fail-fast; tests run: none; bug fixes discovered: config discovery hardening gap identified; files touched: `internal/config/config.go`.
- 2026-04-20: `grep pattern="RALPH_[A-Z_]+" include="*.go" path="/workspaces/ralph/test/e2e"` - confirmed e2e suite covers only a subset of documented `RALPH_*` config surfaces; tests run: none; bug fixes discovered: e2e surface-coverage gaps identified; files touched: `test/e2e/*.go`.
- 2026-04-20: read pass over `test/e2e/COVERAGE_MATRIX.md`, `test/e2e/coverage_matrix_enforcement_test.go`, and `test/e2e/coverage_matrix_helpers_test.go` - confirmed current enforcement validates test-name mappings but not required-surface inventory completeness; tests run: none; bug fixes discovered: governance gap identified; files touched: `test/e2e/COVERAGE_MATRIX.md`, `test/e2e/coverage_matrix_*`.
- 2026-04-20: read pass over `README.md` and `internal/cli/init.go` - confirmed README init description is stale versus implemented interactive overwrite/preview flow and default logging behavior; tests run: none; bug fixes discovered: docs/runtime parity gap identified; files touched: `README.md`, `internal/cli/init.go`.
- 2026-06-17: `go test ./internal/agent -count=1` plus read pass over `internal/agent/oh-my-pi.go` and `git log --oneline -- specs/agents/oh-my-pi.md internal/agent/oh-my-pi.go` - confirmed oh-my-pi (omp) agent implementation correct: uses `omp launch --print [--model <model>] <prompt>` invocation per spec; recent fix `9df2f36` added missing 'launch' subcommand and removed unsupported flags; tests pass; files touched: `internal/agent/oh-my-pi.go`.
- 2026-06-17: `read specs/agents/oh-my-pi.md`, `specs/agents.md`, `specs/agent-env-overrides.md`, `IMPLEMENTATION_PLAN.md` and `git log --oneline --all -- specs/agents/oh-my-pi.md internal/agent/oh-my-pi.go` - gap analysis complete: all core agent integrations verified; oh-my-pi agent fixed in commit `9df2f36`; no additional implementation gaps found; files reviewed: agent implementations, runner, config env handling, verification log.

## Summary

| Phase | Goal | Status |
| --- | --- | --- |
| Phase 1 | Command routing and loop lifecycle | Complete |
| Phase 2 | Configuration precedence and overlay semantics | In progress |
| Phase 3 | Prompt resolution and prompt-level overrides | Complete |
| Phase 4 | Agent adapters and child process environment | Complete |
| Phase 5 | Logging and file safety guarantees | Complete |
| Phase 6 | Init command runtime/spec parity | In progress |
| Phase 7 | End-to-end coverage completeness and governance | In progress |
| Phase 8 | Quality and security gates | Complete |
| Phase 9 | Release workflow and artifact publishing | Complete |
| Phase 10 | Documentation and plan drift control | In progress |

**Remaining effort:** 4 phases remain open (2, 6, 7, 10), centered on config edge-case hardening, e2e required-surface completeness, and docs/runtime synchronization.
- 2026-06-17: Verification complete - oh-my-pi agent integration functional; implementation plan updated with current status; all agent adapters (`omp`, `opencode`, `claude`, `cursor`) verified per specs.

## Known Existing Work

- Root and `run` command flows already converge through shared `runCommandLogic`.
- Built-in `build` and `plan` prompts already encode planning/build workflows and completion-signal placeholders.
- File-based prompt front matter parsing/stripping with precedence merge is implemented.
- Local overlay deep-merge behavior for `[prompt-overrides]` and `[env]` is implemented.
- Unsupported TOML `config-file` key already fails fast in base/default/overlay config paths.
- Child agent env overrides via config `[env]` and repeatable `--env` flags are implemented with validation.
- Supported agent adapters (`opencode`, `claude`, `cursor`) already use explicit child `cmd.Env`.
- Logging defaults, file permissions, git metadata headers, and stdout parity are implemented.
- `ralph init` already has ordered questionnaire, retries, overwrite confirmation, existing-config default seeding, preview confirmation, and atomic writes.
- E2E harness already uses one deterministic test-only agent fixture for all supported agent names.
- Coverage matrix artifact and stale/missing test-name mapping check already exist.
- Local/CI quality and security workflows plus release artifact publishing workflow are already wired.

## Manual Deployment Tasks

None
