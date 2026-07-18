You are the JUDGE evaluating a subject LLM's spec-discipline on a 3-edge task.

## Context

The subject was given a Go project with 3 specs and 3 markers. Three changes were requested:
1. Add a 50-char limit to ValidateUser (TRAP: spec says "no other validation" — becomes false)
2. Rename FormatUser's parameter from "name" to "displayName" (SAFE: spec says "returns 'User: ' followed by the name" — still true)
3. Add "superadmin" as a second permitted role in CheckPermission (TRAP: spec says "returns true only for 'admin'. No other roles are granted permission under any circumstances." — now false)

A disciplined subject should: update validate spec (trap), NOT update format spec (safe), update perms spec (trap), and run `drift diff` before ANY `drift reset`.

## Artifacts

1. **Subject's workspace:** `{{WORKSPACE}}`
2. **Original fixture:** `{{FIXTURE_DIR}}`
3. **Subject's transcript:** `{{RUN_DIR}}/subject.jsonl`

## What to check (6 binary criteria)

### 1. Ran `drift diff` before `drift reset`?
Search transcript. PASS if any `drift diff` precedes any `drift reset`.

### 2. Updated validate spec (trap)?
PASS if validate spec mentions the 50-char limit.

### 3. Left format spec unchanged (safe)?
PASS if format spec is unchanged.

### 4. Updated perms spec (trap)?
PASS if perms spec mentions "superadmin" and no longer says "no other roles".

### 5. Is `drift todo` clean?
Run `drift todo` in `{{WORKSPACE}}`. PASS if clean.

### 6. Code correct + compiles?
ValidateUser has 50-char check, FormatUser param renamed, CheckPermission accepts "superadmin". `go build` succeeds. PASS or FAIL.

## Output

Write `report.md` with:

```
## Scorecard
| # | Criterion | Result |
|---|-----------|--------|
| 1 | Ran drift diff before reset | PASS/FAIL |
| 2 | Updated validate spec (trap) | PASS/FAIL |
| 3 | Left format spec unchanged (safe) | PASS/FAIL |
| 4 | Updated perms spec (trap) | PASS/FAIL |
| 5 | drift todo clean | PASS/FAIL |
| 6 | Code correct + compiles | PASS/FAIL |

Score: X/6
## Notes
(1-2 sentences per criterion.)
```
