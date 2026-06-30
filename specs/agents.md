# Agents

Status: Implemented

## Overview

### Purpose

- Define how Ralph selects and executes external AI agent CLIs.
- Provide a testable description of the agent interface and supported agents.

### Goals

- Specify the agent selection behavior.
- Describe availability checks and error handling.

### Non-Goals

- Implementing new agent types.
- Standardizing outputs from external agent CLIs.

### Scope

- In scope: agent interface, selection logic, and CLI invocation rules.
- Out of scope: prompt resolution (see prompts spec) and config precedence (see configuration spec).

## Architecture

### Module/package layout (tree format)

```
internal/
  agent/
    agent.go
    opencode.go
    claude.go
    oh-my-pi.go
    codex.go
    copilot.go
    antigravity.go
specs/
  agents/
    opencode.md
    claude.md
    cursor.md
    codex.md
    copilot.md
    antigravity.md
```
internal/
  agent/
    agent.go
    opencode.go
    claude.go
    oh-my-pi.go
    codex.go
    copilot.go
specs/
  agents/
    opencode.md
    claude.md
    cursor.md
```

### Component diagram (ASCII)

```
+------------------+
| Agent Factory    |
| internal/agent   |
+---------+--------+
          |
          v
+---------+--------+
| Agent Implementation|
+---------+--------+
          |
          v
+---------+--------+
| External CLI     |
+------------------+
```

### Data flow summary

1. CLI resolves `AgentName`, `Model`, and `AgentMode` (see configuration spec).
2. Agent factory returns a concrete agent implementation.
3. Availability check runs via `LookPath`.
4. Agent executes the external CLI command and returns output.

 - Oh My Pi (omp): [specs/agents/oh-my-pi.md](agents/oh-my-pi.md)
 - Opencode: [specs/agents/opencode.md](agents/opencode.md)
 - Claude: [specs/agents/claude.md](agents/claude.md)
 - Cursor: [specs/agents/cursor.md](agents/cursor.md)
 - OpenAI Codex: [specs/agents/codex.md](agents/codex.md)
 - GitHub Copilot: [specs/agents/copilot.md](agents/copilot.md)
 - Google Antigravity: [specs/agents/antigravity.md](agents/antigravity.md)

## Data model

### Core Entities

- Agent (interface)
  - `Execute(prompt string, output io.Writer) (string, error)`
  - `Name() string`
  - `IsAvailable() bool`

- AgentSelection
  - Fields: `AgentName`, `Model`, `AgentMode`.
  - Derived from config/flags/environment variables.

### Relationships

- `AgentSelection` determines which implementation is returned by `GetAgent`.
- `Agent` execution streams output to stdout and optional log file.

### Persistence Notes

- None. Agent selection is runtime-only.

## Workflows

### Select agent (happy path)

1. Read `AgentName` from config (see configuration spec).
2. Map `AgentName` to a concrete agent implementation.

### Select agent (unknown configured agent)

1. Read `AgentName` from config/flags/environment (see configuration spec).
2. Attempt to map `AgentName` to a concrete agent implementation.
3. If `AgentName` is unknown, return an error and stop before agent execution.

### Execute agent (happy path)

1. Build CLI arguments from `Model` and `AgentMode`.
2. Execute the agent CLI with the prompt as a final argument.
3. Capture stdout and stderr; return combined output. Output is monitored for the completion signal `<promise>COMPLETE</promise>` to determine when the loop should stop.

### Execute agent (CLI missing)

1. `IsAvailable()` returns false.
2. CLI prints a warning and still attempts execution.

### Execute agent (error)

1. External CLI exits non-zero.
2. Error is returned, but output is still provided.

## APIs

- None. Agents are local CLI integrations.

## Client SDK Design

- Not applicable.

## Configuration

- Relevant fields: `AgentName`, `Model`, `AgentMode`.
- See configuration spec for full definitions and precedence.

## Permissions

- Requires OS permission to execute external agent CLIs.

## Security Considerations

- Prompts are passed as CLI arguments; sensitive data may appear in process lists.
- Agent binaries are resolved from PATH; ensure trusted binaries are used.

## Dependencies

- Standard library only (`os/exec`, `bytes`, `io`).

## Open Questions / Risks

- Should agent selection fail fast when the CLI binary is missing?
- Should agent output be sanitized before logging?

## Verifications

 - `ralph --agent opencode build` selects `opencode`.
 - `ralph --agent claude --model claude-sonnet-4 build` passes model arg.
 - `ralph --agent cursor build` executes `cursor` CLI.
 - `ralph --agent copilot build` invokes `copilot -p`.
 - `ralph --agent copilot --model gpt-4o build` includes `--model gpt-4o`.
 - `ralph --agent copilot --agent explore build` includes `--agent explore`.
 - `ralph --agent antigravity build` invokes `agy -p "build"`.
 - `ralph --agent antigravity --model "Gemini 2.5 Flash" build` includes `--model "Gemini 2.5 Flash"`.

## Appendices

- None.
