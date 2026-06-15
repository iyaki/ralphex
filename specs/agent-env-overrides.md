# Agent Environment Overrides

Status: Implemented

## Overview

### Purpose

- Define how Ralph can set environment variables for the underlying agent process using config file values and CLI flags.
- Provide deterministic, testable precedence for combining inherited environment variables with Ralph-provided overrides.

### Goals

- Support multiple environment variable overrides in config files.
- Support repeatable CLI flags for per-run environment variable overrides.
- Ensure flag-provided values override config-provided values for the same key.
- Apply overrides consistently across all supported agents (`omp`, `opencode`, `claude`, `cursor`).

### Non-Goals

- Replacing existing configuration precedence rules for non-env fields.
- Adding external secret managers or vault integrations.
- Changing prompt resolution, loop control, or agent selection logic.

### Scope

- In scope: config schema additions, CLI flag parsing, validation, merge rules, and child-process environment construction.
- Out of scope: parent shell environment mutation and agent-specific secret lifecycle management.

## Architecture

### Module/package layout (tree format)

```
internal/
  cli/
    cmd.go
    run.go
  config/
    config.go
  agent/
    runner.go
specs/
  agent-env-overrides.md
```

### Component diagram (ASCII)

```
+-------------------+      +------------------+
| CLI flags         |      | Config file      |
| --env ...         |      | [env]            |
+---------+---------+      +---------+--------+
          |                          |
          +------------+-------------+
                       v
             +---------+---------+
             | Agent Env Merger  |
             | (with validation) |
             +---------+---------+
                       |
         baseline env  |  overrides
        (os.Environ()) v
             +---------+---------+
             | exec.Command Env  |
             +---------+---------+
                       |
                       v
             +---------+---------+
             | Agent CLI process |
             +-------------------+
```

### Data flow summary

1. Ralph loads config and parses CLI flags.
2. Ralph parses and validates `env` entries from config and flags.
3. Ralph starts with inherited process environment (`os.Environ()`).
4. Ralph applies config `env` overrides.
5. Ralph applies CLI `--env` overrides.
6. Ralph executes the selected agent with explicit `cmd.Env`.

## Data model

### Core Entities

- AgentEnvMap
  - Type: map-like key/value collection.
  - Key: environment variable name.
  - Value: environment variable value (empty string allowed).

- AgentEnvFlagEntry
  - Raw format: `KEY=VALUE`.
  - Parsed into a single key/value pair.

- EffectiveAgentEnv
  - Final environment passed to the child agent process.
  - Built from inherited environment plus deterministic overrides.

### Relationships

- `Config.Env` provides baseline overrides from the selected config file.
- `--env` provides highest-priority per-run overrides.
- `EffectiveAgentEnv` is consumed only by agent subprocess execution.

### Persistence Notes

- Config-based entries are persisted in TOML under `[env]`.
- Flag-based entries are ephemeral for the current command invocation.

## Workflows

### Config-only override (happy path)

1. User defines `[env]` in config file.
2. Ralph resolves config as usual.
3. Ralph applies `[env]` entries on top of inherited environment.
4. Agent process receives merged environment.

### Flag-only override (happy path)

1. User provides one or more `--env KEY=VALUE` flags.
2. Ralph validates and parses each flag entry.
3. Ralph applies parsed entries on top of inherited environment.
4. Agent process receives merged environment.

### Combined config and flags

1. Ralph applies config `[env]` entries.
2. Ralph applies `--env` entries afterward.
3. For duplicate keys, flag value wins.

### Duplicate flag keys

1. User provides the same key multiple times in `--env`.
2. Ralph processes entries in command-line order.
3. Last value for the key wins.

### Invalid flag entry

1. User provides `--env` without `=` or with an invalid key format.
2. Ralph fails fast before starting agent execution.
3. Error identifies the invalid key/entry, never prints sensitive values.

## APIs

- None. Behavior is internal to CLI/config/agent process execution.

## Client SDK Design

- Not applicable.

## Configuration

### CLI flags

| Flag    | Type       | Repeatable | Format      | Meaning                                |
| ------- | ---------- | ---------- | ----------- | -------------------------------------- |
| `--env` | `string[]` | yes        | `KEY=VALUE` | Set/override env var for agent process |

Notes:

- Split on the first `=` only; remaining `=` characters are part of the value.
- Empty values are allowed (`KEY=`).

### Config file keys (TOML)

Config files may define an `env` table:

```toml
[env]
ANTHROPIC_API_KEY = "<redacted>"
OPENAI_API_KEY = "<redacted>"
HTTP_PROXY = "http://127.0.0.1:8080"
```

### Validation

- Key format must match `^[A-Za-z_][A-Za-z0-9_]*$`.
- Key must be non-empty.
- Value may be any string, including empty.

### Precedence for agent process environment

1. Inherited process environment (`os.Environ()`).
2. Config file table `[env]`.
3. CLI flags `--env` (highest priority).

### Interaction with global config precedence

- Existing precedence for core Ralph fields remains unchanged.
- This spec defines precedence only for agent subprocess environment construction.
- `env` in this spec is a child-process override map, not Ralph's own `RALPH_*` environment variable source.

## Permissions

- Requires permission to execute external agent binaries.
- Requires read access to config file when using `[env]`.

## Security Considerations

- `--env` may expose values in shell history; prefer config/local overlays or inherited environment for sensitive keys.
- Ralph must avoid printing raw `env` values in logs and error messages.
- Config files containing secrets should use restrictive filesystem permissions.

## Dependencies

- Standard library only (`os`, `os/exec`, string parsing helpers).

## Open Questions / Risks

- Should a future `--env-file` option be added for non-shell secret loading?
- Should there be an explicit unset semantic to remove inherited keys from the child process?

## Verifications

- `ralph --env FOO=bar build` passes `FOO=bar` to the child agent process.
- `ralph --config ./ralph.toml build` applies values from `[env]`.
- `ralph --config ./ralph.toml --env FOO=flag build` uses `FOO=flag` over config value.
- Repeated flags (`--env FOO=one --env FOO=two`) use `FOO=two`.
- Invalid entries fail before agent execution and do not leak secret values.

## Appendices

### Example invocation

```bash
ralph --agent claude \
  --env ANTHROPIC_API_KEY=<redacted> \
  --env HTTP_PROXY=http://127.0.0.1:8080 \
  build
```

### Example config

```toml
agent = "opencode"
model = "gpt-5.3-codex"

[env]
OPENAI_API_KEY = "<redacted>"
```
