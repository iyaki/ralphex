# Google Antigravity CLI Agent

Status: Proposed

## Overview

### Purpose

- Define how Ralph executes the `agy` (Antigravity) CLI agent from Google.
- Specify command invocation, flags, and runtime behavior for non-interactive execution.
- Document configuration options for model selection, tool permissions, and workspace trust.

### Goals

- Document `agy` invocation shape with all relevant flags for scripting.
- Describe model selection, tool permission policies, and workspace trust handling.
- Specify error handling for CLI missing, authentication failures, and workspace trust prompts.
- Define settings.json integration for Antigravity-specific configuration.

### Non-Goals

- Describing Antigravity internal model behavior or tool selection.
- Implementing interactive TUI mode (Ralph uses non-interactive execution only).
- Covering Antigravity Cloud or remote workspace workflows.

### Scope

- In scope: `agy -p` CLI invocation, ephemeral sessions, settings.json configuration.
- Out of scope: prompt resolution (see prompts spec), config precedence (see configuration spec), Antigravity Cloud workflows.

## Architecture

### Module/package layout (tree format)

```
internal/
  agent/
    antigravity.go
specs/
  agents/
    antigravity.md
```

### Component diagram (ASCII)

```
+------------------+
| AntigravityAgent |
| internal/agent   |
+---------+--------+
          |
          | builds args
          v
+---------+--------+
| agy CLI          |
| --model          |
| -p (prompt)      |
+---------+--------+
          |
          | executes
          v
+---------+--------+
| Gemini API       |
| (via Antigravity)|
+------------------+
```

### Data flow summary

1. Ralph selects `antigravity` when `AgentName` is `antigravity`.
2. The agent builds CLI args: `-p <prompt>`, optional `--model`, and other flags.
3. The agent executes `agy -p <prompt>` and captures plain text output.
4. Output is streamed to stdout; final agent response returned.

## Data model

### Core Entities

- AntigravityAgent
  - Fields: `Model`, `SandboxMode`, `ToolPermission`.
  - Implements `Agent` interface: `Execute(prompt string, output io.Writer) (string, error)`, `Name() string`, `IsAvailable() bool`.

### Relationships

- Selected by `GetAgent` based on `AgentName == "antigravity"`.
- Uses `Model` configuration field for model selection (e.g., `Gemini 2.5 Pro`, `Gemini 2.5 Flash`).
- Uses `SandboxMode` configuration field for sandbox policy (`off`, `proceed-in-sandbox`).
- Uses `ToolPermission` configuration field (`request-review`, `always-proceed`, `strict`).
- Workspace trust is managed by Antigravity via `~/.gemini/antigravity-cli/settings.json`.

### Persistence Notes

- Antigravity stores settings in `~/.gemini/antigravity-cli/settings.json`.
- Workspace trust decisions are persisted per-directory in `trustedWorkspaces` array.
- Ralph does not manage settings.json — Antigravity handles this automatically.
- For first-run automation, users must have previously trusted the workspace via interactive `agy` session.

## Workflows

### Execute agy (happy path)

1. Verify `agy` binary availability via `LookPath`.
2. Build args:
   - Base: `-p <prompt>` (positional prompt required for non-interactive mode)
   - Model: `--model <Model>` if specified
   - Additional flags can be passed via environment or config
3. Execute `agy` CLI with combined args.
4. Stream stdout/stderr to output writer.
5. Return combined output and any error.

### Execute agy (CLI missing)

1. `IsAvailable()` returns false (binary not in PATH).
2. Warning is printed to stderr.
3. Execution is still attempted (may fail with OS-level error).
4. Error is returned along with any partial output.

### Execute agy (authentication failure)

1. Antigravity CLI exits with error (missing or expired credentials).
2. Error message indicates authentication issue.
3. Ralph returns error with message: "Antigravity authentication required. Run 'agy' and login first."
4. User must run `agy` interactively to complete Google OAuth login.

### Execute agy (workspace not trusted)

1. If workspace directory not in `trustedWorkspaces` list in settings.json.
2. Antigravity prompts for workspace trust (interactive TUI).
3. In non-interactive mode (`-p`), this may hang or fail.
4. **Mitigation**: User must run `agy` interactively once per new workspace to trust it.
5. Ralph should document this prerequisite clearly.
6. On timeout or failure, error returned: "Workspace not trusted. Run 'agy' interactively first."

