# Debug a drift finding

<!-- #F id:h1i2j3kl drift.spec_drift drift.site_drift drift.missing drift.orphan drift.malformed -->
<!-- #F id:3jmn6fyk drift.not_in_state -->
<!-- #F id:t4y2bzgg drift.state_file_missing -->
<!-- #F id:vu9girxd drift.both_drift -->

When `filament check` reports a finding, it prints prose explaining what changed and what to do. This document explains each finding type and how to fix it.

## SPEC_DRIFT

The spec clause changed since the marker was last reviewed.

**What it means:** The spec clause's content was modified. The code at this marker location may no longer match the spec's intent.

**How to fix:**
1. Read the spec clause in the spec XML
2. Compare it against the code at the flagged file:line
3. If the code still matches, run `filament resolve --spec <marker_id>`
4. If the code doesn't match, update the code first, then resolve

## SITE_DRIFT

The content near the marker changed since it was last reviewed.

**What it means:** The code near the marker was modified. The spec clause(s) the marker traces to may no longer describe what the code does.

**How to fix:**
1. Read the spec clause(s) listed in the finding
2. Compare them against the code at the flagged file:line
3. If the code still matches the spec, run `filament resolve --site <marker_id>`
4. If the code no longer matches, update the spec first, then sync and resolve

## MISSING

A clause is in the spec but has no #F marker in any scanned file.

**What it means:** No file in the workspace claims to implement this clause. Either the clause is unimplemented, or it's implemented but not tracked.

**How to fix:**
1. Run `filament add <clause_id>` to generate a marker
2. Paste it into the file that implements the clause
3. Run `filament init` or `filament resolve --site <marker_id>`

## ORPHAN

A marker references a clause that doesn't exist in the spec.

**What it means:** The spec removed or renamed a clause, but the marker still references the old id.

**How to fix:**
1. If the clause was renamed, update the marker's clause id to the new name
2. If the clause was deleted, remove the marker from the file

## MALFORMED

A marker has invalid syntax.

**What it means:** The line matches the `#F id:` pattern but doesn't conform to the full marker syntax (e.g., marker id is not 8 characters, or no clause ids are present).

**How to fix:**
1. Run `filament add <clause_id>` to generate a valid marker
2. Replace the malformed line with the valid one

## NOT_IN_STATE

A marker exists in a file but has no entry in the state file.

**What it means:** The marker was added to the file but the state file wasn't updated.

**How to fix:**
1. Run `filament resolve --site <marker_id>` to register the marker
2. Or run `filament init` to regenerate the entire state file

## STATE_FILE_MISSING

No `.filament` file found in the spec's directory.

**How to fix:**
1. Run `filament init` to create the state file
