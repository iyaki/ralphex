# Ralphex

Ralphex is a specs-first CLI for running iterative coding loops against external agent CLIs such as OpenCode, Claude Code, and Cursor. It gives you built-in `plan` and `build` prompts, deterministic configuration precedence, and a simple command model that works well in repository-driven workflows.

It wraps a supported agent CLI in a repeatable loop: resolve a prompt, run the agent, and repeat until the agent emits `<promise>COMPLETE</promise>` or max iterations is reached. Each pass works against the updated repository, which makes it a good fit for specs, implementation plans, and repository-local prompts.

> [!NOTE]
> The repository is `iyaki/ralphex`, but the CLI command remains `ralph`.

> [!IMPORTANT]
> Ralphex does not bundle an agent runtime. Install and authenticate a supported agent CLI separately, then make sure its binary is available on your `PATH`.

## Why Ralphex

- Runs a repeatable prompt loop until the agent signals completion or max iterations is reached.
- Supports built-in `build` and `plan` prompts plus inline, stdin, and file-based prompts.
- Applies predictable precedence across flags, environment variables, config files, and prompt front matter.
- Keeps repository-specific prompts and local overrides easy to manage.
- Fits the Ralph Wiggum style of specs-first development.

## Supported Agents

| Agent | Binary |
| --- | --- |
| Oh My Pi | `omp` |
| OpenCode | `opencode` |
| Claude Code | `claude` |
| Cursor | `cursor` |

**Codex, Copilot, Gemini, and more agents, coming soon**

`model` and `agent-mode` are forwarded when the selected agent CLI supports them.

<details>
<summary><strong>Adding support for new agents</strong></summary>

### Adding Support for New Agents

