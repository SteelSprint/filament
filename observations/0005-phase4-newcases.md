> **Note:** This file predates the Driftpin → Drift rename (breaking change). Historical references to "Driftpin" / "driftpin" / ".driftpin" / "*.pin.xml" / "filament" are retained as-is.

# Observation 0005 — phase4-newcases

Date: 2026-07-16
Runs:
- `/workspaces/filament/eval/runs/phase4-newcases-0`
- `/workspaces/filament/eval/runs/phase4-newcases-1`
- `/workspaces/filament/eval/runs/phase4-newcases-2`
- `/workspaces/filament/eval/runs/phase4-newcases-3`

## Known issues

- **Run 1 (phase4-newcases-1) — answer leakage via `setup.sh`.** The fixture's `setup.sh` contained an explicit comment (`# WRONG link: palindrome_func linked to main.reverse instead of main.palindrome`) that named the bad link before the subject had to reason about it from code. The subject was honest about this in its debrief. `drift list` would have revealed the mismatch independently (`[unlinked]` spec + semantically odd link), so the run still produced valid observations about CLI driving, but it is a **weakened test of cold reasoning about marker/spec correctness**. Findings from run 1 about *tool UX* (line="0", --json, validate heuristic) are retained; any claim that "the agent reasoned out the bad link" is excluded. **Harness issue, not a sandbox escape.** Fixture builders should strip spoiler comments.
- **Run 2 (phase4-newcases-2) — missing `go.mod` in fixture.** The task prompt asserted "the project compiles with `go build`," but the workspace had no `go.mod`, so `go build` failed until the subject ran `go mod init converter`. The subject correctly attributed this to the environment, not the tool. **Harness setup discrepancy.** No findings excluded — the driftpin workflow was unaffected.
- **Run 3 (phase4-newcases-3) — silent re-baselining of untouched markers.** After editing only `div`, `drift todo` flagged `sub_func` and `mul_func` as drifted because the 10-line hash window bled into the changed `div` body. The subject correctly diagnosed this and `reset` all three edges. This is **a real tool behavior, not a taint** — it is recorded as a High-priority finding below, not as a methodology problem.
- No sandbox escapes, no tainted runs in the sense of requiring exclusion of all findings from a run. All four runs' tool-UX findings are retained; only the run-1 "reasoned out the link" claim is discounted.

## Convergent findings

| Theme | Runs | Priority |
|---|---|---|
| `drift skill` is the real onboarding surface but not promoted by the bare `drift` / `--help` output; cold-start costs an extra round-trip | 0, 2, 3 | High |
| No way to see *what* changed when a marker drifts — only *that* it changed; want `drift diff`/`drift inspect`/`drift show <marker>` printing the hashed region and old-vs-new content | 0, 2, 3 | High |
| 10-line hash window semantics under-documented (blank lines, short files, overlap with adjacent markers, end-of-file) | 0, 2, 3 | Medium |
| Collateral drift: a localized edit flags adjacent markers because the 10-line window bleeds past the marker's own function/block boundary | 2, 3 | High |
| `line="0"` persisted in `drift.pin` / shown in `drift list` for specs (and in run 0 also markers); looks like a bug, hampers external tooling | 0, 1 | Medium |
| `main.` auto-qualification convention (specs in `main.pin.xml` become `main.<id>`) is non-obvious without `drift skill` | 0, 1 | Low |
| `--json` / structured output wanted for `drift list` and `drift todo` to avoid fragile regex parsing by LLM agents / CI | 1, 3 | Medium |
| `drift reset` ergonomics: silent success on a state-mutating command (no confirmation line) and no bulk `--all`/`--yes` for multi-edge refactors | 2, 3 | Medium |
| `drift.pin` editing policy unclear ("do not edit by hand" with no why) and XML/validation errors in `*.pin.xml` are not surfaced with line-precise feedback | 0, 2 | Medium |
| Fixture `setup.sh` left in the agent's workspace leaks setup intent / creates ambiguity about whether to run it | 1, 2 | Low (harness) |

## Divergent findings

