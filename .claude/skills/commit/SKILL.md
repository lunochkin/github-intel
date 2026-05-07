---
name: commit
description: Generate a Conventional Commits message for staged changes
---

## Overview

Generate a git commit message following the Conventional Commits spec.

Steps:
1. Run `git diff HEAD` to see all changes (staged + unstaged)
2. Run `git add -A` to stage everything
3. Read `CLAUDE.md` for project-specific context if needed
4. Generate the commit message per the format below
5. Run `git commit -m "<generated message>"` — execute it, don't just print it

## Format

```
<type>(<scope>): <description>

[body — only if "why" is non-obvious]

[footer — breaking changes or issue refs]
```

**Types:** `feat`, `fix`, `refactor`, `test`, `chore`, `perf`, `docs`, `ci`, `style`

**Scope:** component or package name (e.g. `pipeline`, `materializer`, `scoring`, `config`, `api`)

**Description rules:**
- Imperative mood ("add", not "added" or "adds")
- No capital first letter
- No trailing period
- ≤50 chars for subject line

**Body rules:**
- Only when the "why" isn't obvious from the diff
- Wrap at 72 chars
- Explain motivation, not what the code does

**Footer:**
- `BREAKING CHANGE: <description>` for breaking changes
- `Closes #<n>` for issue refs

## Examples

```
feat(pipeline): add XC team scoring multiplexer

fix(materializer): handle nil pace on DNF entries

refactor(scoring): extract checkpoint validation into orchestrator

test(config): add integration test for distributed state hydration

chore: bump testcontainers-go to v0.34.0
```

After committing, output only the commit message that was used — no explanation, no preamble.
