> **Note:** This file predates the Driftpin → Drift rename (breaking change). Historical references to "Driftpin" / "driftpin" / ".driftpin" / "*.pin.xml" / "filament" are retained as-is.

# Observation 0002 — phase1-clean

Date: 2026-07-16
Runs: `phase1-clean-0`, `phase1-clean-1`, `phase1-clean-2`

## Known issues

No runs were tainted by sandbox escapes or harness breakage. All three runs are valid and included in the convergent analysis.

Two methodology caveats (not disqualifying):
- In **all three runs** the subject exhausted its step budget before writing the required `self-debrief.md`. The judge in each case substituted the agent's in-transcript "Maximum Steps Reached" summary. This is a recurring harness/protocol gap: the eval requires an artifact the cold-use scenario structurally prevents the subject from producing. The judge should consider either reserving a mandatory final step for the debrief or making it optional when the step cap is hit.
- Run 0 **passed** all rubric items; runs 1 and 2 **failed** all but the "entry point exists" item. The convergence below is therefore primarily about driftpin's discoverability, not about the task itself — the one passing run demonstrates the tool is functionally sound when the schema is known.

## Convergent findings

| Theme | Runs | Priority |
|---|---|---|
| No real `--help`/`help`/per-subcommand help; bare usage line omits marker syntax, XML schema, `module.spec` convention, workflow | 0, 1, 2 | High |
| `D! id=<id>` marker comment token is undiscoverable; subjects guessed `// drift:marker:`, `// @drift:`, `[DRIFT:id]`, `// DRIFT:...`, etc. | 1, 2 (run 0 found it via `strings` only after ~20 probes) | High |
| `<specs>` wrapper under `<main>` is silently accepted (zero specs extracted) instead of rejected/warned — a natural mistake mirroring `drift.pin`'s own shape | 0, 2 | High |
| Error messages name symptom, not cause/expected shape: `link references unknown spec`, `marker in drift.pin not found on disk` give no schema hint | 0, 1, 2 | High |
| `drift todo` prints identical "No changes detected." for empty-vs-clean projects — vacuously clean, actively misleading | 0, 2 (run 1 also observed) | Medium |
| `module.spec` namespacing convention unclear; subjects tried bare specIds, `main.spec1`, etc. | 0, 1, 2 | Medium |
| `drift init` should scaffold a runnable example (`main.pin.xml` + module file + marker source) instead of an empty `drift.pin` | 1, 2 | High |
| `--help`/`-h` rejected as "unknown command" — basic CLI courtesy missing | 1, 2 | Low |
| Cold users forced to `strings`/`strace` the binary; stdlib noise degrades signal-to-noise | 0, 1, 2 | Medium |
| Required `self-debrief.md` never produced (step budget exhausted) | 0, 1, 2 | Medium (methodology) |

## Divergent findings

- **Run 0 (passing):** Subject eventually succeeded and produced a textbook-correct, drift-clean workspace. Uniquely flagged that `drift todo`'s "No changes detected." is ambiguous between empty and synced (a finding runs 1/2 only half-noted because they never reached sync). Also uniquely recommended a `drift show`/`drift list` introspection command and a `drift init` example scaffold (Low priority there, but converges with runs 1/2's High-priority "runnable example" ask).
- **Run 1 (failing):** Subject hand-edited `drift.pin` to inject a `<marker>` element, creating an internal line-number inconsistency (`line=3` vs `line=4`) — a textbook symptom of bypassing the tool's commands. Uniquely recommended first-class `drift spec add`/`drift marker add` commands to prevent hand-editing, and a `drift sync` scan-and-register step.
- **Run 2 (failing):** Subject ran `strings ./drift | grep -E '(DRIFT|drift)\b'` — one regex character away from the actual `D!\s+id=(\S+)` pattern present in the binary — and used `strace` to observe `main.pin.xml` being opened, but never cross-referenced the on-disk format. Also uniquely recommended accepting `<specs>` wrapper as a synonym (Low) and a `drift markers --format` introspection command.

