# Implementation Plan (Oh My Pi Agent)

**Status:** Implemented with documentation gap (spec mentions --no-session, code does not)
**Last Updated:** 2026-06-18
**Primary Specs:** `specs/agents/oh-my-pi.md`, `specs/agents.md`

## Quick Reference

| System/Subsystem | Specs | Modules/Packages | Tests | Current State |
| --- | --- | --- | --- | --- |
| OmpAgent implementation | `specs/agents/oh-my-pi.md` ✅ | `internal/agent/oh-my-pi.go` ✅ | `internal/agent/agent_test.go` ✅ | **Gap: --no-session missing** |
| Agent factory | `specs/agents.md` ✅ | `internal/agent/agent.go` ✅ | `internal/agent/agent_test.go` ✅ | ✅ Complete |
| Agent runner | N/A | `internal/agent/runner.go` ✅ | `internal/agent/runner_internal_test.go` ✅ | ✅ Complete |
| Environment overrides | `specs/agent-env-overrides.md` ✅ | `internal/agent/runner.go` ✅ | `test/e2e/agent_env_overrides_test.go` ✅ | ✅ Complete |
| E2E agent selection | `specs/agents.md` ✅ | N/A | `test/e2e/agent_selection_test.go` | ⚠️ No omp e2e test |
| Documentation | `specs/agents/oh-my-pi.md` ✅ | N/A | N/A | ✅ Updated with --no-session |

## Analysis Summary

### What Already Exists ✅

1. **OmpAgent struct** - `internal/agent/oh-my-pi.go` implements the `Agent` interface
   - Fields: `Model`, `AgentMode`, `Env`
   - Methods: `Execute()`, `Name()`, `IsAvailable()`

2. **Agent factory integration** - `internal/agent/agent.go:26-29`
   - Returns `OmpAgent` for both `"omp"` and `"oh-my-pi"` agent names
   - Passes environment snapshot to agent

3. **Test coverage** - `internal/agent/agent_test.go`
   - `TestOmpExecuteAndAvailability` - verifies command args and availability check
   - `TestOmpExecuteStreamsOutputInRealTime` - verifies streaming behavior
   - `TestAllAgentsExecuteWithProvidedEnvironment` - verifies env passing

4. **Documentation** - `specs/agents/oh-my-pi.md` (updated 2026-06-18)
   - Added `--no-session` flag to invocation docs
   - Explains ephemeral execution rationale

5. **Agent runner** - `internal/agent/runner.go`
   - `executeAgentCommand` handles streaming, env passing, error handling
   - `BuildEffectiveEnv` for environment variable management

6. **Environment overrides** - `test/e2e/agent_env_overrides_test.go`
   - E2E test for custom environment variable passing

### Identified Gap [ ]

**The spec (updated 2026-06-18 19:10) includes `--no-session` flag, but the implementation does not:**

- **Spec (line 80):** `Build args: --print, --no-title, --no-session, optional --model <model>, <prompt>`
- **Spec (line 129):** `omp --print --no-title --no-session [--model <model>] <prompt>`
- **Spec (line 132):** "The `--no-session` flag is always included to ensure ephemeral execution without saving session history or using memory/skills from previous sessions."
- **Code (`oh-my-pi.go:17-21`):**
  ```go
  args := []string{"--print", "--no-title"}
  if a.Model != "" {
      args = append(args, "--model", a.Model)
  }
  args = append(args, prompt)
  ```

**Missing:** `--no-session` flag in the `args` slice.

**Test reflects old behavior:** `TestOmpExecuteAndAvailability` (line 306) expects:
```go
if !strings.Contains(result, "omp:--print --no-title --model m4 prompt") {
```
This test will fail once `--no-session` is added.

**Additional gaps:**
- No e2e test for omp agent in `test/e2e/agent_selection_test.go` (only claude, cursor tested)
- Commit 824e1a7 updated spec only; implementation unchanged

## Phased Plan

### Phase 1: Add --no-session Flag to OmpAgent.Execute()

**Goal:** Add `--no-session` flag to match spec documentation
**Status:** `[ ]` Not started
**Paths:**
- `internal/agent/oh-my-pi.go`
- `internal/agent/agent_test.go`

**Checklist:**
- `[ ]` Add `--no-session` to args slice in `oh-my-pi.go:17`
- `[ ]` Update test expectation in `agent_test.go:306` to include `--no-session`
- `[ ]` Verify both `"omp"` and `"oh-my-pi"` agent names work