Agent support Pull Requests are always welcomed. To add or update agent integrations, follow the workflow in [`CONTRIBUTING.md` ("Adding Support for a New Agent")](CONTRIBUTING.md#adding-support-for-a-new-agent):

1. `agent-spec-creation` for spec definition
2. `agent-implementation` for TDD-based code changes

</details>

## Install

Requirements:

- A supported agent CLI in `PATH`
- Go `1.25` if you are building from source

Prebuilt binaries are published on [GitHub Releases](https://github.com/iyaki/ralphex/releases). The latest release page is https://github.com/iyaki/ralphex/releases/latest.

Devcontainer feature:

`ghcr.io/iyaki/devcontainer-features/ralph:1`

Build from source:

```bash
make build
./bin/ralph --help
./bin/ralph version
```

Build with custom version:

```bash
make build VERSION=1.2.3
./bin/ralph version
# ralph v1.2.3
```

Run directly from source without building:

```bash
make run ARGS='--help'
```

Install system-wide after building:

```bash
make install
```

For a reproducible local environment, open the repository with the included dev container in `.devcontainer/devcontainer.json`.

## Quick Start

If you are already doing spec-driven development:

```bash
# Create a starter config
ralph init

# Generate or refresh the implementation plan for a scope
ralph plan authentication

# Run the default build loop
ralph

# Run the same default loop through the explicit entrypoint and prompt name
ralph run build

# Use a different agent for one run
ralph --agent claude --model claude-sonnet-4 build
```

Common prompt entrypoints:

```bash
# Inline prompt text
ralph --prompt "Review the repository and summarize the highest-risk area"

# Prompt from stdin
printf 'Prompt from stdin' | ralph -

# Prompt from a file with an explicit prompt name
ralph --prompt-file ./prompts/review.md review

# Per-run child agent environment override
ralph --env HTTP_PROXY=http://127.0.0.1:8080 build
```

## Command Model

| Command | Behavior |
| --- | --- |
| `ralph` | Equivalent to `ralph run build` |
| `ralph run [prompt] [scope]` | Explicit loop entrypoint |
| `ralph <prompt> [scope]` | Alias to `ralph run <prompt> [scope]` when `<prompt>` is not a subcommand |
| `ralph prompts list` | List all available built-in and custom prompts |
| `ralph prompts show <name>` | Display full content of a specific prompt |
| `ralph init` | Generate a starter config file |
| `ralph run init` | Run a prompt named `init` |
| `ralph version` | Print the version number |

Useful examples:

```bash
# Built-in planning prompt
ralph plan payment-service

# List available prompts
ralph prompts list

# View the full build prompt
ralph prompts show build

# View the full plan prompt
ralph prompts show plan

# Explicit run command
ralph run build

# Inline prompt text
ralph --prompt "Review the repository and summarize the highest-risk area"

# Prompt from stdin
printf 'Prompt from stdin' | ralph -

# Prompt from a file with an explicit prompt name
ralph --prompt-file ./prompts/review.md review

# Per-run child agent environment override
ralph --env HTTP_PROXY=http://127.0.0.1:8080 build
# Check Ralph version
ralph version
```

### `ralph init`

`ralph init` currently writes a starter config with repository-friendly defaults. It requires an interactive terminal, writes to `./ralph.toml` by default, and fails if the target already exists unless `--force` is set.

```bash
ralph init
ralph init --output ./team/ralph.toml
ralph init --force
```

The generated starter config disables logging by default (log file is empty). Users can enable logging during the interactive questionnaire by providing a log file path. The default prompts directory is set to `.ralph/prompts` so prompt files can live inside the repository.

### `ralph prompts`

The `prompts` command helps you discover and inspect built-in and custom prompts.

```bash
# List all available prompts
ralph prompts list

# View the full content of the build prompt
ralph prompts show build

# View the full content of the plan prompt
ralph prompts show plan

# View a custom prompt file
ralph prompts show review
```

The `prompts list` subcommand displays:
- Built-in Prompts: `build` and `plan` with brief descriptions
- Custom Prompts: Any prompt files found in the configured prompts directory with their file paths

This command is useful when you want to:
- Inspect built-in prompts to understand what instructions Ralphex sends to the agent
- Review custom prompts before using them
- Discover available prompts in your repository
- Debug prompt content without actually running the agent loop

The full prompt content is displayed exactly as it would be sent to the agent (with frontmatter stripped for custom prompts), making it easy to verify your prompt configuration or understand the default behavior.

## Creating Custom Prompts

When writing custom prompts (inline via `--prompt` or in prompt files), always include the completion signal so Ralphex knows when the agent is done.

**Inline prompt example:**
```bash
ralph --prompt "Review the authentication module and suggest improvements. When done, output: <promise>COMPLETE</promise>"
```

**Prompt file example (`prompts/review.md`):**
```markdown
Review the authentication module and identify security issues.
Write tests to cover edge cases.
When everything is done, output: <promise>COMPLETE</promise>
```

You can also use the placeholder `<COMPLETION_SIGNAL>` — Ralphex automatically replaces it with `<promise>COMPLETE</promise>` at runtime:
```markdown
Implement feature X.
Write tests for Y.
When everything is done, output: <COMPLETION_SIGNAL>
```

For full details on prompt resolution order and front matter, see [Prompt Resolution](#prompt-resolution).

 ## Prompt Resolution

Prompt content is resolved in this order:

For `model` and `agent-mode`, effective precedence is:

```text
flags > environment variables > prompt front matter > [prompt-overrides.<prompt>] > global config
```

## Configuration

Ralphex resolves settings with this precedence:

```text
flags > environment variables > config file > defaults
```

### Config Files and Local Overlays

Base config selection happens in this order:

1. `--config <path>`
2. `RALPH_CONFIG`
3. `./ralph.toml` in the current directory

If a base config file is selected, a sibling `ralph-local.toml` is merged over it before environment variables and flags are applied. This lets you commit shared defaults in `ralph.toml` and keep machine-specific overrides untracked in `ralph-local.toml`.

For example, `--config ./team/ralph.toml` loads `./team/ralph-local.toml` if it exists. It does not look for `./ralph-local.toml` in the current working directory.

`[env]` entries and `[prompt-overrides.<prompt>]` entries merge by key across `ralph.toml` and `ralph-local.toml`. Parse errors in either file stop the run before the agent starts.

> [!TIP]
> Keep shared team defaults in `ralph.toml`, add `ralph-local.toml` to `.gitignore`, and use the local file for personal overrides and secrets.

Example `ralph.toml`:

```toml
agent = "opencode"
model = "gpt-5"
max-iterations = 30

specs-dir = "specs"
specs-index-file = "README.md"
implementation-plan-name = "IMPLEMENTATION_PLAN.md"
prompts-dir = ".ralph/prompts"

no-log = false
log-file = "./ralph.log"

[prompt-overrides.plan]
agent-mode = "planner"

[env]
HTTP_PROXY = "http://127.0.0.1:8080"
```

### Environment Variables

| Variable | Purpose |
| --- | --- |
| `RALPH_CONFIG` | Select the base config file path |
| `RALPH_MAX_ITERATIONS` | Override `max-iterations` |
| `RALPH_SPECS_DIR` | Override `specs-dir` |
| `RALPH_SPECS_INDEX_FILE` | Override `specs-index-file` |
| `RALPH_IMPLEMENTATION_PLAN_NAME` | Override `implementation-plan-name` |
| `RALPH_CUSTOM_PROMPT` | Provide inline prompt text, similar to `--prompt` |
| `RALPH_LOG_FILE` | Override the log file path |
| `RALPH_LOG_ENABLED` | Boolean: `1` or `true` enables logging, `0` or `false` disables it |
| `RALPH_LOG_APPEND` | Boolean: `1` or `true` appends, `0` or `false` truncates on start |
| `RALPH_PROMPTS_DIR` | Override `prompts-dir` |
| `RALPH_AGENT` | Select the agent: `opencode`, `claude`, or `cursor` |
| `RALPH_MODEL` | Override `model` |
| `RALPH_AGENT_MODE` | Override `agent-mode` |

If `prompts-dir` is unset everywhere, the runtime default is `$HOME/.ralph`. The starter config written by `ralph init` pins `prompts-dir` to `.ralph/prompts` so prompt files live in the repository.

### Child Agent Environment

Ralphex's own `RALPH_*` settings are separate from the environment passed to the child agent. Use `[env]` in config for shared values and repeatable `--env KEY=VALUE` flags for per-run overrides.

Child agent environment precedence is:

```text
inherited environment < [env] in config < repeated --env flags
```

Later `--env` flags win for the same key. Each flag entry must use `KEY=VALUE`, and keys must match `[A-Za-z_][A-Za-z0-9_]*`.

### Logging

With no config file, logging is disabled by default. When logging is enabled and no explicit path is configured, the effective default log file is `./ralph.log` in the current working directory.

Logs mirror stdout and are written with `0600` permissions. Each run starts with a header that includes a timestamp plus git branch and commit metadata; unresolved git values are recorded as `N/A`.

```bash
# Enable logs for one run even if config disables them
ralph --no-log=false --log-file ./ralph.log build

# Start from an empty log file
ralph --no-log=false --log-file ./ralph.log --log-truncate build

# Same behavior through environment variables
RALPH_LOG_ENABLED=1 RALPH_LOG_APPEND=0 ralph build
```

## Completion Signal

Ralph detects task completion by searching for the following XML-like tag in agent output:

```xml
<promise>COMPLETE</promise>
```

The built-in `build` and `plan` prompts use the placeholder `<COMPLETION_SIGNAL>`, which Ralph automatically replaces with `<promise>COMPLETE</promise>` at runtime before sending the prompt to the agent.

When creating custom prompts (inline via `--prompt` or in prompt files), you can either:

1. Use the placeholder `<COMPLETION_SIGNAL>` — Ralph will replace it for you:
   ```markdown
   Implement feature X.
   Write tests for Y.
   When everything is done, output: <COMPLETION_SIGNAL>
   ```

2. Or directly include the actual signal:
   ```markdown
   Implement feature X.
   Write tests for Y.
   When everything is done, output: <promise>COMPLETE</promise>
   ```

The tag is case-sensitive and must appear exactly as shown. Ralph will stop after detecting the first occurrence.

> [!TIP]
> For technical details on the completion signal mechanism, see [specs/prompts.md#completion-signal](specs/prompts.md#completion-signal).
> [!CAUTION]
> Prompt text and agent output are written to stdout and, when logging is enabled, to the configured log file. Treat prompt files, config overrides, and logs as sensitive.

## Workflow

Ralphex is designed around a specs-first loop inspired by Geoffrey Huntley's [Ralph methodology](https://ghuntley.com/ralph/).

Typical flow:

1. Define or refine requirements in `specs/`.
2. Run `ralph plan <scope>` to generate or update `IMPLEMENTATION_PLAN.md`.
3. Run `ralph` or `ralph build` to implement the next task.
4. Repeat until the agent returns the completion signal.

The built-in prompts use a `<COMPLETION_SIGNAL>` placeholder, which Ralphex replaces with `<promise>COMPLETE</promise>` before sending the prompt to the agent.

## Spec Creator Skill

This repo includes the `spec-creator` [skill](https://agentskills.io/home) (see [.agents/skills/spec-creator/SKILL.md](.agents/skills/spec-creator/SKILL.md)) for the first phase of Geoffrey Huntley's Ralph Wiggum methodology.

To install it using Vercel's skills CLI, run:

```sh
npx skills add https://github.com/iyaki/ralphex/ --skill spec-creator
```

## Contributing

[See CONTRIBUTING.md](CONTRIBUTING.md)

## License

[MIT License](LICENSE)