### Execute agy (tool permission timeout)

1. If `ToolPermission` is `request-review` and tool execution requires approval.
2. Antigravity may wait for approval in interactive mode.
3. In non-interactive mode, behavior depends on setting:
   - `request-review`: May hang or auto-reject (Antigravity behavior).
   - `always-proceed`: Tools execute without prompting.
   - `strict`: Read-only operations only.
4. Ralph should configure `ToolPermission = always-proceed` for automation.
5. Timeout enforced on execution (configurable, default 300s).

### settings.json Configuration (Antigravity-managed)

1. **Location**: `~/.gemini/antigravity-cli/settings.json`
2. **Auto-managed** by Antigravity CLI via `/config` command or interactive setup.
3. **Key fields**:
   - `colorScheme`: UI preference (not relevant for Ralph).
   - `model`: Default model for Antigravity (overridden by `--model` flag).
   - `trustedWorkspaces`: Array of directory paths user has trusted.
   - `toolPermission`: Default permission mode.
   - `sandboxMode`: Whether to use sandboxed execution.
4. **Ralph does not manage settings.json** — Antigravity handles this automatically.
5. To customize: user runs `agy` interactively and uses `/config` command or edits settings.json directly.
6. Ralph reads config from its own config.toml; Antigravity settings are separate.

## APIs

- None. This is a local CLI integration with Google AI API access managed by Antigravity.

## Client SDK Design

- Not applicable.

## Configuration

### Ralph Configuration Fields

| Field            | Type   | Default           | Description                                      |
| ---------------- | ------ | ----------------- | ------------------------------------------------ |
| `AgentName`      | string | `"antigravity"`   | Must be `"antigravity"` for this agent.          |
| `Model`          | string | `"Gemini 2.5 Pro"`| Model override (e.g., `Gemini 2.5 Flash`).       |
| `SandboxMode`    | string | `"off"`           | Sandbox execution policy.                        |
| `ToolPermission` | string | `"always-proceed"`| Tool approval behavior for automation.           |

### Antigravity CLI Flag Mapping

| Ralph Config     | Antigravity Flag        | Notes                                               |
| ---------------- | ----------------------- | --------------------------------------------------- |
| `Model`          | `--model <MODEL>`       | Overrides settings.json model.                      |
| `SandboxMode`    | N/A (settings.json)     | Configure via `/config` → Sandbox Mode.             |
| `ToolPermission` | N/A (settings.json)     | Configure via `/config` → Tool Permission.          |

### Example Configuration

```toml
# Ralph config.toml
agent_name = "antigravity"
model = "Gemini 2.5 Flash"
sandbox_mode = "off"
tool_permission = "always-proceed"
```

### Antigravity Settings (managed separately)

```json
// ~/.gemini/antigravity-cli/settings.json
{
  "colorScheme": "dark",
  "model": "Gemini 2.5 Pro (High)",
  "trustedWorkspaces": [
    "/home/user/my-project",
    "/home/user/agy-cli-projects"
  ],
  "toolPermission": "always-proceed",
  "sandboxMode": "off"
}
```

**Note**: Ralph config and Antigravity settings.json are independent. Ralph passes `--model` flag; other settings must be pre-configured in settings.json via interactive `agy` session.

## Permissions

- Requires OS permission to execute `agy` binary.
- Requires file system access to working directory (for code reading/editing).
- Requires network access for Google AI API calls (handled by Antigravity).
- Workspace trust must be granted via interactive `agy` session before non-interactive use.
- Tool permissions affect what actions Antigravity can perform without approval:
  - `always-proceed`: Full autonomy for terminal commands and file operations.
  - `request-review`: Prompts for approval (not suitable for automation).
  - `strict`: Read-only operations only.

## Security Considerations

### Workspace Trust

- **First-run requirement**: Each new workspace directory requires one-time interactive trust grant.
- **Risk**: Running `agy -p` in untrusted workspace will fail or hang.
- **Mitigation**:
  - Run `agy` interactively in new workspace once and choose "Yes, I trust this folder".
  - Directory is added to `trustedWorkspaces` in settings.json.
  - Subsequent non-interactive runs succeed.