## Prioritized recommendations (consolidated)

1. **[High] Ship real `drift help`/`--help`/per-subcommand help** with the marker syntax (`// D! id=<id>`), the `*.pin.xml` schema (specs are *direct* children of `<main>`/`<module>`, no `<specs>` wrapper), the `module.specId` reference convention, and a 4-line worked example. Accept `-h`/`--help` as aliases and exit 0. — runs 0, 1, 2
2. **[High] Make `drift init` scaffold a runnable reference**, not an empty `drift.pin`: drop a sample `main.pin.xml` with an `<import>`, a `<module name="…">` file with one `<spec>`, and a `.go` file with one `// D! id=…` marker, pre-linked so `drift todo` passes out of the box. (Consider a separate `drift demo`/`drift example`.) — runs 1, 2 (run 0 flagged as Low)
3. **[High] Reject or warn on silently-ambiguous `*.pin.xml` shapes** (e.g. a `<specs>` wrapper under `<main>`). Either accept both forms, or emit a parse-time warning naming the expected structure. Silent zero-spec extraction is the worst case for cold users. — runs 0, 2
4. **[High] Make error messages prescriptive, not descriptive.** `link references unknown spec: X` → show how many specs were scanned, from which file, and the expected shape (`<spec id="…">` direct child of root/module). `marker in drift.pin not found on disk: X` → show the recognized comment pattern and the line it expected. — runs 0, 1, 2
5. **[Medium] Disambiguate `drift todo` output** between "empty pin store" and "fully synced." Empty should say e.g. `Nothing to check: no specs/markers registered.`; clean should say e.g. `No drift: N specs, M markers, K links in sync.` — runs 0, 2 (run 1 indirectly)
6. **[Medium] Document the `module.specId` namespacing convention** in `drift link`'s own usage with a one-line example (`e.g. drift link dk poker.deck`); consider auto-resolving bare specIds where unambiguous. — runs 0, 1, 2
7. **[Medium] Add introspection commands** (`drift show`/`drift list`/`drift markers --format`/`drift spec`) that print discovered specs/markers/links and the recognized marker pattern grammar, removing the need to `strings` the binary. — runs 0, 2
8. **[Medium] Add first-class `drift spec add`/`drift marker add` (or `drift sync`) commands** that validate as they write, preventing the hand-edited-`drift.pin` inconsistencies seen in run 1. — run 1
9. **[Low] Consider supporting the `<specs>` wrapper as a synonym** for bare `<spec>` under `<main>`/`<module>` — near-zero cost, rescues the most natural mistake. — run 2 (run 0 raised as alternative)
10. **[Low] Strip stdlib noise from the shipped binary** (`-ldflags "-s -w"`) so cold users who do reach for `strings` get usable signal; or document that `strings` is unsupported. — run 0
11. **[Low] Print an onboarding banner on first `drift init`** pointing to the workflow or a bundled `drift guide`. — run 1

## Next steps

The tool is functionally sound (run 0 proves the end-to-end loop works when the schema is known); the failure mode across runs is purely **discoverability for cold, doc-less users**. The single highest-leverage fix is recommendation #2 — make `drift init` emit a runnable, pre-linked 3-file example — because it simultaneously teaches the marker syntax, the spec/module schema, the `module.specId` convention, and the init→mark→link→todo workflow without requiring any prose docs. Pair it with #1 (real `--help` that mirrors the example) and #3 (reject the silent `<specs>`-wrapper trap) to eliminate the three dead-ends that consumed runs 1 and 2. After those, address #4 (prescriptive errors) and #5 (disambiguate `drift todo`) so that *when* a user still guesses wrong, the tool course-corrects them in one step instead of inviting another blind guess. Separately, the eval harness should resolve the `self-debrief.md` gap (reserve a mandatory final step or make the debrief optional at the step cap) so the subject-feedback channel survives cold-use scenarios.
