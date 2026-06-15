---
description: Never use --no-verify or -n to bypass git hooks. Fix pre-commit failures at their source.
condition:
  - "--no-verify"
scope:
  - "tool:bash"
interruptMode: "always"
---

# Thou Shalt Not Bypass the Gate

`git commit --no-verify` (and its short form `-n`, `git commit --no-gpg-sign`) are forbidden in this workspace.

## Why

The pre-commit hook is the last line of defense. Every check it runs — lint, typecheck, test-coverage, security, arch — catches real problems. Bypassing the hook means shipping a known defect.

Bypassing once breaks the discipline of fixing at the source. The hook only fails when something is wrong: a missing dependency, a genuine lint, a test regression. Those are fixable in seconds — but only if you read the error and fix the root cause instead of throwing a bypass flag.

## What to do instead

1. Read the hook output carefully. Every failure names the tool and the specific violation.
2. Fix the underlying problem, not the symptom.
   - Missing tool? Install it (`go install`, `pip install`, `npm install -g`).
   - Lint error? Fix the code.
   - Test failure? Fix the logic.
3. Stage the fix and retry the commit.

There is no legitimate use of `--no-verify` in this repository. If the hook is genuinely broken (false positive), fix the hook config, not the bypass flag.