### Tool Execution Autonomy

- **`always-proceed`**: Allows Antigravity to execute terminal commands and file operations without approval.
  - Use only in trusted environments with no sensitive data.
  - Suitable for automation and CI.
- **`request-review`**: Prompts for each tool execution.
  - Safe for interactive use.
  - Not suitable for Ralph automation (will hang).
- **`strict`**: Read-only operations only.
  - Safest mode for untrusted prompts.
  - Limits Antigravity to code analysis and explanation.

### Code Execution

- Antigravity can execute arbitrary shell commands via terminal tool.
- Commands are subject to tool permission policies.
- **Risk**: Malicious prompts could attempt to exploit Antigravity tool access.
- **Mitigation**:
  - Use `strict` mode for untrusted prompts.
  - Use `request-review` for interactive sessions (not automation).
  - Audit outputs from automated runs.

### Authentication

- Antigravity requires Google account authentication via:
  - `agy` interactive login (Google OAuth).
  - System keyring for credential storage.
  - Fallback to browser-based Google Sign-In.
- **Risk**: OAuth tokens stored in system keyring could be accessed by other processes.
- **Mitigation**:
  - Use strong system-level security (file permissions, keyring encryption).
  - Log out via `/logout` when not in use (clears saved credentials).
  - Use separate Google accounts for different security contexts.

### Prompt Exposure

- Prompts are passed as `-p` argument; sensitive data may appear in process lists.
- **Mitigation**: For highly sensitive prompts, consider passing via stdin if Antigravity supports it.

### Binary Trust

- `agy` binary is resolved from PATH.
- **Risk**: Malicious binary substitution could intercept prompts or credentials.
- **Mitigation**:
  - Install Antigravity via official installer script.
  - Verify binary checksums when possible.
  - Use absolute path in Ralph config if PATH is untrusted.

## Dependencies

| Dependency       | Purpose                           | Required | Version Constraints |
| ---------------- | --------------------------------- | -------- | ------------------- |
| `agy` CLI        | Google Antigravity agent          | Yes      | Latest stable       |
| Go `os/exec`     | Execute external CLI              | Yes      | Standard library    |
| Go `bytes`, `io`| Output buffering and streaming    | Yes      | Standard library    |

### Installation

```bash
# Install Antigravity CLI (macOS / Linux)
curl -fsSL https://antigravity.google/cli/install.sh | bash

# Windows PowerShell
irm https://antigravity.google/cli/install.ps1 | iex

# Verify installation
agy --version
```

### Authentication Setup

```bash
# Run agy interactively to authenticate
agy

# Follow browser OAuth flow
# Trust workspace when prompted
```

### Configuration

```bash
# Configure settings via interactive CLI
agy

# Use /config command to adjust:
# - Model selection
# - Tool permissions
# - Sandbox mode
# - Workspace trust
```

## Installation & Setup Guide

### Step 1: Install Antigravity CLI

```bash
curl -fsSL https://antigravity.google/cli/install.sh | bash
```

### Step 2: Authenticate

```bash
agy
# Select Google OAuth login
# Complete authentication in browser
# Accept terms of service
```

### Step 3: Trust Workspace

```bash
# Navigate to your project directory
cd /path/to/project

# Run agy interactively
agy

# When prompted "Do you trust the contents of this project?"
# Select "Yes, I trust this folder"
```

### Step 4: Configure Tool Permissions (Optional)

```bash
agy
# Type /config
# Navigate to Tool Permission
# Select "always-proceed" for automation
# Navigate to Sandbox Mode
# Select "off" or "proceed-in-sandbox" based on needs
```

### Step 5: Use with Ralph

```bash
# Run Ralph with Antigravity
ralph --agent antigravity build

# With model override
ralph --agent antigravity --model "Gemini 2.5 Flash" build
```

## Testing & Verification

### Manual Verification Steps

1. **Binary availability**:
   ```bash
   agy --version
   # Should output version number
   ```

2. **Authentication check**:
   ```bash
   agy -p "Say hello"
   # Should complete without auth error
   ```

