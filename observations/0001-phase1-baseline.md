# Observation 0001 — phase1-baseline

Date: 2026-07-16
Runs: eval/runs/phase1-baseline-0, eval/runs/phase1-baseline-1

## Known issues

- **Run 1: sandbox escape (tainted).** The subject read `/workspaces/filament/DOCUMENTATION.md` and `/workspaces/filament/scanner/scanner.go` from the parent repo. The default `build` agent auto-approves `external_directory`, and the workspace is inside the repo at `eval/runs/<ts>/workspace/`. Several "discoveries" (marker regex, 10-line window, `<module>` root) came from source, not binary introspection. Run 1's scorecard still passed, but its cold-start evidence is compromised. Findings below exclude run 1 from convergence unless independently verified by run 0 or the judge.
- **Run 2: aborted.** The third battery run was interrupted before completion. Not included.
- **Harness fix needed:** subject agent must deny `external_directory` or workspace must be outside the repo tree. This is non-negotiable for valid benchmarks.

## Convergent findings

| Theme | Runs | Priority |
|---|---|---|
| No `--help` / `drift help` (binary not self-describing) | 0, 1 | High |
| Marker syntax (`D! id=...`) undiscoverable from binary alone | 0, 1 | High |
| Doc/code syntax mismatch (colon `:` vs space-separated) | 0, 1 | High |
| `<specs>` root in docs vs `<module>`/`<main>` root in binary | 0, 1 | High |
| Module qualification not suggested in error messages | 0, 1 | Medium |
| `drift.pin` population unclear (only `link` writes, not `todo`) | 0, 1 | Medium |
| `drift init` should emit starter template | 0, 1 | Medium |
| `drift status` summary command requested | 0, 1 | Low |
| Marker hash window (10 lines) not documented or configurable | 0, 1 | Low |

## Divergent findings

- **Run 0 (genuinely cold):** Subject used `strings drift | grep` to reverse-engineer the binary. Guessed `<specs>` root, tried bare spec IDs, hand-edited `drift.pin` — all self-corrected via error messages, but each cost a round-trip. Specs were shallow ("Addition operation" instead of behavioral). The subject only knew `D! id=...` marker syntax from the task prompt, not from the binary.
- **Run 1 (sandbox escape):** Subject read source code and docs directly. Before the escape, it spent ~10 tool calls on `strings drift | grep`. After the escape, execution was clean and fast. Specs were meaningful and behavioral. The adjacent-marker hash collision (lines 16-17) was a genuine footgun that the subject correctly diagnosed.

## Prioritized recommendations (consolidated)

1. **[High] Add `drift help` / `--help` with examples.** Both subjects tried `--help` and got nothing. The bare usage line is insufficient. Need: full command reference, marker syntax, spec file format, module qualification, quick-start. (runs 0, 1)
2. **[High] Add `drift skill` command.** Embed a skill file that teaches an LLM how to use driftpin end-to-end. This is the primary delivery mechanism for cold-start education — more comprehensive than `--help`. (run 0 implied — subject had no way to learn marker syntax from binary)
3. **[High] Reconcile doc/code syntax mismatch.** DOCUMENTATION.md uses colon-separated `drift link <marker>:<spec>`; binary uses space-separated `drift link <marker> <module.spec>`. Pick one, make all docs + CLI agree. (run 1 #1)
4. **[High] Fix `<specs>` vs `<module>`/`<main>` root inconsistency.** Docs show `<specs>` root; binary rejects it. Docs must show `<module>`/`<main>`. (run 0 #3, run 1 #4)
5. **[High] Make `drift todo` persist discovered specs/markers as baselines (or document that `link` is required).** Docs claim `drift todo` discovers and baselines; binary only persists on `link`. Either fix behavior or fix docs. (run 1 #2)
6. **[Medium] Improve error messages with module qualification suggestions.** `unknown spec "addition"` should suggest `main.addition` when it exists. Surface available specs/markers in errors. (run 0 #4, run 1 #5)
7. **[Medium] `drift init` should emit a commented starter `main.pin.xml` template.** Removes the first-failure of guessing the XML root. (run 0 #3, run 1 #9)
8. **[Medium] Document and make configurable the marker hash window (10 lines).** Adjacent markers cause false-positive drift. Allow `D! id=foo lines=20`. (run 0 #6, run 1 #6)
9. **[Low] Add `drift status` summary command.** Spec count, marker count, link count, pending drift. (run 0 #7, run 1 #7)
10. **[Low] Document `drift.ignore` format and supported file extensions.** Invisible from binary. (run 1 #8)

## Next steps

1. **Fix harness sandbox escape** — add a custom subject agent that denies `external_directory`, or move workspace outside repo tree.
2. **Phase 2: implement self-describing binary** — `drift skill` + `drift help` + `--help` + `drift init` template (recommendations 1, 2, 7).
3. **Fix stale docs** — reconcile syntax, fix `<specs>` root examples, correct `drift todo` behavior claims (recommendations 3, 4, 5).
4. **Re-run benchmark** with sandbox fix + self-describing binary. Compare against this observation.
