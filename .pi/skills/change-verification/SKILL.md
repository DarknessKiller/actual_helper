---
name: change-verification
description: Verify a finished change through an independent subagent that checks every changed line against CODING_STANDARDS.md and AGENTS.md, runs the Verify block, and reports PASS/FAIL with file:line findings. Use after a change is done — before handoff or commit — and for "review this diff".
---

# Change verification

Fire this after every change, unasked. A fresh subagent grades the change; the main agent
fixes blockers. `CODING_STANDARDS.md` and `AGENTS.md` stay the source of truth.

## 1. Scope the change

`git status --porcelain` is the scope for uncommitted work. For a task already committed,
diff against the commit it started from: `git diff --name-only <base>`.

**Done when:** the full list is in hand, or `NO CHANGES` is reported and the run stops.

## 2. Dispatch the verifier

Call the subagent tool (`Agent` in pi) once, with a fresh agent. Paste this prompt:

```text
You are the independent verifier for a change in <repo root>.

The author hands off this task: <one-line task summary>
Base commit for the diff (default `HEAD`): <base>
Changed paths:
<paste the list>

Work read-only: read files, run commands, cite evidence. You are the final verifier:
answer in this report; dispatch no further subagents.

1. Read AGENTS.md and CODING_STANDARDS.md.
2. Read the change: `git diff <base>` for tracked files, each untracked file in full.
3. Account for every changed line against every CODING_STANDARDS.md section — errors,
   logging, layers, HTTP handlers, config and sessions, tests, synthetic test data — and
   against the AGENTS.md architecture and conventions. A line is either compliant or a
   finding with `file:line`.
4. Run every command in the Verify block at the end of CODING_STANDARDS.md and record
   each result with its failing output.
5. Scan fixtures and docs for real personal data (command scans the repo; a hit outside
   the change is a note, inside is a blocker):
   rg -n '\b\d{12,}\b' --glob '*_test.go' --glob '*.csv' --glob '**/testdata/**'
   Every hit is a placeholder from the table in CODING_STANDARDS.md; a real value is a blocker.

Report exactly:

VERDICT: PASS | FAIL
COMMANDS: <command> <pass|fail>   (one line per Verify-block command)
FINDINGS:
- <blocker|should-fix|note> <file>:<line> — <standard section> — <problem>; <fix>
NOT CHECKED: <what you could not verify, and why>

PASS requires every verify command green and zero blockers. A finding names file:line or
command output. Every changed line is accounted for.
```

**Done when:** the verdict block is in hand.

## 3. Act on the verdict

**Done when:** a PASS is handed off, or the remaining findings are reported to the user.

- `PASS` — hand off, stating the verdict and each command result.
- `FAIL` — fix every blocker, then dispatch a second verifier over the updated scope. Hand
  off only after the re-run returns PASS.
- `should-fix` / `note` — fix inside the task's scope; otherwise list them in the handoff.

## Without a subagent tool

Run the same prompt inline in a fresh turn, then apply step 3.
