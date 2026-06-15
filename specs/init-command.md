# Init Command

Status: Implemented

## Overview

### Purpose

- Define an interactive `ralph init` command that guides users through initial CLI configuration.
- Reduce setup friction by generating a valid `ralph.toml` from guided answers instead of manual edits.

### Goals

- Provide a deterministic interactive questionnaire for common Ralph configuration fields.
- Generate a TOML file compatible with the existing configuration loader and precedence rules.
- Prevent accidental data loss with explicit overwrite confirmation.

### Non-Goals

- Replacing configuration precedence rules (flags > env > config file > defaults).
- Validating external agent account/authentication status.
- Managing secrets or writing credentials to config files.

### Scope

- In scope: command UX, question flow, validation rules, TOML generation, and file write behavior.
- Out of scope: runtime execution loop behavior, prompt resolution changes, and agent CLI internals.

## Architecture

### Module/package layout (tree format)

```
internal/
  cli/
    cmd.go
    init.go
  config/
    config.go
    writer.go
specs/
  init-command.md
```

### Component diagram (ASCII)

```
+--------------------+
| User (TTY)         |
+---------+----------+
          |
          v
+---------+----------+
| init command       |
| internal/cli       |
+---------+----------+
          |
          v
+---------+----------+
| Question engine    |
+---------+----------+
          |
          v
+---------+----------+
| Config serializer  |
| (TOML)             |
+---------+----------+
          |
          v
+---------+----------+
| Config file on disk|
+--------------------+
```

### Data flow summary

1. User runs `ralph init` in an interactive terminal.
2. Command resolves output path (`./ralph.toml` by default).
3. Command seeds defaults from an existing target config file (if present) or from configuration defaults.
4. Command asks ordered interactive questions and validates each answer.
5. Answers are normalized into config keys compatible with `internal/config`.
6. Command renders deterministic TOML and writes it atomically.
7. Command prints a success summary and suggested next commands.

## Data model

### Core Entities

- InitSession
  - Fields: `OutputPath`, `IsTTY`, `ExistingConfigFound`, `Questions`, `Answers`, `Confirmed`.
  - Represents one interactive run of `ralph init`.

- InitQuestion
  - Fields: `Key`, `Prompt`, `Type`, `DefaultValue`, `Options`, `Required`, `Validator`.
  - `Type` values: `select`, `input`, `confirm`.

- InitAnswers
  - Fields: `AgentName`, `Model`, `AgentMode`, `MaxIterations`, `SpecsDir`, `SpecsIndexFile`, `ImplementationPlanName`, `PromptsDir`, `NoLog`, `LogFile`, `LogTruncate`.
  - Mirrors existing configuration fields in [specs/configuration.md](configuration.md).

### Relationships

- `InitSession` owns `InitQuestion` sequence and collects `InitAnswers`.
- `InitAnswers` is serialized to TOML keys defined in the configuration spec.
- `NoLog` is derived from an affirmative/negative logging question and is persisted as the existing `no-log` key.

### Persistence Notes

| Store  | Format | Location                  | Notes                                      |
| ------ | ------ | ------------------------- | ------------------------------------------ |
| Config | TOML   | `./ralph.toml` by default | Written atomically via temp file + rename. |

## Workflows

### `ralph init` (new file, happy path)

1. Verify stdin/stdout are attached to a TTY.
2. Resolve output path from `--output` or default `ralph.toml` in current directory.
3. Build question defaults from baseline config defaults.
4. Ask questions in order, validating input at each step.
5. Show final preview summary (not raw TOML) and ask for confirmation.
6. Write TOML to target path and print success message.

### `ralph init` (existing file)

1. Detect existing target config file.
2. Parse existing values and use them as question defaults when valid.
3. Prompt for overwrite confirmation unless `--force` is provided.
4. If user declines overwrite, exit without changing files.

### Input validation and retry

1. For enum questions (for example agent), accept only allowed options.
2. For integer questions (`max-iterations`), require an integer greater than 0.
3. For path/file-name questions, reject empty values where required.
4. On invalid input, display a validation message and re-prompt.

### Non-interactive invocation

