# Prompts Command

Status: Implemented

## Overview

### Purpose

Document the `ralph prompts` CLI command and its subcommands for listing and viewing prompt content.

### Goals

- Specify the behavior of `prompts list` and `prompts show` subcommands.
- Define output formats and user experience.
- Describe discovery and display of built-in and custom prompts.

### Non-Goals

- Prompt resolution logic (see [prompts.md](../prompts.md)).
- Agent execution or loop behavior.

### Scope

- In scope: `prompts list`, `prompts show`, prompt discovery, output formatting.
- Out of scope: prompt resolution order, built-in prompt content generation.

## Architecture

### Module/package layout (tree format)

```
internal/
  cli/
    prompts.go
  prompt/
    prompts.go
```

## Data model

### Core Entities

- `PromptInfo`
  - `Name string` — prompt identifier (filename without `.md` for custom, literal for built-in).
  - `Desc string` — human-readable description (from built-in metadata or file content).
  - `Kind string` — either `"built-in"` or `"custom"`.

## Workflows

### List available prompts

1. User invokes `ralph prompts list`.
2. Prompt command scans for built-in prompts (always available: `build`, `plan`).
3. Command discovers custom prompt files by scanning `PromptsDir` and parent directories for `*.md` files.
4. Command outputs a list showing:
   - Prompt name
   - Description (from frontmatter or built-in metadata)
   - Source type (built-in or custom file path)
5. If no custom prompts exist, only built-in prompts are shown.

### Show prompt content

1. User invokes `ralph prompts show <name>` where `<name>` is a prompt identifier.
2. Command resolves the prompt:
   - If `<name>` matches a built-in prompt (`build` or `plan`), generates the full prompt text.
   - If `<name>` matches a custom prompt file, reads and displays file content.
3. Command outputs prompt content to stdout:
   - For built-in prompts: full generated text.
   - For custom prompts: file content with frontmatter stripped (if present).
4. If prompt not found, returns error: `prompt "<name>" not found`.

**Output behavior:**

- Prompt content written to stdout (supports shell redirection and piping).
- No frontmatter in output for custom prompts (YAML/TOML between `---` markers removed).
- No truncation — full content always shown.
- Exit code 0 on success, non-zero on error.

### Output Format

The `prompts list` command outputs prompts in the following format:

```
Built-in Prompts:
  build      Implement a single task from IMPLEMENTATION_PLAN.md after studying specs,
             then validate, commit, and update the plan.
  plan       Generate or update IMPLEMENTATION_PLAN.md with a phase-based plan after
             studying specs, existing code, and identifying gaps.

Custom Prompts:
  review     Code review workflow with security checklist
             ./prompts/review.md
  deploy     Deployment automation and verification steps
             ./prompts/deploy.md

Use 'ralph run <prompt-name>' to execute a prompt.
```

**Formatting rules:**

- Built-in prompts always listed first, under "Built-in Prompts:" heading.
- Custom prompts listed under "Custom Prompts:" heading (omit section if none found).
- Each prompt shows:
  - Name (left-aligned, padded to 10 characters minimum).
  - Description on same line or wrapped to next line with indentation.
  - File path on separate line below description (custom prompts only).
- Description truncated to 80 characters with `...` ellipsis if longer.
- Footer message suggests usage with `ralph run <prompt-name>`.

### Description Extraction for Custom Prompts

For custom prompt files, the description is extracted using the following precedence:

1. **Frontmatter `description` field** (if present in YAML/TOML frontmatter).
2. **First non-empty, non-heading line** of the file content.
3. **Fallback**: `(no description)` if neither is available.

**Note:** Frontmatter parsing is optional for initial implementation. If not implemented, use strategy #2 only.

## APIs

- `internal/prompt.DiscoverPrompts(cfg) []PromptInfo` — returns all available prompts (built-in + custom).
- `internal/prompt.BuiltInPrompts() []PromptInfo` — returns metadata for built-in prompts only.
- `internal/prompt.GetPromptContent(cfg, name string) (string, error)` — returns full prompt content for a given name, stripping frontmatter for custom prompts.
- `internal/prompt.PromptInfo` — struct with `Name`, `Desc`, `Kind` fields.

## Dependencies

- Standard library only (`os`, `io`, `path/filepath`, `strings`).
- `internal/prompt` for prompt discovery and content retrieval.

## Verifications

- `ralph prompts list` shows built-in prompts (`build`, `plan`) and any custom prompt files.
- `ralph prompts show build` outputs the full built-in build prompt content.
- `ralph prompts show plan` outputs the full built-in plan prompt content.
- `ralph prompts show review` outputs the content of `./prompts/review.md` with frontmatter stripped.
- `ralph prompts show nonexistent` returns error `prompt "nonexistent" not found`.

## Related Specifications

- [prompts.md](../prompts.md) — Prompt resolution and content generation.
- [run.md](run.md) — Using prompts with the `run` command.
## Implementation Notes

The prompts command was implemented with the following features:

### prompts list
- Lists built-in prompts (build and plan) with their descriptions
- Discovers custom prompts by scanning the configured PromptsDir
- Truncates descriptions to 80 characters for readability
- Provides usage hint to execute prompts with ralph run prompt-name

### prompts show name
- For built-in prompts: generates and displays the full prompt content
- For custom prompts: reads and displays file content with frontmatter stripped
- Returns error if prompt is not found

### Usage Examples

```bash
# List all available prompts
$ ralph prompts list
Built-in Prompts:
  build      Implement a single task from IMPLEMENTATION_PLAN.md after studying specs...
  plan       Generate or update IMPLEMENTATION_PLAN.md with a phase-based plan after study...

Use 'ralph run <prompt-name>' to execute a prompt.

# View the build prompt
$ ralph prompts show build
# Agent Instructions (Build Mode)

- Study specs/* (including specs/README.md and related specs).
- Study IMPLEMENTATION_PLAN.md and pick the single most important task.
...

# View the plan prompt
$ ralph prompts show plan
# Agent Instructions (Planning Mode)
...
```

## Verifications

The following verifications confirm the command works as specified:

- ralph prompts list - shows built-in prompts with descriptions (DONE)
- ralph prompts show build - displays full build prompt content (DONE)
- ralph prompts show plan - displays full plan prompt content (DONE)
- Command visible in ralph --help under Available Commands (DONE)