3. **Workspace trust check**:
   ```bash
   cd /path/to/project
   agy -p "List files"
   # Should execute without workspace trust prompt
   ```

### Ralph Integration Tests (Proposed)

```go
// internal/agent/antigravity_test.go
func TestAntigravity_Execute(t *testing.T) {
    // Verify agy binary is available
    agent := NewAntigravityAgent(Config{})
    if !agent.IsAvailable() {
        t.Skip("agy not installed")
    }
    
    // Test basic execution
    output, err := agent.Execute("Say hello", &bytes.Buffer{})
    assert.NoError(t, err)
    assert.Contains(t, output, "hello")
}

func TestAntigravity_ModelOverride(t *testing.T) {
    agent := NewAntigravityAgent(Config{
        Model: "Gemini 2.5 Flash",
    })
    // Verify --model flag is included
}

func TestAntigravity_UntrustedWorkspace(t *testing.T) {
    // Test behavior when workspace not trusted
    // Should return clear error message
}
```

## Open Questions / Risks

- **Workspace trust automation**: Should Ralph provide a helper to pre-trust workspaces? Currently requires manual interactive setup.
- **Tool permission defaults**: What is the safest default for `ToolPermission` in automation? Recommending `always-proceed` but this has security implications.
- **Stdin support**: Does `agy` support reading prompts from stdin to avoid process list exposure? Need to verify.
- **JSON output mode**: Does Antigravity support structured output (like `--json` flag) for better parsing? Currently assuming plain text.
- **Timeout behavior**: What happens if `agy -p` hangs waiting for approval? Need to document timeout requirements.
- **Multiple concurrent runs**: Can multiple `agy -p` invocations run concurrently in same workspace? Potential conflicts in shared settings.

## Verifications

- `ralph --agent antigravity build` invokes `agy -p "build"`.
- `ralph --agent antigravity --model "Gemini 2.5 Flash" build` includes `--model "Gemini 2.5 Flash"`.
- `ralph --agent antigravity` fails gracefully when `agy` binary is missing.
- `ralph --agent antigravity` returns clear error when workspace not trusted.
- `ralph --agent antigravity` returns clear error when authentication expired.
- Tool permission set to `always-proceed` allows autonomous execution.
- Output from `agy -p` is captured and returned by Ralph.

## Appendices

### Invocation

```
agy -p <prompt> [--model <model>]
```

**Flags**:
- `-p <prompt>`: Required. Prompt text for non-interactive execution.
- `--model <model>`: Optional. Model override (e.g., `Gemini 2.5 Pro`, `Gemini 2.5 Flash`).

**Examples**:
```bash
# Basic execution
agy -p "Explain this codebase"

# With model override
agy -p "Build this feature" --model "Gemini 2.5 Flash"

# In automation with timeout
timeout 300 agy -p "Run tests"
```

### Available Models (as of 2026-06)

To list available models:

```bash
agy models
```

Common models:
- `Gemini 2.5 Pro` (or `Gemini 2.5 Pro (High)` for high-compute tier)
- `Gemini 2.5 Flash` (or `Gemini 2.5 Flash (High)`)
- Additional models may be available based on account type and region

**Note**: Model names may vary. Use `agy models` to see exact identifiers.

### Tool Permission Modes

| Mode              | Behavior                                         | Use Case                    |
| ----------------- | ------------------------------------------------ | --------------------------- |
| `request-review`  | Prompts for each tool execution                  | Interactive, safe           |
| `always-proceed`  | Executes tools autonomously                      | Automation, trusted prompts |
| `strict`          | Read-only operations only                        | Analysis, untrusted prompts |
| `proceed-in-sandbox` | Tools run in isolated container               | Experimental, higher risk   |

### Related Documentation

- Official CLI Docs: https://antigravity.google/docs/cli/overview
- Product Page: https://antigravity.google/product/antigravity-cli
- GitHub Repo: https://github.com/google-antigravity/antigravity-cli
- Codelab: https://codelabs.developers.google.com/antigravity-cli-hands-on
- Extension Commands: https://docs.cloud.google.com/data-cloud-extension/antigravity/extension-commands