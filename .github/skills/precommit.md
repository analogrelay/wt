---
name: precommit
description: This skill MUST be run before any `git commit` in this repository. It is a blocking gate — commits may not proceed unless all checks pass or the user explicitly authorizes a broken commit.
---

## Checks (run in order)

Run all of the following. Do not skip any step:

1. **`go build ./...`** — the project must compile
2. **`go vet ./...`** — no vet warnings
3. **`go test ./...`** — all tests must pass
4. **`golangci-lint run ./...`** — no lint violations

## On failure

If any check fails:

1. **Attempt to fix** the issue automatically, provided the fix stays within the intended scope of the commit. Do not introduce unrelated changes.
2. **Re-run all checks** after fixing. Repeat up to 3 attempts.
3. If the issue **cannot be fixed** without going out of scope, **stop and report** the failure to the user. Include:
   - Which check failed
   - The error output
   - Why the fix is out of scope
4. **Ask the user** whether to commit anyway ("broken commit").
   - If the user **authorizes** a broken commit → proceed with the commit.
   - If the user **does not respond** or **declines** → **do not commit**. The agent must not commit without explicit user authorization of a broken commit.

## GPG signing

- **NEVER** disable `gpgsign` — do not run `git config commit.gpgsign false` or pass `--no-gpg-sign`. This is a hard rule with no exceptions.
- Do not proactively check signing configuration. Just attempt the commit.
- If the commit fails due to a GPG/signing error, **report the failure to the user** and do not retry or work around it.

## Usage

This skill is meant to be invoked by other agents/skills before they commit. It can also be invoked manually by the user to validate the working tree.
