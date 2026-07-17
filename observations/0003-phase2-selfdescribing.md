> **Note:** This file predates the Driftpin → Drift rename (breaking change). Historical references to "Driftpin" / "driftpin" / ".driftpin" / "*.pin.xml" / "filament" are retained as-is.

# Observation 0003 — phase2-selfdescribing

Date: 2026-07-16
Runs:
- `/workspaces/filament/eval/runs/phase2-selfdescribing-0`
- `/workspaces/filament/eval/runs/phase2-selfdescribing-1`
- `/workspaces/filament/eval/runs/phase2-selfdescribing-2`

## Known issues

No harness issues, sandbox escapes, or tainted runs were discovered. All three runs completed the task cleanly:
- Run 0: single-file Go calculator, 5 specs/markers/links, `drift todo` clean on first check.
- Run 1: multi-module Go URL shortener, 8 specs/markers/links, clean on first check.
- Run 2: multi-module Go poker CLI (6 modules), 27 specs/markers/links, clean on first check.

Minor non-driftpin observations (not affecting convergence):
- Run 1 noted the shortener used 6 hex chars rather than the task's "alphanumeric" code — a spec↔task drift, not spec↔code drift. Driftpin correctly tracked spec↔code; it cannot (and shouldn't) track task↔spec. Noted for tool-scope awareness only.
- Run 0 noted `drift todo` prints "No drift: N specs…" rather than the task prompt's literal "No changes detected." — a wording mismatch that could cause false FAILs in automated harness checks. Flagged in recommendations.
- Run 2 noted a harmless marker/spec namespace mismatch (`main_entry` marker → `main.entry` spec), which works because the namespaces are independent but reduces human readability. Not a tool bug.

All three runs are included in convergent analysis.

## Convergent findings

| Theme | Runs | Priority |
|---|---|---|
| Need for `drift unlink` / destructive correction command (bad links currently force hand-editing the "do not edit" file) | 0, 1, 2 | High |
| Need for `drift list` / `drift status` — read-only inspection of specs, markers, links (only `drift todo` + raw XML exist today) | 0, 1, 2 | High |
| Marker 10-line hash window semantics under-documented (marker line inclusion, blank lines, short functions) — risks false drift on cosmetic edits | 0, 1, 2 | Medium |
| Need for `drift diff` showing what changed on a drifted edge, not just that it drifted | 0, 1, 2 | Medium |
| Need for `--json` output (especially `drift todo --json`) for agent/CI consumption | 0, 1, 2 | Medium |
| `drift skill` is the critical self-describing entry point — all three cold-start subjects discovered and relied on it; this design works | 0, 1, 2 | High (positive signal) |
| Binary self-describing enough for cold-use *happy path*; gaps are in *recovery/inspection* depth, not entry | 0, 1, 2 | High (positive signal) |
| `drift.pin` schema undocumented and "do not edit by hand" leaves users helpless on corruption | 0, 1, 2 | Medium |

## Divergent findings

- **Run 0:** Subject's first action confusion was the `main.` prefix rule (whether spec IDs in `main.pin.xml` need the prefix in-file vs. only at link time) — resolved by skill guide before any wrong action. Also flagged `line="0"` attribute on `<spec>` entries looks like an unpopulated sentinel (erodes trust). Also re-ran `drift todo` after `go build` to confirm the compiled binary isn't treated as a marker host — thoughtful scope check.
- **Run 1:** Flagged binary named `drift` but tool called "driftpin" — naming inconsistency. Also requested per-subcommand `--help` (errors generically on bad args with no usage hint) as its #1 friction. Also noted 6-hex vs alphanumeric spec↔task drift (see Known issues).
- **Run 2:** Flagged `link` vs `reset` distinction initially unclear (which creates vs. collapses a relationship) — wants a one-line note in `drift help`, not just the skill doc. Also flagged duplicate marker IDs across files as a silent-failure risk (last-found wins). Also noted the marker/spec ID namespace independence (`main_entry` → `main.entry`) reduces human readability.

## Prioritized recommendations (consolidated)

