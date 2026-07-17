> **Note:** This file predates the Driftpin → Drift rename (breaking change). Historical references to "Driftpin" / "driftpin" / ".driftpin" / "*.pin.xml" / "filament" are retained as-is.

# Observation 0004 — phase3-recovery

Date: 2026-07-16
Runs:
- `/workspaces/filament/eval/runs/phase3-recovery-0`
- `/workspaces/filament/eval/runs/phase3-recovery-1`
- `/workspaces/filament/eval/runs/phase3-recovery-2`

## Known issues

No sandbox escapes, no tainted runs, no harness breakage. All three runs are valid and included in convergent analysis.

Methodology / scope caveats (none disqualifying, but worth recording):
- **Run 2 — task failure, NOT a tool failure.** The poker binary panics at runtime (`slice bounds out of range [5:4]` in `Game.DrawRound`, `game.go:112`) on most playthroughs because discard-index removal assumes descending order. The subject never smoke-tested the binary. This is a Go-code bug and a smoke-test gap, not a driftpin defect: `drift todo` correctly reports the code is in sync with its specs (it is — the specs describe intended behavior, and the code matches the specs' shape). Run 2's driftpin findings remain fully valid; its tool usage was exemplary (it even caught and repaired a missed `draw_round` link via `drift list`). The failure is instructive about *expectations* (see Divergent findings) and is retained, not excluded.
- **Run 1 — transient environment quirk.** A `file: command not found` occurred when the subject tried to inspect the binary type; it recovered in a single step via `./drift --help`. No impact on the run.
- **Run 0 — coverage gap by task design.** The single-file `main.go` calculator task exercised no `<module>`/`<import>` path. Run 0's findings about the single-file flow are valid, but it contributes no evidence on multi-file/import workflows (Runs 1 and 2 do). Future phases should standardize on multi-module tasks so import-system evidence keeps flowing.

## Convergent findings

| Theme | Runs | Priority |
|---|---|---|
| `drift list` (with `[unlinked]`/`[synced]` annotations) works as the recovery/inspection loop — closed-loop `list -> spot [unlinked] -> link` is now functional (positive signal; phase2's #1 ask, now shipped and confirmed) | 0, 1, 2 | High (positive) |
| `drift skill` remains the load-bearing self-describing surface; all cold-start subjects discovered and relied on it; binary is self-describing for cold-use happy path AND recovery | 0, 1, 2 | High (positive) |
| All three runs achieved clean `drift todo` with zero tool errors — the end-to-end workflow (init -> code -> specs -> markers -> link -> todo) is robust | 0, 1, 2 | High (positive) |
| Need for `--json` output across `drift list`/`drift todo`/`drift link` for reliable LLM-agent/CI consumption | 0, 1, 2 | High |
| Need for `drift validate` to lint `*.pin.xml` (XML well-formedness, import resolution, `<spec>` placement, module-name consistency) before `drift link` produces a cryptic failure | 0, 1, 2 | High |
| Marker-ID vs qualified-spec-ID (`module.specId`) confusion in `drift link`; spec-ID qualification rule lives only in `drift skill`, absent from brief `--help` | 0, 2 | High |
| 10-line marker hash window semantics under-documented (marker-line inclusion, blank/whitespace handling, short functions <10 lines, EOF, adjacent/overlapping markers) — risks false drift or mis-placement | 0, 1, 2 | Medium |
| `drift diff` / `drift todo --verbose` should show current-vs-baseline hashed content (or unified diff) per drifted edge — `todo` reports *that* something drifted, not *what* | 0, 1, 2 | Medium |
| File-discovery / scanning rules undocumented (recursive? `.gitignore`/`drift.ignore` respected? syntax? diamond-import dedup?) | 0, 1, 2 | Medium |
| Edge-case / lifecycle semantics undocumented (marker/spec deletion -> dangling link, re-running `init` idempotency, many-to-many links, `reset` baseline-collapse trigger) | 0, 1 | Medium |
| `drift new-spec` / `drift add-spec <module.id> "<text>"` CLI command to generate/append `<spec>` entries and reduce manual-XML errors | 1, 2 | Medium |

## Divergent findings

- **Run 0 (single-file calc, 7/7 PASS):** Only run that never touched the module/import system — by task design, not defect. Uniquely recommended `drift init --from-existing` to seed `drift.pin` from pre-existing `D!` markers / `*.pin.xml` for adopting driftpin on an existing codebase (a common real-world entry point this greenfield task did not exercise). Also uniquely asked for improved `drift link` failure ergonomics (suggest `drift list` + existing `*.pin.xml` files; `--dry-run` flag). Its single substantive friction — the `main.` prefix rule being inferable only from `drift skill` examples — converges with Run 2's spec-ID qualification finding.
- **Run 1 (multi-module URL shortener, 7/7 PASS):** Only run to flag a concrete *bug*: `drift list` prints `:0` for every spec line number, which reads as a parse failure and erodes trust (this echoes phase2's `line="0"` sentinel finding — now visible in `drift list` output, so the bug surfaced more prominently). Uniquely recommended: non-zero exit code on drift for CI gating (`--check`/`--exit-code`); non-Go marker examples (Python, TypeScript) in `drift skill`; explicit `.drift.ignore` syntax docs; and `drift new-spec`. Also made a workflow observation: specs were authored *after* the Go code (code-first, specs reverse-engineered from signatures) — legitimate since driftpin doesn't enforce ordering, but a spec-first workflow would make drift detection more valuable.
- **Run 2 (multi-module poker CLI, 6/7 — task FAIL):** The defining divergent finding is the **expectations gap**: the subject treated "compiles + `drift todo` clean" as task completion and never ran the binary, shipping a panic-prone game. This suggests `drift todo`'s clean output gives agents false confidence that the *task* is done when it only certifies *spec/code alignment*. Uniquely recommended: (a) `drift init` template should demonstrate the module/import system (scaffold `main.pin.xml` importing a `core/core.pin.xml` stub) so the multi-file pattern is the default mental model; (b) `drift todo`/`drift list` should emit a copy-pasteable "next action" hint for `[unlinked]` markers (e.g. `-> run drift link draw_round core.draw_round`); (c) `drift skill` should state that `drift todo` certifies alignment only, not runtime correctness, and recommend a separate smoke test; (d) `drift show <marker>` command; (e) high-level `drift.pin` schema docs. Notably Run 2 is the only run that actually *exercised* the recovery loop (caught a missed `draw_round` link via `drift list` and repaired it) — proof the newly-shipped `drift list` delivers the recovery affordance phase2 asked for.

## Prioritized recommendations (consolidated)

1. [High] Add `--json` output mode to `drift list`, `drift todo`, and `drift link` (specs, markers, links, sync state, drift deltas) — agents are the primary audience and currently must regex-parse human tables. — runs 0, 1, 2
2. [High] Add `drift validate` to lint `*.pin.xml` independent of markers/links: XML well-formedness, import-path resolution (relative + transitive + cycle detection), `<spec>` direct-child placement, module-name/spec-id prefix consistency, with file+line reporting. — runs 0, 1, 2
3. [High] Clarify marker-ID vs qualified-spec-ID in `drift link` usage and errors, and surface the `module.specId` qualification rule in the brief `--help` (not only `drift skill`). Usage line should read `drift link <markerId> <module.specId>`; on a bad spec ID, suggest the closest match ("did you mean `core.draw_round`?"). — runs 0, 2
4. [High] Fix `drift list` showing `:0` for spec line numbers — either track and display the real XML line of each `<spec>`, or omit the line for specs and keep it only for markers (where it is meaningful). Small fix, outsized trust impact. — run 1 (echoes phase2 `line="0"` finding)
5. [Medium] Document the 10-line marker hash window precisely in *both* `drift help` and `drift skill`: marker-line inclusion, trailing-newline/leading-trailing-whitespace handling, tab-vs-space, behavior when fewer than 10 lines follow, adjacent/overlapping markers. — runs 0, 1, 2 (recurring from phase2)
6. [Medium] Add `drift diff` / `drift todo --verbose` showing current-vs-baseline hashed content (or unified diff) plus both hashes per drifted edge. — runs 0, 1, 2 (recurring from phase2)
7. [Medium] Document file-discovery rules: recursive scan from project root, `.gitignore`/`drift.ignore` handling and pattern syntax with a 3-line example, diamond-import dedup by absolute path, treatment of non-imported `*.pin.xml` as top-level modules. — runs 0, 1, 2
8. [Medium] Document edge-lifecycle & baseline semantics: what `drift todo` reports on marker deletion (dangling link?), what happens to a link when its spec is removed, `drift init` idempotency/no-overwrite, many-to-many links, and a concrete definition of when `drift reset` collapses baselines. — runs 0, 1
9. [Medium] Make `drift init` scaffold a module/import demonstration (a `main.pin.xml` that imports a `core/core.pin.xml` module stub) so the multi-file pattern is the default mental model rather than something inferred from `drift skill` examples. — run 2
10. [Medium] Have `drift todo`/`drift list` emit a copy-pasteable "next action" hint for `[unlinked]` markers (e.g. `-> run drift link <id> <module.spec>`), converting a diagnostic into a remedy. Low implementation cost, high payoff for agent autonomy. — run 2
11. [Medium] Add `drift new-spec`/`drift add-spec <module.id> "<text>"` to generate/append `<spec>` entries (and optionally scaffold a module file) from the CLI, reducing manual-XML errors. — runs 1, 2
12. [Low] State in `drift skill` that `drift todo` certifies spec/code *alignment only, not runtime correctness*, and recommend a separate build+run smoke test — to counter the "compiles + drift-clean = done" false-confidence seen in Run 2. — run 2
13. [Low] Add non-zero exit code on drift (`--check`/`--exit-code`) for pre-commit hooks / CI gating. — run 1
14. [Low] Add non-Go marker examples (Python, TypeScript) to `drift skill` to internalize the comment-style-agnostic marker regex across ecosystems. — run 1
15. [Low] Add `drift show <marker>` for quick single-marker/spec/edge inspection. — run 2
16. [Low] Document the `drift.pin` schema at a high level ("tool-managed XML with `<specs>`/`<markers>`/`<links>`/`<resolutions>`; do not hand-edit; `init`/`link` rewrite it") for debugging. — run 2 (recurring from phase2)
17. [Low] Improve `drift link` failure ergonomics: on failure suggest `drift list` and which `*.pin.xml` files exist; add `--dry-run` to preview edge creation. — run 0
18. [Low] Consider `drift init --from-existing` to seed `drift.pin` from pre-existing `D!` markers / `*.pin.xml` for adoption on existing codebases. — run 0

## Next steps

The headline signal is strongly positive and convergent: **driftpin's investment in inspection/recovery affordances is paying off.** Phase2's single highest-priority ask (`drift list`/`drift status`) was implemented, and all three phase3 cold-start subjects used `drift list` successfully — Run 2 is the showcase, performing the exact closed-loop recovery (`drift list` -> spot `[unlinked]` -> `drift link` repair) the phase name implies. Every run reached a clean `drift todo` with zero tool errors, and the multi-module import system was correctly inferred from `drift skill` in both multi-file tasks (Runs 1, 2). The tool's onboarding + recovery story now works end-to-end for cold LLM agents.

The remaining work is the *next layer down* — machine-readable output, pre-link validation, and the documentation cluster that has now recurred across two phases. In priority order:

1. **Ship `--json` and `drift validate` as the next milestone ("machine-readable output + pre-link validation").** These are the only two themes *all three* runs flagged, and they unblock the two audiences driftpin is built for: LLM agents (reliable programmatic consumption) and CI (gating). `drift validate` in particular closes the gap where a malformed `*.pin.xml` surfaces only as a cryptic `drift link` failure or `drift todo` silence.
2. **Fix the two highest-friction, cheapest wins in the core loop immediately:** (a) surface the `module.specId` qualification rule in brief `--help` and make `drift link`'s usage line + bad-spec-ID error explicit/suggestive (Runs 0, 2 — `link` is the most error-prone step); (b) fix the `:0` spec line numbers in `drift list` (Run 1 — a trust-eroding bug with an outsized payoff-to-effort ratio).
3. **Batch the recurring documentation gaps into one "drift skill — Internals" pass:** the 10-line hash window, file-discovery/`.drift.ignore` rules, and edge-lifecycle/`reset` semantics. These have now been flagged in both phase2 and phase3 and remain unaddressed; they are cheap to write and remove cold-start uncertainty in exactly the spots agents guess at.
4. **Add `drift diff` alongside the `--json` work** so that *when* drift occurs, agents see what changed rather than only that it changed — the natural complement to the now-working `drift list` inspection loop.
5. **Address the expectations gap exposed by Run 2** with a one-line note in `drift skill` that `drift todo` certifies spec/code alignment, not runtime correctness, plus a smoke-test recommendation. This is nearly free and prevents agents from over-trusting a green `todo` as task completion — the single most consequential non-tool observation in this batch.
6. **Defer Low-priority polish** (`drift new-spec`, `drift show`, `--exit-code`, non-Go examples, `drift.pin` schema docs, `--from-existing`) until the JSON+validate+docs milestone lands; they are real but not blocking cold-use or recovery success.

Two process notes for the eval itself: (i) Run 0's single-file task produced no module/import evidence — future phases should standardize on multi-module tasks to keep that signal flowing; (ii) consider adding a "did you run the binary?" rubric item or smoke-test requirement so task-level runtime failures (Run 2's panic) are caught by the harness rather than discovered only at grading time.