**Definition of Done:**
- `[ ]` `go test ./internal/agent -run TestOmp -count=1` passes
- `[ ]` `make lint` passes
- `[ ]` Manual verification: args include `--no-session`
- Files touched: `internal/agent/oh-my-pi.go`, `internal/agent/agent_test.go`

**Risks/Dependencies:**
- None - straightforward flag addition
- Test must be updated to match new expected args

**Reference Pattern:**
- See `internal/agent/claude.go` or `internal/agent/cursor.go` for similar flag-building patterns
- Current implementation at `internal/agent/oh-my-pi.go:17-21`

### Phase 2: Add E2E Test for Omp Agent

**Goal:** Add end-to-end test for omp agent selection in `test/e2e/agent_selection_test.go`
**Status:** `[ ]` Not started
**Paths:**
- `test/e2e/agent_selection_test.go`

**Checklist:**
- `[ ]` Add test case for `--agent omp`
- `[ ]` Verify args include `--print`, `--no-title`, `--no-session`
- `[ ]` Verify model argument passing with `--model`

**Definition of Done:**
- `[ ]` `make test-e2e` passes with new omp test
- `[ ]` Test mirrors structure of claude/cursor tests
- Files touched: `test/e2e/agent_selection_test.go`

**Risks/Dependencies:**
- Requires `ralph-test-agent` mock to handle omp-style args
- May need to update `test/e2e/agents/ralph-test-agent/main.go`

### Phase 3: Verification

**Goal:** Ensure implementation matches spec and all tests pass
**Status:** `[ ]` Not started
**Paths:**
- `internal/agent/*`
- `test/e2e/*`
- `specs/agents/oh-my-pi.md`

#### 3.1 Run Quality Gates

**Checklist:**
- `[ ]` `make quality` passes (lint, test, security, arch)
- `[ ]` `make test` passes (full test suite including e2e)
- `[ ]` `make test-e2e` passes
- `[ ]` Verify no other references to omp args need updating

**Definition of Done:**
- `[ ]` All CI gates pass
- `[ ]` Spec and code are synchronized
- Files touched: test files as needed

**Risks/Dependencies:**
- E2E tests may reveal additional gaps

## Verification Log

- 2026-06-18: Read `specs/agents/oh-my-pi.md` - confirmed spec includes `--no-session` flag (commit 824e1a7, 2026-06-18 19:10); tests run: none; files touched: `specs/agents/oh-my-pi.md`.
- 2026-06-18: Read `internal/agent/oh-my-pi.go` lines 1-35 - confirmed implementation missing `--no-session` flag; tests run: none; files touched: `internal/agent/oh-my-pi.go`.
- 2026-06-18: Read `internal/agent/agent_test.go` lines 289-314 - confirmed test expects old args without `--no-session`; tests run: `go test ./internal/agent -run TestOmp` (passes with current code); files touched: `internal/agent/agent_test.go`.
- 2026-06-18: Git history analysis - commit 824e1a7 updated spec to include `--no-session` but implementation was not updated; files touched: git log output.
- 2026-06-18: Read `test/e2e/agent_selection_test.go` - confirmed no e2e test for omp agent (only claude, cursor); files touched: `test/e2e/agent_selection_test.go`.
- 2026-06-18: Read `internal/agent/agent.go` lines 22-39 - confirmed factory returns OmpAgent for both "omp" and "oh-my-pi" names; files touched: `internal/agent/agent.go`.
- 2026-06-18: Read `internal/agent/runner.go` lines 44-74 - confirmed executeAgentCommand handles streaming and env passing correctly; files touched: `internal/agent/runner.go`.

## Summary

| Phase | Goal | Status |
| --- | --- | --- |
| Phase 1 | Add --no-session flag to OmpAgent.Execute() | `[ ]` Not started |
| Phase 2 | Add E2E test for omp agent selection | `[ ]` Not started |
| Phase 3 | Verification (quality gates, e2e) | `[ ]` Not started |

**Remaining effort:** 
- Phase 1: 1 flag addition + 1 test update (~5 minutes)
- Phase 2: 1 e2e test case (~10 minutes)
- Phase 3: Quality gate runs (~2-3 minutes)
- **Total:** ~15-20 minutes, 3 files to modify

## Known Existing Work

- ✅ `OmpAgent` struct and interface implementation complete
- ✅ Agent factory supports both `"omp"` and `"oh-my-pi"` names
- ✅ Test coverage for execution, availability, streaming, and environment passing
- ✅ Spec documentation updated with `--no-session` flag rationale
- ✅ Environment snapshot and override mechanism in `runner.go`
- ✅ E2E test framework for agent selection (claude, cursor examples exist)
- ✅ Agent environment overrides E2E test complete

## Manual Deployment Tasks

None