1. [High] Add `drift unlink <marker> <module.spec>` (and ideally `drift rm <marker>`) — every bad link currently forces hand-editing the "do not edit by hand" state file. — runs 0, 1, 2
2. [High] Add `drift list` / `drift status` — a read-only human+agent view of specs, markers, links, and sync state; the only inspection path today is grepping `drift.pin` XML. — runs 0, 1, 2
3. [High] Add per-subcommand `--help` and usage-on-no-args/misuse. Today `drift link` with wrong/missing args errors generically; table-stakes CLI ergonomics. — run 1 (broadly applicable; aligns with the cold-use ethos)
4. [High] Reconcile `drift todo` clean-state wording with task/harness expectations ("No changes detected." vs "No drift: N specs…"). Mismatched wording risks false FAILs in automated checks. — run 0
5. [Medium] Clarify the 10-line marker hash window in `drift skill` AND surface a one-liner in `drift help`: marker-line inclusion, blank-line/trailing-whitespace handling, behavior when fewer than 10 lines follow. Cosmetic edits near markers currently trip drift. — runs 0, 1, 2
6. [Medium] Add `drift diff` showing old vs. new content (or changed line range) for a drifted edge. `drift todo` reports *that* something drifted, not *what*. — runs 0, 1, 2
7. [Medium] Add `drift todo --json` (and ideally `--json` across commands) for LLM/CI consumption. — runs 0, 1, 2 (runs 1,2 rated High; run 0 rated Low — settled at Medium as consolidated priority)
8. [Medium] Document the `drift.pin` schema (e.g. a `drift schema` command emitting the shape) so "do not edit by hand" doesn't leave users helpless on corruption. — run 1 (echoed by runs 0,2's `line="0"` confusion)
9. [Medium] Clarify `link` vs `reset` semantics in `drift help` with a one-line note ("`link` creates a new edge; `reset` re-baselines an existing drifted edge"). — run 2
10. [Medium] Warn on duplicate marker IDs across files at `drift todo`/`link` time (silent last-found-wins failure mode). — run 2
11. [Medium] Add `drift lint` / `drift validate` to check `.pin.xml` well-formedness, module-name/spec-id prefix consistency, and orphan markers before link time. — run 1
12. [Low] Drop or correctly populate the `line="0"` attribute on `<spec>` entries in `drift.pin` — currently looks like an unpopulated sentinel. — run 0
13. [Low] Document the `main.` prefix rule up front and redundantly (in `drift help` and the `drift init` template comment): bare IDs in `.pin.xml`, `main.<id>` at link time. — run 0
14. [Low] Add per-command examples to `drift help` (e.g. `drift link add main.add`). — run 0
15. [Low] Standardize naming: binary `drift` vs tool "driftpin" — rename, alias, or self-identify in help. — run 1
16. [Low] Add `drift --version` for reproducibility and bug reports. — run 1
17. [Low] Include the supported file-extension list in `drift help` (currently only `drift skill` lists scanned file types). — run 2
18. [Low] Add guidance on spec granularity to `drift skill` ("one spec per logical unit you'd want to track independently"). — run 2
19. [Low] Make `drift skill`'s role as the comprehensive guide more prominent in `drift help` (e.g. "run `drift skill` for the full tutorial"). — run 2
20. [Low] Consider language-aware marker autoplacement for Go (auto-suggest markers above `func`/`type` declarations matching spec IDs). — run 1

## Next steps

The signal is unusually clean and convergent across all three independent cold-start runs: **driftpin's self-describing binary design works for the happy path** — every subject discovered `drift` → `drift skill` → workflow and finished a clean coverage run on the first attempt with zero tool errors. The friction is concentrated in a narrow, well-defined band: **inspection and recovery depth**, not entry.

The tool authors should, in priority order:

1. **Ship `drift unlink` and `drift list`/`drift status` as the next two commands.** These were the #1 and #2 recommendations in every single run. They close the asymmetry between "easy to create wiring" and "impossible to safely inspect or correct wiring." Both are read/symmetric-only operations with low implementation risk and high payoff for both human and agent users.
2. **Add `drift diff` and `drift todo --json`.** `diff` makes drift actionable (what changed, not just that it changed); `--json` makes the tool reliably consumable by the LLM-agent audience it's explicitly built for. Bundle these with the list/status work as a "state introspection" milestone.
3. **Fix the documentation/wording gaps that risk automated-check failures and false drift:** (a) reconcile `drift todo`'s clean-state string with the "No changes detected." phrasing harnesses expect, (b) surface the 10-line hash window rule and its blank-line fragility in `drift help` (not just `drift skill`), (c) add per-subcommand `--help`/usage-on-misuse, and (d) clarify `link` vs `reset` in `drift help`. These are cheap, high-frequency wins.
4. **Treat `drift skill` as a first-class asset and invest in it.** It is the load-bearing self-describing surface — all three subjects rated it the single most useful discovery. Expand it with the missing items above (prefix rule, hash window, schema, granularity guidance, supported extensions) and keep it authoritative.
5. **Defer the Low-priority polish** (naming, `--version`, marker autoplacement, `line="0"` cleanup) until the inspection/recovery milestone lands — they're real but not blocking cold-use success.

The headline finding for the synthesis record: **a cold-start LLM agent can complete a fully clean, idiomatic driftpin coverage run with zero external docs; the tool's remaining weakness is the absence of read-back and correction affordances, not its onboarding story.**