1. If no TTY is detected, command fails fast.
2. Exit non-zero with guidance to run in an interactive terminal.

### Write failure

1. If directory creation or file write fails, command returns an error and exits non-zero.
2. Existing config file content remains unchanged when atomic rename is not reached.

## APIs

- None. This is a local interactive CLI command.

## Client SDK Design

- Not applicable.

## Configuration

- `ralph init` writes TOML keys already defined in [specs/configuration.md](configuration.md).
- Runtime precedence is unchanged: flags > env vars > config file > defaults.

### Command interface

- `ralph init`
- Optional flags:
  - `--output`, `-o`: target file path (default: `./ralph.toml`)
  - `--force`: overwrite existing target file without overwrite prompt

### Interactive question set

| Prompt                                    | Config key                 | Type    | Default                  | Validation                      |
| ----------------------------------------- | -------------------------- | ------- | ------------------------ | ------------------------------- |
| AI agent (`omp`, `opencode`, `claude`, `cursor`) | `agent`                    | select  | `opencode`               | Must be one of supported agents |
| Model (optional)                          | `model`                    | input   | empty                    | Free text; empty allowed        |
| Agent mode/sub-agent (optional)           | `agent-mode`               | input   | empty                    | Free text; empty allowed        |
| Maximum iterations                        | `max-iterations`           | input   | `25`                     | Integer > 0                     |
| Specs directory                           | `specs-dir`                | input   | `specs`                  | Non-empty path                  |
| Specs index file                          | `specs-index-file`         | input   | `README.md`              | Non-empty file name             |
| Implementation plan file                  | `implementation-plan-name` | input   | `IMPLEMENTATION_PLAN.md` | Non-empty file name             |
| Prompts directory                         | `prompts-dir`              | input   | `.ralph/prompts`         | Non-empty path                  |
| Enable logging?                           | `no-log` (inverted)        | confirm | `no`                     | Boolean                         |
| Log file path (when logging enabled)      | `log-file`                 | input   | `./ralph.log`            | Non-empty path                  |
| Truncate log file on each run?            | `log-truncate`             | confirm | `no`                     | Boolean                         |

### Generated TOML behavior

- Answers are converted into config keys defined in [specs/configuration.md](configuration.md).
- `no-log` is written from the logging confirmation answer (`Enable logging?` is the inverse of `no-log`).
- Writes use atomic temp-file + rename semantics through `internal/config/writer.go`.

## Permissions

- Requires interactive terminal access for question prompts.
- Requires read permission for an existing target config file (if present).
- Requires write permission in the target config directory.

## Security Considerations

- The questionnaire must avoid requesting secrets or tokens.
- Generated config should contain only operational settings already documented in the configuration spec.
- The command should not print sensitive prompt text or agent output; it only prints setup values and status.

## Dependencies

| Dependency                                        | Purpose                                         |
| ------------------------------------------------- | ----------------------------------------------- |
| `github.com/spf13/cobra`                          | Command and flag wiring                         |
| Standard library (`bufio`, `os`, `path/filepath`) | Interactive input, validation, and file writing |

## Open Questions / Risks

- Should `ralph init` preserve comments and unknown keys when overwriting an existing TOML file?
- Should a `--minimal` mode exist to ask only agent/model/mode and skip advanced options?
- Should non-interactive bootstrap mode be added later for CI or automation?

## Verifications

- Running `ralph init` in a TTY and accepting defaults creates `ralph.toml` with valid TOML.
- Running `ralph init` with invalid `max-iterations` re-prompts until a valid integer > 0 is provided.
- Running `ralph init` when `ralph.toml` exists prompts for overwrite and leaves file unchanged when declined.
- Running `ralph init` without a TTY exits non-zero with a clear guidance message.
- A generated config is loaded successfully by existing config resolution logic.

## Appendices

### Example generated config (accept defaults)

```toml
agent = "opencode"
max-iterations = 25
specs-dir = "specs"
specs-index-file = "README.md"
implementation-plan-name = "IMPLEMENTATION_PLAN.md"
prompts-dir = ".ralph/prompts"
no-log = true
log-truncate = false
```
