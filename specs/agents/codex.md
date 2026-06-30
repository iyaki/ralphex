# OpenAI Codex CLI Agent

Status: Proposed

## Overview

### Purpose

- Define how Ralph executes the `codex` CLI agent from OpenAI.
- Specify command invocation, flags, and runtime behavior for non-interactive execution.
- Document AGENTS.md integration for project context loading.

### Goals

- Document `codex exec` invocation shape with all relevant flags for scripting.
- Describe model selection, sandbox policies, and approval modes.
- Specify error handling for CLI missing, authentication failures, and approval timeouts.
- Define AGENTS.md loading behavior and configuration precedence.

### Non-Goals

- Describing Codex internal model behavior or tool selection.
- Implementing interactive TUI mode (Ralph uses non-interactive execution only).
- Covering Codex Cloud or remote app-server workflows.

### Scope

- In scope: `codex exec` CLI invocation, JSON output, ephemeral sessions, AGENTS.md integration.
- Out of scope: prompt resolution (see prompts spec), config precedence (see configuration spec), Codex Cloud workflows.

## Architecture

### Module/package layout (tree format)

```
internal/
  agent/
    codex.go
specs/
  agents/
    codex.md
```

### Component diagram (ASCII)

```
+------------------+
| CodexAgent       |
| internal/agent   |
+---------+--------+
          |
          | builds args
          v
+---------+--------+
| codex exec CLI   |
| --model          |
| --sandbox        |
| --ask-for-approval|
| --ephemeral      |
| --json           |
+---------+--------+
          |
          | executes
          v
+---------+--------+
| OpenAI API       |
| (via Codex)      |
+------------------+
```

### Data flow summary

1. Ralph selects `codex` when `AgentName` is `codex`.
2. The agent builds CLI args: `exec`, optional `--model`, `--sandbox`, `--ask-for-approval`, `--ephemeral`, `--json`, `--output-last-message`, and prompt.
3. Codex loads AGENTS.md files (global → project root → nested overrides).
4. The agent executes `codex exec ... <prompt>` and captures JSONL or plain text output.
5. Output is streamed to stdout; final message optionally written to file.

## Data model

### Core Entities

- CodexAgent
  - Fields: `Model`, `Sandbox`, `ApprovalMode`, `Ephemeral`, `OutputPath`.
  - Implements `Agent` interface: `Execute(prompt string, output io.Writer) (string, error)`, `Name() string`, `IsAvailable() bool`.

### Relationships

- Selected by `GetAgent` based on `AgentName == "codex"`.
- Uses `Model` configuration field for model selection (e.g., `gpt-5`, `gpt-5.4`, `o3`, `o4-mini`).
- Uses `Sandbox` configuration field for sandbox policy (`read-only`, `workspace-write`, `danger-full-access`).
- Uses `ApprovalMode` configuration field (`never`, `on-request`, `untrusted`).
- `Ephemeral` flag controls session persistence (default: `true` for Ralph).

### Persistence Notes

- When `--ephemeral` is set, Codex does not persist session rollout files to disk.
- Without `--ephemeral`, sessions are stored under `~/.codex/sessions/` and can be resumed.
- Ralph defaults to `--ephemeral` for stateless execution unless explicitly configured otherwise.

## Workflows

### Execute codex exec (happy path)

1. Verify `codex` binary availability via `LookPath`.
2. Build args:
   - Base: `exec`
   - Model: `--model <Model>` if specified
   - Sandbox: `--sandbox <Sandbox>` (default: `read-only`)
   - Approval: `--ask-for-approval <ApprovalMode>` (default: `never` for automation)
   - Ephemeral: `--ephemeral` (default: `true`)
   - Output format: `--json` for structured output (optional)
   - Output file: `--output-last-message <path>` or `-o <path>` (optional)
   - Profile: `--profile <name>` if configured
   - Inline config: `-c key=value` overrides
   - Prompt: final positional argument
3. Execute `codex` CLI with combined args.
4. Stream stdout/stderr to output writer.
5. If `--json` enabled, parse JSONL events for progress; extract final agent message.
6. Return combined output and any error.

### Execute codex exec (CLI missing)

1. `IsAvailable()` returns false (binary not in PATH).
2. Warning is printed to stderr.
3. Execution is still attempted (may fail with OS-level error).
4. Error is returned along with any partial output.

### Execute codex exec (authentication failure)

1. Codex CLI exits with error (missing or expired credentials).
2. Error message indicates authentication issue.
3. Ralph returns error with message: "Codex authentication required. Run 'codex login' first."
4. User must run `codex login` to authenticate via ChatGPT OAuth, device auth, or API key.

