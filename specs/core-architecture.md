# Core Architecture

## Overview

### Purpose

- Define the current Ralph CLI architecture to standardize behavior across prompts and configuration.
- Provide a single, testable source of truth for future extensions (new agents, prompts, or config options).

### Goals

- Specify the CLI execution loop and completion signal detection.
- Document module boundaries so new features are added in the right layer.

### Non-Goals

- UI or web application design.
- Remote APIs or services.
- New AI agent implementations beyond the current CLI integrations.

### Scope

- In scope: CLI flow, iteration loop.
- Out of scope: external agent CLI behaviors, model performance, and upstream prompt authoring guidance.

## Architecture

### Module/package layout (tree format)

```
cmd/
  ralph/
    main.go
internal/
  agent/
    agent.go
    claude.go
    cursor.go
    opencode.go
  cli/
    cmd.go
  config/
    config.go
  executor/
    executor.go
  logger/
    logger.go
  prompt/
    prompts.go
specs/
  README.md
```

### Component diagram (ASCII)

```
+------------------+
|   CLI (cobra)    |
| internal/cli     |
+---------+--------+
          |
          v
+---------+--------+        +------------------+
|   Config Loader  |<------>| Config File/Env  |
| internal/config  |        +------------------+
+---------+--------+
          |
          v
+---------+--------+        +------------------+
| Prompt Resolver  |<------>| Prompt Files     |
| internal/prompt  |        +------------------+
+---------+--------+
          |
          v
+---------+--------+        +------------------+
|   Agent Runner   |<------>| Agent CLIs       |
| internal/agent   |        | opencode/claude  |
+---------+--------+        | cursor           |
          |                 +------------------+
          v
+---------+--------+
|  Logger (opt)    |
| internal/logger  |
+------------------+
```

### Data flow summary

1. CLI parses args and flags into a `Config` instance.
2. Config loader merges flags, env vars, config files, and defaults (see configuration spec).
3. Prompt resolver selects inline prompt, stdin, prompt file, or built-in prompt (see prompts spec).
4. The loop injects the completion signal and runs agent iterations (see agents spec).
5. Agent output is streamed to stdout and optionally to the log file (see logging spec).
6. Completion is detected by searching for `<promise>COMPLETE</promise>` in output. See [Prompts spec](prompts.md#completion-signal) for details on the completion signal mechanism.

## Data model

### Core Entities

- Config
  - See the configuration spec for full details on fields, defaults, and precedence.

- ExecutionResult
  - Fields: `Output` (string), `Error` (error or nil), `Completed` (bool).
  - Derived from agent execution and completion signal detection.

### Relationships

- `CLI` owns a single `Config` per run.
- Prompt resolution is defined in the prompts spec.
- Agent selection and execution are defined in the agents spec.
- Logging behavior is defined in the logging spec.

### Persistence Notes

- See logging spec.

## Workflows

### Run CLI (happy path)

1. User runs `ralph` with optional args/flags.
2. CLI loads config (see configuration spec).
3. Logger initializes (unless disabled, see logging spec).
4. Prompt is resolved from inline, stdin, file, or built-in defaults.
5. Loop injects completion signal and begins iterations.
6. Agent executes prompt and streams output.
7. If output contains completion signal, loop ends with success.

### Run CLI (prompt file missing)

1. CLI resolves prompt file path and fails to read it.
2. CLI returns an error, exits non-zero, and prints failure.

### Run CLI (agent not available)

1. CLI detects agent binary missing via `IsAvailable()`.
2. Warning is printed; loop still attempts execution.
3. Agent execution may fail; loop continues until completion or max iterations.

### Run CLI (max iterations reached)

1. Loop completes `MaxIterations` without completion signal.
2. CLI prints a warning and returns an error.

## APIs

- None. Ralph is a local CLI and does not expose HTTP APIs.

## Client SDK Design

- Not applicable. This is a CLI-only tool.

## Permissions

- No internal role model.
- Requires file system access to read prompt/config files.
- Requires OS permission to execute external CLIs (see agents spec).

## Security Considerations

- Agent-related security considerations are covered in the agents spec.
- Logging-related risks are covered in the logging spec.
- Executing external CLIs relies on PATH; ensure trusted binaries are used.

## Dependencies

| Dependency                   | Purpose                        |
| ---------------------------- | ------------------------------ |
| `github.com/spf13/cobra`     | CLI parsing and command wiring |
| `github.com/BurntSushi/toml` | TOML config decoding           |
| Go `os/exec`                 | Execute agent CLIs             |

## Open Questions / Risks

- Should prompts be passed via stdin to reduce exposure in process lists?
- Do we need stricter validation on `AgentName` or `AgentMode` values?
- Should log file rotation or size limits be enforced?

## Verifications

- Running `./bin/ralph --help` returns without error.
- Running `./bin/ralph plan` renders a plan prompt (no config file).
- Running with missing prompt file returns a non-zero exit.
- Running with `DEBUG=1` exits after first iteration and prints prompt.

## Appendices

- Future: add a prompt source for remote templates or registries.