- **Run 0** — Stale-entry handling is a severe UX trap: removing a `<spec>` from a `.pin.xml` orphans its `drift.pin` entry, after which *every* `drift list` / `drift todo` hard-fails with `spec in drift.pin not found on disk` and emits no other output. No `drift remove` command exists; the subject had to violate the "do not edit drift.pin by hand" rule to recover. Also: marker↔spec cardinality (can one marker link to multiple specs?) is undocumented and was hit directly with the abandoned `main.error_handling` spec; batch-link ergonomics (`drift link` one-at-a-time is tedious for 6 links).
- **Run 1** — The whole case rests on `drift todo` happily reporting "in sync" for a *wrong-but-consistent* link; a `drift validate` / mismatch-warning heuristic (e.g. warn when a marker ID and its linked spec ID share no token, or when a spec has zero links while a marker points at an already-targeted spec) would catch the class of error this case tests. Also: `drift reset` baseline-collapse mechanics undocumented ("collapses baselines when all edges resolved" — what actually changes in `drift.pin`?); `drift skill --save` / persistable skill output; a `drift links`-only subcommand.
- **Run 2** — Explicit marker ranges (opt-in `// D! id=foo lines=20` or paired `// D! end=foo`) for functions longer than 10 lines or for precise anchoring during refactors; a top-line `drift status` summary count (N synced, M drifted) in addition to per-edge `[synced]` tags.
- **Run 3** — Intentional spec divergence workflow is undefined: when code deliberately extends behavior beyond the spec (this task's `div` now handles negative zero, but the spec still says only "division by zero"), the docs don't say whether to update the spec text and `reset`, or just `reset` to re-baseline. The subject chose `reset` (correct per the criteria) but the decision is currently reinvented per run. Also: configurable hash window; `--version` and idempotent `drift init` ("already initialized" vs clobber); `drift check` as an alias for `todo`.

## Prioritized recommendations (consolidated)

1. **[High]** Constrain the 10-line hash window so a marker's hash does not extend past the end of its own function/block (stop at the next `// D!` marker line, or at a blank-line-then-declaration boundary). This eliminates collateral drift on adjacent markers and removes the need for users to `reset` edges they never touched. — runs 2, 3
2. **[High]** Add `drift inspect`/`drift show`/`drift diff <marker>` that prints the exact lines being hashed plus current vs. last-baselined hash (ideally a unified diff of the marker's scope). Turns "the marker has changed" into actionable information and is essential for distinguishing true drift from window-overlap noise. — runs 0, 2, 3
3. **[High]** Promote `drift skill` to the top of the bare `drift` / `--help` output (e.g. a bolded "First time? Run `drift skill` for the full guide." line, or print the skill guide by default). The command surface alone is insufficient for cold agents; the workflow, marker contract, and module-qualification rules live only in `skill`. — runs 0, 2, 3
4. **[High]** Add a `drift remove <spec-id|marker-id>` command and/or auto-clean stale entries; separately, stop hard-failing the *entire* `drift list`/`drift todo` on a single stale reference — report it as one diagnostic line and continue rendering the rest of the state. Today one orphaned spec suppresses all useful output and forces a hand-edit of the "do not edit by hand" state file. — run 0 (High severity, single-run but acute)
5. **[High]** Add a `drift validate` / mismatch-warning heuristic that catches wrong-but-consistent links (warn when marker ID and linked spec ID share no token; warn when a spec has zero links while a marker points at a spec already targeted by another marker; warn on `[unlinked]` specs). `drift todo`'s "in sync" verdict cannot detect semantic mismatches. — run 1
6. **[Medium]** Document the 10-line hash window edge cases explicitly in `drift skill` with a worked example: (a) fewer than 10 lines remaining, (b) whether blank/comment lines count, (c) overlap with adjacent markers' windows, (d) end-of-file shortfalls. Include the exact lines hashed and resulting SHA1. — runs 0, 2, 3
7. **[Medium]** Fix `line="0"` in `drift.pin` / `drift list`: persist the real line of each `<spec>` element and marker, or document explicitly that specs are whole-element hashes with no meaningful line number. Silence the ambiguity. — runs 0, 1
8. **[Medium]** Add `--json` / `--jsonl` output to `drift list` and `drift todo` for reliable LLM-agent and CI consumption (no fragile regex over arrows / `[synced]` / `[unlinked]` tags). — runs 1, 3
9. **[Medium]** Make `drift reset` emit a confirmation line (e.g. `Resolved: <marker> → <spec> (baseline updated)`) and add a bulk `drift reset --all` / `--yes` for verified multi-edge refactors. Silent success on a state-mutating command invites over-polling and one-by-one resets are tedious. — runs 2, 3
10. **[Medium]** Document the intended resolution model for *intentional* spec divergence in `drift skill`: when code deliberately extends behavior beyond the spec, update the spec text in `*.pin.xml` then `reset`, or just `reset` to re-baseline? Provide a short "resolving drift" decision tree. — run 3
11. **[Medium]** Document *why* `drift.pin` must not be hand-edited (state invariants) and surface `*.pin.xml` parse errors with the offending line/column rather than silently ignoring the spec. — runs 0, 2
12. **[Low]** Clarify the `main.` auto-qualification rule in `drift --help` ("specs in `main.pin.xml` are prefixed with `main.`"). — runs 0, 1
13. **[Low]** Add a `drift check` alias for `drift todo` (the name "todo" is unintuitive for a status/check operation) and/or a lightweight `drift validate` for CI that checks XML well-formedness, marker-ID uniqueness, and link validity without the full drift comparison. — runs 0, 3
14. **[Low]** Fixture hygiene: strip spoiler comments from `setup.sh` or relocate setup scripts outside the agent's working directory so agents can't short-circuit the reasoning the task is meant to test. — runs 1, 2
15. **[Low]** Single-run, lower-impact additions worth triaging: batch-link ergonomics (`drift link --from mapping.json`) [run 0]; explicit opt-in marker ranges (`lines=N` or paired `end=` markers) [run 2]; a top-line `drift status` summary count (N synced, M drifted) [run 2]; a `drift links`-only subcommand [run 1]; persistable `drift skill --save` [run 1]; document `drift reset` baseline-collapse mechanics ("current hash becomes the new baseline") [run 1]; `--version` and idempotent `drift init` ("already initialized" / `--force`) [run 3]; configurable hash window [run 3].

## Next steps

The four runs converge on three High-priority tool gaps that the authors should address first, in this order:

1. **Fix the hash-window scope (rec 1).** The collateral-drift behavior in run 3 (and the overlap edge cases flagged in runs 0 and 2) is the most consequential defect: it makes a localized edit look like a multi-marker event, forces users to reason about hashing internals before resolving, and silently re-baselines untouched markers. Constraining the window to the marker's own function/block is the single highest-leverage fix — it both removes false positives and shrinks the surface for every other recommendation.
2. **Ship `drift inspect`/`diff` (rec 2) together with the window fix.** Once the window is well-scoped, users still need to *see* what changed to judge whether drift is cosmetic or behavioral before resolving. The two features are mutually reinforcing and were independently requested in three of four runs.
3. **Onboarding (rec 3).** Promote `drift skill` to the top of the bare `drift` output. This is trivial to implement and removes one round-trip from every cold start; it was flagged in three of four runs.

After those, the next tier is the **state-file robustness cluster** (recs 4, 7, 11): stale-entry handling, `line="0"`, and `drift.pin` editing policy are all symptoms of the same "the persisted state file is fragile and the user has no safe recovery path" problem. A single pass that adds `drift remove`, auto-prunes stale references, stops hard-failing read commands, and persists real line numbers would close the cluster.

A second tier is **agent-integration ergonomics** (recs 8, 9, 10, 13): `--json` output, `drift reset` confirmation + bulk, an intentional-divergence decision tree, and a `drift check` alias. These are cheap and directly improve LLM-agent scriptability and CI use; they should be batched into one "agent/CI friendliness" work item.

Finally, the harness side (rec 14) should be addressed by the eval authors, not the tool authors: strip `setup.sh` spoiler comments and ensure fixtures have a working `go.mod` so "the project compiles with `go build`" is true on first run.