### Execute codex exec (approval timeout)

1. If `--ask-for-approval on-request` or `untrusted` and user interaction required in non-interactive mode.
2. Codex may hang waiting for approval input.
3. Ralph should enforce a timeout on execution (configurable, default 300s).
4. On timeout, process is terminated; error returned: "Codex approval timeout exceeded".

### Execute codex exec (sandbox violation)

1. Codex attempts command outside allowed sandbox scope.
2. With `read-only`: all write commands blocked.
3. With `workspace-write`: only working directory writes allowed.
4. Codex returns error; Ralph propagates with context.

### AGENTS.md loading (Codex automatic)

1. **Before execution**, Codex automatically discovers and loads AGENTS.md files:
   - Global: `~/.codex/AGENTS.md` or `~/.codex/AGENTS.override.md` (first non-empty).
   - Project root: Git root or current directory if no Git repo detected.
   - Nested: Each directory from root to cwd, checking `AGENTS.override.md` → `AGENTS.md` → fallback filenames.
2. Files concatenated in order (global → root → nested), with later files overriding earlier ones.
3. Combined size limited by `project_doc_max_bytes` (default 32 KiB).
4. Empty files skipped; truncation occurs if limit exceeded.
5. **Ralph does not manage AGENTS.md** — Codex handles discovery automatically.
6. To customize: user creates/edits AGENTS.md files directly; Codex reloads on each `codex exec` run.

## APIs

- None. This is a local CLI integration with OpenAI API access managed by Codex.

## Client SDK Design

- Not applicable.

## Configuration

### Ralph Configuration Fields

| Field           | Type   | Default            | Description                                      |
| --------------- | ------ | ------------------ | ------------------------------------------------ |
| `AgentName`     | string | `"codex"`          | Must be `"codex"` for this agent.                |
| `Model`         | string | `"gpt-5"`          | Model override (e.g., `gpt-5.4`, `o3`, `o4-mini`). |
| `Sandbox`       | string | `"read-only"`      | Sandbox policy for shell commands.               |
| `ApprovalMode`  | string | `"never"`          | Approval behavior for non-interactive execution. |
| `Ephemeral`     | bool   | `true`             | Skip session persistence to disk.                |
| `OutputPath`    | string | `""`               | Path to write final agent message (optional).    |
| `Profile`       | string | `""`               | Config profile name (optional).                  |

### Codex CLI Flag Mapping

| Ralph Config    | Codex Flag                        | Notes                                               |
| --------------- | --------------------------------- | --------------------------------------------------- |
| `Model`         | `--model <MODEL>` or `-m`         | Overrides config.toml model.                        |
| `Sandbox`       | `--sandbox <POLICY>` or `-s`      | `read-only` | `workspace-write` | `danger-full-access`. |
| `ApprovalMode`  | `--ask-for-approval <MODE>` or `-a` | `never` | `on-request` | `untrusted`. `on-failure` deprecated. |
| `Ephemeral`     | `--ephemeral`                     | Flag present = true; omitted = false.               |
| `Profile`       | `--profile <NAME>` or `-p`        | Layers `$CODEX_HOME/profile-name.config.toml`.      |
| Inline overrides| `-c key=value`                    | Repeatable; parses as TOML if possible.             |

### AGENTS.md Configuration (Codex-managed)

| Config Field                  | Default   | Description                                      |
| ----------------------------- | --------- | ------------------------------------------------ |
| `project_doc_max_bytes`       | 32768     | Max combined size of AGENTS.md files (32 KiB).   |
| `project_doc_fallback_filenames` | `[]`   | Additional filenames treated as AGENTS.md (e.g., `["TEAM_GUIDE.md", ".agents.md"]`). |

**Note**: These are configured in `~/.codex/config.toml`, not Ralph config. Ralph does not manage AGENTS.md content.

### Example Configuration

```toml
# Ralph config.toml
agent_name = "codex"
model = "gpt-5.4"
sandbox = "workspace-write"
approval_mode = "never"
ephemeral = true

# Optional: Codex-specific config in ~/.codex/config.toml
# [settings]
# model = "gpt-5.4"
# web_search = "cached"
# project_doc_max_bytes = 65536
```

## Permissions

- Requires OS permission to execute `codex` binary.
- Requires file system access to read AGENTS.md files (handled by Codex).
- Requires network access for OpenAI API calls (handled by Codex).
- Sandbox policies restrict shell command execution:
  - `read-only`: No filesystem writes, no network access.
  - `workspace-write`: Writes allowed only in working directory.
  - `danger-full-access`: Unrestricted access (use only in isolated environments).

