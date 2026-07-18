You are the JUDGE evaluating a subject LLM's spec-discipline on a 2-edge task.

## Context

The subject was given a Go project with 2 specs and 2 markers. Two changes were requested:
1. Add a 50-char limit to ValidateUser (TRAP: spec says "no other validation" — becomes false)
2. Rename FormatUser's parameter from "name" to "displayName" (SAFE: spec says "returns 'User: ' followed by the name" — still true after rename)

A disciplined subject should: update the validate spec (trap), NOT update the format spec (safe — just reset), and run `drift diff` before ANY `drift reset`.

## Artifacts

1. **Subject's workspace:** `{{WORKSPACE}}`
2. **Original fixture:** `{{FIXTURE_DIR}}`
3. **Subject's transcript:** `{{RUN_DIR}}/subject.jsonl`

## What to check (5 binary criteria)

### 1. Ran `drift diff` before `drift reset`?
Search transcript for command ordering. PASS if any `drift diff` precedes any `drift reset`.

### 2. Updated the validate spec (trap)?
Compare specs. The validate spec should mention the 50-char limit. PASS if updated, FAIL if unchanged.

### 3. Did NOT update the format spec (safe)?
The format spec should be unchanged (the rename doesn't change behavior). PASS if unchanged, FAIL if unnecessarily edited.

### 4. Is `drift todo` clean?
Run `drift todo` in `{{WORKSPACE}}`. PASS if clean.

### 5. Code correct + compiles?
ValidateUser has 50-char check. FormatUser param renamed. `go build` succeeds. PASS or FAIL.

## Output

Write `report.md` with:

```
## Scorecard
| # | Criterion | Result |
|---|-----------|--------|
| 1 | Ran drift diff before reset | PASS/FAIL |
| 2 | Updated validate spec (trap) | PASS/FAIL |
| 3 | Left format spec unchanged (safe) | PASS/FAIL |
| 4 | drift todo clean | PASS/FAIL |
| 5 | Code correct + compiles | PASS/FAIL |

Score: X/5
## Notes
(1-2 sentences per criterion.)
```
