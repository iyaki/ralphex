# Oh My Pi (omp) Agent

Status: Implemented

## Overview

### Purpose

- Define how Ralph executes the `omp` (oh-my-pi) CLI agent.
- Specify command invocation and argument behavior.

### Goals

- Document invocation format and optional flags.
- Describe availability checks and error handling.

### Non-Goals

- Describing `omp` internal model or tool behavior.
- Implementing new agent features.

### Scope

- In scope: CLI invocation shape and runtime behavior in Ralph.
- Out of scope: prompt resolution and config precedence.

## Architecture

### Module/package layout (tree format)

```
internal/
  agent/
    oh-my-pi.go
specs/
  agents/
    oh-my-pi.md
```

### Component diagram (ASCII)

```
+------------------+
| OmpAgent         |
+---------+--------+
          |
          v
+---------+--------+
| omp CLI          |
+------------------+
```

### Data flow summary

1. Ralph selects `omp` when `AgentName` is `omp`.
2. The agent builds CLI args based on `Model` and `AgentMode`.
3. The agent executes `omp launch --print ... <prompt>` and returns output.

## Data model

### Core Entities

- OmpAgent
  - Fields: `Model`, `AgentMode`.
  - Implements `Agent` interface.

### Relationships

- Selected by `GetAgent` based on `AgentName`.
- Uses `Model` and `AgentMode` configuration fields.

### Persistence Notes

- None.

## Workflows

### Execute omp (happy path)

1. Build args: `launch`, `--print`, optional `--model <model>`, `<prompt>`.
2. Execute `omp` CLI.
3. Stream stdout/stderr and return combined output.

### Execute omp (error)

1. CLI exits non-zero.
2. Error is returned along with output.

## APIs

- None. This is a local CLI integration.

## Client SDK Design

- Not applicable.

## Configuration

- Relevant fields: `AgentName`, `Model`, `AgentMode`.
- See configuration spec for definitions and precedence.

## Permissions

- Requires OS permission to execute `omp`.

## Security Considerations

- Prompt text is passed as a CLI argument; sensitive data may appear in process lists.
- Executable resolved from PATH; ensure trusted `omp` binary.

## Dependencies

- Standard library only (`os/exec`, `bytes`, `io`).

## Open Questions / Risks

- Should the `--print` flag be configurable for interactive use?

## Verifications

- `ralph --agent omp build` invokes `omp launch --print`.
- `ralph --agent omp --model gpt-5.3-codex build` includes `--model gpt-5.3-codex`.

## Appendices

### Invocation

```
omp launch --print [--model <model>] <prompt>
```