## Security Considerations

### Sandbox Bypass Risks

- **`danger-full-access`**: Allows arbitrary shell commands, network access, and filesystem operations. Only use in:
  - Isolated CI runners or containers.
  - Environments with no sensitive data or credentials.
  - Scenarios where the cost of compromise is acceptable.
- **Workspace-write**: Safer default; restricts writes to project directory.
- **Read-only**: Most restrictive; suitable for code review, analysis, and planning tasks.

### Code Execution

- Codex can execute arbitrary shell commands via `bash`, `sh`, or other interpreters.
- Commands are subject to sandbox and approval policies.
- **Risk**: Malicious prompts could attempt to exploit Codex tool access.
- **Mitigation**:
  - Use `--sandbox read-only` for untrusted prompts.
  - Enable approval mode `on-request` for interactive review (not suitable for full automation).
  - Audit AGENTS.md files for injection risks (they are loaded verbatim into prompts).

### Authentication

- Codex requires authentication via:
  - `codex login` (ChatGPT OAuth, device auth).
  - `CODEX_API_KEY` environment variable (for API key auth).
  - `auth.json` file (stored credentials).
- **Risk**: API keys or auth tokens exposed in process lists, logs, or environment.
- **Mitigation**:
  - Never set `CODEX_API_KEY` as a job-level environment variable in CI when running untrusted code.
  - Use inline export: `CODEX_API_KEY=<key> codex exec ...` for single invocation.
  - Store `auth.json` with restrictive permissions (`chmod 600`).
  - Rotate credentials regularly.

### Prompt Injection via AGENTS.md

- AGENTS.md files are concatenated and injected into the system prompt.
- **Risk**: Malicious content in AGENTS.md could override instructions or exfiltrate data.
- **Mitigation**:
  - Version-control AGENTS.md files; review changes via pull requests.
  - Limit `project_doc_max_bytes` to prevent excessive token consumption.
  - Audit global and project-level AGENTS.md files periodically.

### Process List Exposure

- Prompts are passed as CLI arguments; sensitive data may appear in `ps` output.
- **Mitigation**: For highly sensitive prompts, consider passing via stdin (Codex supports piped stdin with prompt argument).

### Binary Trust

- `codex` binary is resolved from PATH.
- **Risk**: Malicious binary substitution could intercept prompts or credentials.
- **Mitigation**:
  - Install Codex via official channels (`npm install -g @openai/codex`).
  - Verify binary checksums when possible.
  - Use absolute path in Ralph config if PATH is untrusted.

## Dependencies

| Dependency       | Purpose                           | Required | Version Constraints |
| ---------------- | --------------------------------- | -------- | ------------------- |
| `codex` CLI      | OpenAI Codex command-line agent   | Yes      | Latest stable       |
| Go `os/exec`     | Execute external CLI              | Yes      | Standard library    |
| Go `bytes`, `io`| Output buffering and streaming    | Yes      | Standard library    |

### Installation

```bash
# Install Codex CLI via npm
npm install -g @openai/codex

# Or via Homebrew (macOS)
brew install openai-codex

# Authenticate after installation
codex login
```

### Verification

```bash
# Check Codex availability
codex --version

# Verify authentication
codex doctor
```

## Open Questions / Risks

- Should Ralph enforce a default timeout for `codex exec` to prevent hangs in non-interactive mode?
- Should `--json` output be the default for easier parsing, or should plain text remain default for readability?
- Should Ralph detect and warn when AGENTS.md files exceed `project_doc_max_bytes`?
- Should Ralph support `--ignore-rules` or `--ignore-user-config` flags for CI environments?

## Verifications

### Test Cases

1. **Basic execution**:
   ```bash
   ralph --agent codex "Explain this codebase"
   ```
   - Expected: `codex exec "Explain this codebase"` invoked; output returned.

2. **Model override**:
   ```bash
   ralph --agent codex --model gpt-5.4 "Refactor this function"
   ```
   - Expected: `codex exec --model gpt-5.4 "Refactor this function"`.

3. **Sandbox policy**:
   ```bash
   ralph --agent codex --sandbox workspace-write "Add a new file"
   ```
   - Expected: `codex exec --sandbox workspace-write "Add a new file"`.

4. **Ephemeral session**:
   ```bash
   ralph --agent codex --ephemeral "Generate documentation"
   ```
   - Expected: `codex exec --ephemeral "Generate documentation"`; no session files persisted.

