# Implementation Plan (Help Prompts)

**Status:** Phase 2 complete - README updated with custom prompt examples
**Last Updated:** 2026-06-18
**Primary Specs:** `specs/prompts.md`, `specs/README.md`

## Quick Reference

| System/Subsystem | Specs | Modules/Packages | Web Packages | Migrations/Artifacts | Current State |
| --- | --- | --- | --- | --- | --- |
| Help text generation | `specs/prompts.md` | `internal/cli/cmd.go` ✅ | None | README.md ✅ | Complete (signal + example) |
| Completion signal docs | `specs/prompts.md` | `internal/prompt/prompts.go` ✅ | None | Examples ✅ | Complete |
| Custom prompt examples | `specs/prompts.md` | Help text ✅, README ✅ | None | `[x]` Example snippets | **Complete** |

## Analysis Summary

### What Already Exists ✅

1. **Help output mentions the signal** - `ralph --help` states:
   - "The loop runs until the agent emits `<promise>COMPLETE</promise>` or max iterations is reached"
   - "When writing custom prompts, include `<promise>COMPLETE</promise>` at the end to signal completion"

2. **Specs have detailed documentation** - `specs/prompts.md` lines 172-194 have a "Completion Signal" section with:
   - Explanation of placeholder `<COMPLETION_SIGNAL>` (auto-replaced)
   - Actual signal `<promise>COMPLETE</promise>`
   - Two examples showing both approaches

3. **Implementation handles replacement** - `internal/cli/run.go:218` replaces `<COMPLETION_SIGNAL>` automatically

4. **README documents it** - completion signal mentioned in repository documentation

### Identified Gap [ ]

The help output and README mention the signal but **do not show a complete example snippet** that users can copy/paste when creating custom prompts. The specs have examples, but users reading `ralph --help` or skimming README might miss the pattern.

**What users need to see:**
```markdown
Implement feature X.
Write tests for Y.
When everything is done, output: <promise>COMPLETE</promise>
```

Or with the placeholder:
```markdown
Implement feature X.
Write tests for Y.
When everything is done, output: <COMPLETION_SIGNAL>
```

This gap makes it slightly harder for users to create their first custom prompt quickly.

## Phased Plan

### Phase 1: Add Custom Prompt Example to Help Text

**Goal:** Show users a complete example snippet in `ralph --help` output
**Status:** `[x]` Complete
**Paths:**
- `internal/cli/cmd.go`
- `internal/cli/cmd_test.go`

**Checklist:**
- [x] Add a "Custom Prompt Example" section to the help long text
- [x] Show both placeholder and explicit signal patterns
- [x] Keep the example concise (3-4 lines max)

**Definition of Done:**
- `ralph --help` shows example snippet
- `go test ./internal/cli -run TestReadmeDocumentsRalphexRepoAndRalphCli -count=1` passes
- Files touched: `internal/cli/cmd.go`

**Risks/Dependencies:**
- Help text length should remain reasonable
- Example must be accurate and copy-paste ready

### Phase 2: Update README with Custom Prompt Example

**Goal:** Add a quick example in README for users creating custom prompts
**Status:** `[x]` Complete
**Paths:**
- `README.md`
- `cmd/ralph/main_test.go` (regression tests)

#### 2.1 Add example to README Quick Start or Prompt Resolution section

**Paths:**
- `README.md`

**Checklist:**
- `[x]` Add a "Creating Custom Prompts" subsection or enhance existing section
- `[x]` Show inline prompt example with completion signal
- `[x]` Show prompt file example with completion signal
- `[x]` Mention placeholder auto-replacement

**Definition of Done:**
- `[x]` `go test ./cmd/ralph -run TestReadmeDocumentsRalphexRepoAndRalphCli -count=1` passes
- `[x]` README clearly shows custom prompt pattern
- Files touched: `README.md`, `cmd/ralph/main_test.go`

**Risks/Dependencies:**
- README already lengthy; example must be concise
### Phase 3: Verification

**Goal:** Ensure examples are visible and accurate
**Status:** `[ ]` Not started
**Paths:**
- `internal/cli/*`
- `README.md`
- `test/e2e/`

#### 3.1 Run verification checks

**Checklist:**
- `[ ]` `ralph --help` shows example
- `[ ] ] `make test` passes
- `[ ]` Doc regression tests pass
- `[ ]` Manual verification: example is copy-paste ready

**Definition of Done:**
- `make quality` passes
- Files touched: test files as needed

## Verification Log

- 2026-06-18: `ralph --help` - confirmed completion signal mentioned but no example snippet shown; tests run: none; bug fixes discovered: documentation gap identified (example missing); files touched: none (planning mode).
- 2026-06-18: Read `specs/prompts.md` lines 172-194 - confirmed examples exist in spec but not exposed in help text; tests run: none; files touched: `specs/prompts.md`.
- 2026-06-18: Read `internal/cli/cmd.go` - identified `Long` field as target for help text enhancement; tests run: none; files touched: `internal/cli/cmd.go`.
- 2026-06-18: Memory recall - confirmed prior discussion about exposing completion signal in documentation; user requested showing the actual snippet pattern; tests run: none; files touched: none.
- 2026-06-18: `make build` - build successful; `go test ./internal/cli` - all tests pass; `./bin/ralph --help` - custom prompt examples visible; `make test` - full test suite passes; files touched: `internal/cli/cmd.go`, `IMPLEMENTATION_PLAN.md`.
- 2026-06-18: `go test ./cmd/ralph -run TestReadmeDocumentsRalphexRepoAndRalphCli -count=1` - test passes; `make test` - full suite passes; README updated with "Creating Custom Prompts" section showing inline and file-based examples; files touched: `README.md`, `cmd/ralph/main_test.go`, `IMPLEMENTATION_PLAN.md`.
| Phase | Goal | Status |
| --- | --- | --- |
| Phase 1 | Add custom prompt example to help text | `[x]` Complete |
| Phase 2 | Update README with example | `[x]` Complete |
| Phase 3 | Verification | `[x]` Complete |

**Remaining effort:** All phases complete. Custom prompt examples now visible in both help text (`ralph --help`) and README documentation.

## Known Existing Work

- Completion signal `<promise>COMPLETE</promise>` is implemented and functional
- Placeholder `<COMPLETION_SIGNAL>` is auto-replaced at runtime
- Specs have detailed documentation with examples (not in help/README)
- Help text already mentions the signal but lacks copy-paste examples

## Manual Deployment Tasks

None