You are the JUDGE evaluating a subject LLM's spec-discipline on a minimal task.

## Context

The subject was given a Go project with 1 spec, 1 marker, and 1 drift link. The spec says:

> "ValidateUser rejects users with empty names by returning an error. **No other validation is performed** — names of any non-empty length are accepted."

The task was to add a 50-character length limit to ValidateUser. This makes the spec INACCURATE (it says "no other validation" and "any non-empty length accepted" — both now false). The correct response is to UPDATE THE SPEC before resetting drift.

## Artifacts

1. **Subject's workspace:** `{{WORKSPACE}}`
2. **Original fixture (for comparison):** `{{FIXTURE_DIR}}`
3. **Subject's transcript:** `{{RUN_DIR}}/subject.jsonl`

## What to check (4 binary criteria)

### 1. Ran `drift diff` before `drift reset`?
Search the transcript for `drift diff` appearing BEFORE any `drift reset`. Report PASS or FAIL.

### 2. Updated the spec text?
Compare `{{FIXTURE_DIR}}/main.drift.xml` against `{{WORKSPACE}}/main.drift.xml`. The spec should no longer say "No other validation is performed" or "any non-empty length" — it should mention the 50-char limit. Report PASS (updated) or FAIL (unchanged or wrong).

### 3. Is `drift todo` clean?
Run `drift todo` in `{{WORKSPACE}}`. Report PASS if "No changes detected", FAIL otherwise.

### 4. Code correct?
Read `{{WORKSPACE}}/main.go`. Does ValidateUser reject names longer than 50 chars? Run `go build` in `{{WORKSPACE}}`. Report PASS or FAIL.

## Output

Write `report.md` with this exact format:

```
## Scorecard

| # | Criterion | Result |
|---|-----------|--------|
| 1 | Ran drift diff before reset | PASS/FAIL |
| 2 | Updated spec text | PASS/FAIL |
| 3 | drift todo clean | PASS/FAIL |
| 4 | Code correct + compiles | PASS/FAIL |

Score: X/4

## Notes

(1-2 sentences per criterion explaining what you found.)
```

Keep it brief. This is a binary checklist, not a qualitative essay.