5. **JSON output**:
   ```bash
   ralph --agent codex --json "List files" | jq '.type'
   ```
   - Expected: JSONL stream parsed; event types visible.

6. **Output file**:
   ```bash
   ralph --agent codex -o result.md "Write summary"
   ```
   - Expected: Final message written to `result.md` and stdout.

7. **CLI missing**:
   - Remove `codex` from PATH.
   - Run: `ralph --agent codex "test"`.
   - Expected: Warning printed; execution attempted with OS error.

8. **Authentication failure**:
   - Clear Codex credentials: `codex logout`.
   - Run: `ralph --agent codex "test"`.
   - Expected: Error: "Codex authentication required. Run 'codex login' first."

9. **AGENTS.md loading**:
   - Create `~/.codex/AGENTS.md` with content.
   - Create `AGENTS.md` in project root.
   - Run: `ralph --agent codex "Show active instructions"`.
   - Expected: Codex reports both global and project AGENTS.md content.

10. **Approval timeout** (interactive mode in non-interactive context):
    - Set `approval_mode = "on-request"`.
    - Run: `ralph --agent codex "Run git push"`.
    - Expected: Timeout after 300s (or configured duration); error returned.

## Appendices

### Invocation Examples

#### Basic non-interactive execution

```bash
codex exec "Explain the architecture of this repository"
```

#### With model selection and workspace write access

```bash
codex exec --model gpt-5.4 --sandbox workspace-write "Refactor the authentication module"
```

#### Ephemeral session with JSON output

```bash
codex exec --ephemeral --json "Generate unit tests for utils.go" | jq 'select(.type == "item.completed")'
```

#### Output final message to file

```bash
codex exec --output-last-message result.md --ephemeral "Write a changelog entry for recent changes"
```

#### Using inline config overrides

```bash
codex exec -c model_provider="openai" -c web_search="live" "Research latest Go 1.23 features"
```

#### With config profile

```bash
codex exec --profile ci-runner --ephemeral --sandbox danger-full-access "Run security audit"
```

#### Piping stdin with prompt

```bash
cat error.log | codex exec "Analyze this error and suggest fixes" > analysis.md
```

#### Resuming previous session

```bash
codex exec "Implement the plan you proposed"
codex exec resume --last "Continue with the implementation"
```

### Error Handling Reference

| Error Scenario           | Exit Code | Sample Error Message                         | Ralph Behavior                    |
| ------------------------ | --------- | -------------------------------------------- | --------------------------------- |
| CLI binary missing       | 127       | `codex: command not found`                   | Warn; attempt execution; propagate OS error |
| Authentication required  | 1         | `Error: Authentication required. Run 'codex login'` | Return error with login hint      |
| Approval timeout         | 1         | (hangs until killed)                         | Timeout after 300s; terminate process |
| Sandbox violation        | 1         | `Command blocked by sandbox policy`          | Propagate error with context      |
| Git repo not found       | 1         | `Error: Not a git repository`                | Propagate error; suggest `--skip-git-repo-check` |
| Invalid flag             | 1         | `Error: unknown flag: --invalid`             | Propagate error                   |
| API rate limit           | 1         | `Error: Rate limit exceeded`                 | Propagate error                   |

### References

- **Official Documentation**:
  - [Codex CLI Reference](https://developers.openai.com/codex/cli/reference)
  - [Non-interactive mode](https://developers.openai.com/codex/noninteractive)
  - [AGENTS.md Guide](https://developers.openai.com/codex/guides/agents-md)
  - [Codex Features](https://developers.openai.com/codex/cli/features)
  - [Authentication](https://developers.openai.com/codex/auth)
  - [Security Best Practices](https://developers.openai.com/codex/security)

- **GitHub Repository**:
  - [openai/codex](https://github.com/openai/codex)

- **Installation**:
  - NPM: `npm install -g @openai/codex`
  - Homebrew: `brew install openai-codex`

### Glossary

| Term               | Definition                                                                 |
| ------------------ | -------------------------------------------------------------------------- |
| `codex exec`       | Non-interactive subcommand for scripting and automation.                   |
| AGENTS.md          | Markdown file for persistent agent instructions (global or project-level). |
| Sandbox            | Isolated execution environment restricting filesystem and network access.  |
| Approval mode      | Controls when Codex pauses for human approval before executing commands.   |
| Ephemeral session  | Session that does not persist to disk after completion.                    |
| JSONL              | JSON Lines format; each line is a valid JSON object.                       |
| CODEX_HOME         | Environment variable or directory for Codex configuration and sessions.    |