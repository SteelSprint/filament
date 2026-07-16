# Plan

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  CLI (cli.go)                                                 │
│  drift init                                                   │
│  drift todo                                                    │
│  drift list                                                   │
│  drift link <marker> <module.spec>                            │
│  drift unlink <marker> <module.spec>                          │
│  drift reset <marker> <module.spec>                           │
│  drift reset <id>           (orphan cleanup)                  │
│  drift help / drift skill                                     │
├──────────────────────────────────────────────────────────────┤
│  Orchestrator                                                 │
│  load pin → scan → reconcile → build ctx → core              │
│  → save (reset/link/unlink only)                              │
├─────────────────────────┬────────────────────────────────────┤
│  PinStore               │  Scanner                           │
│  read/write drift.pin   │  follow main.pin.xml imports →     │
│  (XML codec)            │  discover specs (module-qualified)  │
│                         │  validate spec/marker ID format     │
│                         │  walk dir tree → discover markers   │
│                         │  hash content → produce ScanResult  │
├─────────────────────────┴────────────────────────────────────┤
│  Core (core.go)                                               │
│  pure, stateless                                            │
│  EvaluateState(ctx) → EvaluatedState                        │
│  - drift detection (including deletion = drift)              │
│  - collapse (prune deleted nodes after resolution)            │
└──────────────────────────────────────────────────────────────┘
```

## Module system

Specs live in `.pin.xml` files. Each file is a module. Files import each other forming a DAG. The scanner starts at `main.pin.xml` (the entry point) and follows imports transitively to discover specs.

### File format

Entry point (`main.pin.xml`) — pure manifest, no specs:
```xml
<main>
  <import path="./core.pin.xml" />
  <import path="./utils.pin.xml" />
</main>
```

Entry point with direct specs (implicit `"main"` module):
```xml
<main>
  <spec id="validate_input">Input MUST be validated.</spec>
</main>
```

Module file:
```xml
<module name="core">
  <import path="./utils.pin.xml" />

  <spec id="validate">
    Validation MUST reject duplicates. Validation
    <ref spec="utils.hash">hashes</ref> each spec's content.
  </spec>
</module>
```

### Rules

- Each `.pin.xml` has exactly one root: `<main>` or `<module name="...">`.
- `<main>` is the entry point. Found by convention in the working directory.
- `<main>` with direct `<spec>` children gets implicit module name `"main"`.
- `<import path="..."/>` resolves relative to the file containing the import.
- Both `<main>` and `<module>` use `<import>`. One keyword, one operation.
- Imports trigger file loading. Transitive: A imports B, B imports C → all loaded.
- Explicit visibility: a module can only `<ref>` specs from modules it directly imports.
- Diamond imports: same file loaded once, deduplicated by absolute path.
- Duplicate module names across the graph: error.
- Cycles: hard error with trace (`main → subA → subB → subA`).
- Spec IDs are per-module. Referenced as `module.specid` (dot-qualified).
- **Spec local IDs must not contain a dot.** Dots are reserved for module qualification. The scanner rejects spec IDs containing dots.
- **Marker shortcodes must not contain a dot.** Dots are reserved for spec ID qualification. The scanner rejects marker IDs containing dots.
- `<ref>` elements are not parsed by the scanner in this phase. They are part of spec content and get hashed as-is. Ref-based drift is a future steel cable.

### ID format invariants

These two invariants enable unambiguous disambiguation in CLI commands:
- **Spec IDs contain exactly one dot**: `module.localId`. The local `id` attribute in `<spec>` must not contain a dot.
- **Marker IDs contain no dots**: bare shortcodes only.

This allows `drift reset <id>` (single-arg orphan cleanup) to determine: dot → spec, no dot → marker.

### Monorepo / multiworkspace

Each workspace (directory with `main.pin.xml`) is an independent drift context:

```
services/auth/
  main.pin.xml       ← entry point
  drift.pin          ← auth's baselines, links, resolutions
  drift.ignore       ← (optional) auth's marker scan exclusions

services/payments/
  main.pin.xml
  drift.pin          ← payments' baselines, links, resolutions
  drift.ignore

shared/
  common.pin.xml     ← no drift.pin, no drift.ignore — just a spec source
```

Running `drift todo` in `services/auth/` uses `services/auth/main.pin.xml` and `services/auth/drift.pin`. Specs can be imported from outside the workspace (`../../shared/common.pin.xml`), but markers are only scanned within the workspace directory tree.

## Decisions

| Decision | Choice |
|---|---|
| drift.pin format | XML (stdlib `encoding/xml`, zero deps) |
| Hash function | SHA1 hex-encoded |
| Missing drift.pin | `drift init` required first |
| CLI output | Match DOCUMENTATION.md exactly |
| Test doubles | Hand-written fakes |
| Testing | Red/green, exhaustive arity, clamped validations |
| Build approach | Walking skeleton / steel cable, end-to-end per iteration |
| Spec files | `*.pin.xml` with `<module>` or `<main>` root |
| Module declaration | `<module name="...">` — one per file |
| Entry point | `<main>` — found by convention in cwd |
| Import syntax | `<import path="..."/>` — relative to importing file |
| Spec IDs | Module-qualified: `module.specid` (e.g., `core.validate`) |
| Spec local ID format | Must not contain a dot (scanner rejects) |
| Spec struct | `Spec.ID` = full qualified string. `Spec.Module` = module name. |
| drift.pin storage | Module not stored separately — derived from qualified ID. |
| Refs | `<ref spec="module.specid">text</ref>` — inline in prose, hashed as content. Not parsed for drift yet. |
| Markers | `D! id=<shortcode>` comment lines in code files. Shortcodes are bare (not module-qualified). |
| Marker ID format | Must not contain a dot (scanner rejects) |
| Marker-to-spec links | `drift link <marker> <module.spec>` — space-separated. |
| drift.ignore | Applies to marker discovery only (code files). Spec discovery is via imports. |
| Marker hashing | Next 10 lines from marker line — **SEE Phase 6: hash model rework** |
| Deleted spec/marker | Treated as drift (not error). Sentinel hash `""` in scan. Surfaces as todo. Pruned after `drift reset`. |
| Orphan (deleted, no links) | Shows as `[deleted]` in `drift list`. Cleaned via `drift reset <id>` (single-arg). |
| Stale entry handling | No hard errors. Reconciler keeps stale entries. Deletion flows through normal drift→reset workflow. |

## XML format for drift.pin

```xml
<drift>
  <specs>
    <spec id="core.validate" hash="afd4321ea69c..." filepath="core.pin.xml" line="0"/>
  </specs>
  <markers>
    <marker id="cval" hash="7dc34f7516f4..." filepath="core.go" line="108"/>
  </markers>
  <links>
    <link specId="core.validate" markerId="cval"/>
  </links>
  <resolutions>
    <resolution specId="core.validate" markerId="cval" currentSpecHash="..." currentMarkerHash="..."/>
  </resolutions>
</drift>
```

Spec IDs are stored as fully qualified strings (`core.validate`). Module is derivable by splitting on the first dot — not stored as a separate attribute.

## Scanner interface

```go
type ScanResult struct {
    Specs   []Spec   // ID is qualified ("core.validate"), Module is "core"
    Markers []Marker // ID is bare shortcode ("cval")
}

type Scanner interface {
    Scan() (ScanResult, error)
}
```

### Spec discovery

1. Look for `main.pin.xml` in the working directory. Error if not found.
2. Parse root element: `<main>` or `<module name="...">`.
3. If `<main>` with direct `<spec>` children: implicit module name `"main"`.
4. Follow `<import path="...">` relative to the importing file.
5. Track visiting stack (by absolute path) for cycle detection.
6. Track loaded files (by absolute path) for dedup.
7. Track module names (by string) for duplicate detection.
8. Each spec: `Module` = module name, `ID` = `Module + "." + localID`.
9. **Validate**: local ID must not contain a dot. Error if it does.
10. Hash: SHA1 of trimmed inner content (including any `<ref>` elements).

### Marker discovery

1. Walk working directory tree for code files (`.go`, `.py`, `.js`, etc.).
2. Apply `drift.ignore` patterns.
3. Find lines matching `D! id=<shortcode>` pattern.
4. **Validate**: shortcode must not contain a dot. Error if it does.
5. SHA1 hash the next 10 lines from the marker line.
6. Record: shortcode (bare ID), filepath, line number, hash.

## Orchestrator reconciliation

On `drift todo` / `drift reset`:
1. Load `PinState` from `drift.pin`
2. Get `ScanResult` from scanner (specs via import graph, markers via dir walk)
3. **Reconcile specs**: for each in ScanResult:
   - In PinState → keep baseline hash, update filepath/line if changed
   - NOT in PinState → new, baseline = current hash (no drift)
   - In PinState but NOT in ScanResult → **keep in reconciled list** (stale entry, baseline preserved). No error. Will be treated as drift via sentinel hash.
4. Same for markers
5. Build `Scan` from ScanResult hash maps. For stale specs/markers (in reconciled list but not in scan), add sentinel hash `""`.
6. Build `CoreAlgorithmContext` with reconciled specs/markers + links/resolution from PinState
7. Run core

## Deletion = drift model

When a spec or marker is deleted from disk but still referenced in drift.pin:

### With links (common case)
1. Reconciler keeps the stale entry with baseline hash preserved
2. `buildScan` adds sentinel hash `""` for the stale entry
3. `computeTodoList`: `scan.SpecHashes[link.SpecID] == ""` → `specChanged = true`, `SpecDeleted = true`
4. `drift todo` shows drift with deletion-specific message: "The spec term has been deleted from disk. If this was intentional, run `drift reset <marker> <spec>` to acknowledge the removal."
5. `drift reset <marker> <spec>` resolves the edge
6. `collapseResolvedNodes`: when scan hash is `""`, **delete the node** from the map (instead of updating baseline). Also remove all its links.
7. `pin.Save`: pruned spec/marker/links are no longer in drift.pin → clean state

### Without links (orphan case)
1. Reconciler keeps the stale entry
2. `buildScan` adds sentinel hash `""`
3. No links → `computeTodoList` generates no todos for it
4. `drift todo` does not mention it (no edge to drift)
5. `drift list` shows it with `[deleted]` tag
6. User runs `drift reset <id>` (single-arg) to clean up
7. Orchestrator removes the entry from pin state + saves

### `drift reset <id>` (single-arg orphan cleanup)

- `drift reset <id>` where `id` contains a dot → look up as spec
- `drift reset <id>` where `id` has no dot → look up as marker
- If found and stale (not on disk, no links) → remove from drift.pin
- If found but still on disk → error: `"%q is still on disk; nothing to remove"`
- If found but has links → error: `"%q still has N links; resolve them first with drift reset <marker> <spec>"`
- If not found → error: `"no spec/marker %q found in drift.pin"`

## CLI commands

| Command | What it does |
|---|---|
| `drift init` | Creates empty drift.pin + starter main.pin.xml |
| `drift todo` | Scans → reconciles → runs core → outputs todos (read-only) |
| `drift list` | Shows all specs, markers, links, sync state (read-only) |
| `drift link <marker> <module.spec>` | Validates + adds link to drift.pin |
| `drift unlink <marker> <module.spec>` | Removes link + resolution from drift.pin |
| `drift reset <marker> <module.spec>` | Resolves a drifted edge, collapses baselines |
| `drift reset <id>` | Removes an orphaned (deleted, no links) spec/marker from drift.pin |
| `drift help` | Prints command reference |
| `drift skill` | Prints comprehensive guide for LLM agents |

## File structure

```
cmd/drift/main.go        # entry point (imports cli)
core/
  core.go                # pure algorithm (drift detection, collapse, deletion handling)
  core_test.go           # exhaustive tests
  core.pin.xml           # specs for validate, todo, reset, collapse, edge, scan, deleted_drift
scanner/
  scanner.go             # import graph + marker discovery + ID format validation
  scanner_test.go        # module format, import graph, cycle detection, ID format
  scanner.pin.xml        # specs for discovery, hashing, duplicates, ID format
pinstore/
  pin_file.go            # XML codec for drift.pin
  pin_file_test.go       # round-trip tests
  pinstore.pin.xml       # specs for load, save, not_found
orchestrator/
  orchestrator.go        # reconcile, init, todo, reset, link, unlink
  orchestrator_test.go   # fakes-based tests
  orchestrator.pin.xml   # specs for init, todo, reset, link, unlink, reconcile
cli/
  cli.go                 # command dispatch + formatters
  cli_test.go            # end-to-end CLI tests
  cli.pin.xml            # specs for dispatch, help, skill, format, reset, link, unlink, list
  help.txt               # embedded help text
  skill.md               # embedded comprehensive guide
  init_main.pin.xml      # embedded starter template
internal/testutil/       # shared test helpers
  testutil.go
  fixtures.go            # markerLine() — excluded from drift scan
main.pin.xml             # entry point importing all project modules
drift.pin                # rebuilt after each phase
drift.ignore             # excludes testutil/fixtures.go, examples/, eval/
eval/                    # LLM-as-judge eval pipeline
  main.go                # parallel battery runner with --runs flag
  pipeline.go            # stage/subject/judge/surface/synthesize
  agents/                # eval-subject.md, eval-judge.md
  prompts/               # task prompts + fixture directories
    drift-detection.md   # prompt: detect drift after code change
    drift-detection/     # fixture: pre-made calculator + drift.pin
    bad-link.md          # prompt: find and fix wrong link
    bad-link/            # fixture: pre-made project with wrong link + drift.pin
    code-refactor.md     # prompt: refactor triggers drift, resolve
    code-refactor/       # fixture: pre-made temp converter + drift.pin
    apply-existing.md    # prompt: add specs to existing code
    library.md           # prompt: greenfield library
    small-cli.md         # prompt: greenfield CLI
observations/            # filed observation records (auto-numbered)
```

## Phase history

### Phase 1-3: ✓ DONE

- Core algorithm, pin store, scanner (import graph), orchestrator, CLI
- Many-to-many topologies, module/import system
- Self-describing binary (`drift help`, `drift skill`, `drift init`)
- `drift unlink`, `drift list`, per-subcommand `--help`
- `drift todo` wording: "No changes detected."
- All tests pass, vet clean, gofmt clean

### Phase 4: ✓ DONE

- Fixture-based eval cases (drift-detection, bad-link, code-refactor)
- Parallel eval pipeline with `--runs` flag
- Observation 0005 filed: 5 High-priority findings

### Phase 5: ✓ DONE

- ID format validation (scanner rejects dots in spec local IDs and marker shortcodes)
- Stale entry handling (deletion = drift, sentinel hash, graceful cleanup)
- `drift reset <id>` orphan cleanup (dot=spec, no dot=marker)
- Promoted `drift skill` at top of help
- Fixture-based eval cases (drift-detection, bad-link, code-refactor)
- Parallel eval pipeline with `--runs` flag
- Observation 0006 filed: full 6-run battery, all subjects+judges+synthesis complete

## Phase 6: Hash model rework + agent ergonomics

**Source**: Observation 0006 — Phase 5 full battery (6 runs)

### Problem statement

Observation 0006 surfaced two fundamental issues:

1. **Drift detection is not robust to marker placement.** The fixed 10-line positional hash window produces two failure modes:
   - **Positional cascade drift** (run 3): Adding an import shifted the hash windows of unrelated markers, producing spurious drift on untouched code.
   - **False "in sync"** (run 5): Markers placed coarsely (tens of lines from implementing logic) meant 3 real edits to betting/draw code produced "in sync" — the tool silently missed the drift it exists to catch.

2. **The tool gives agents little help understanding *what* drifted.** `drift todo` says "something changed" but not what. `drift list` shows IDs and paths but not spec text. Agents must hand-parse `drift.pin` to investigate.

### Recommendations (from observation 0006, consolidated)

#### High priority

| # | Recommendation | Runs | Description |
|---|---|---|---|
| 1 | Rework hash model | 3, 5 | Replace/augment fixed 10-line window with a semantic region (function body / next decl boundary / `fn=` anchoring) |
| 2 | `drift diff <marker\|spec>` | 0, 1, 2, 3, 4 | Show old-vs-new hash and changed content, not just "drifted" |
| 3 | Spec text in `drift list` | 0, 1, 4, 5 | Show spec description inline so consistent-but-wrong links are visible |
| 4 | Fix `line="0"` for specs | 0, 3, 4 | Record real `<spec>` line or drop the `:0` suffix for specs |
| 5 | `drift lint` placement check | 1, 5 | Warn when a marker's hashed window contains no `func`/`class`/`def` line |

#### Medium priority

| # | Recommendation | Runs | Description |
|---|---|---|---|
| 6 | `drift inspect` / `drift show` | 0, 1, 2, 4 | Read-only command to print a marker's hashed region + linked spec text |
| 7 | `--json` output | 0, 1, 2 | Add `--json` to `drift todo`, `drift list`, `drift link` |
| 8 | `drift check` / `drift verify` | 0, 4 | Assert all linked, no orphans, in sync, non-zero exit on failure |
| 9 | `drift reset --all` | 2, 3 | Bulk resolve for multi-edge refactors |
| 10 | `drift reset` confirmation | 2, 3 | Emit `✓ Resolved…` line on success; clarify "collapses baselines" in skill |
| 11 | Document edge cases in `drift skill` | 0, 2, 3, 4 | Deleted markers, removed specs, duplicate IDs, marker-line inclusion, adjacent markers, `line=` auto-update |

#### Low priority

| # | Recommendation | Runs | Description |
|---|---|---|---|
| 12 | Spec-ID vs marker-ID qualification docs | 0, 1, 4 | Clarify convention in skill/help |
| 13 | `--dry-run` on `drift reset` | 3 | Preview which baselines change |
| 14 | Per-subcommand `--help` | 4 | e.g. `drift link --help` |
| 15 | Normalize help-flag handling | 5 | `--help`/`-h`/`help`/no-arg as strict aliases |
| 16 | Detect duplicate `drift link` | 5 | Emit "already linked" message |
| 17 | `drift init` placement hint | 5 | 2-line marker-placement hint on stdout |
| 18 | `drift demo` / `drift init --demo` | 4 | Self-contained demo teaching drift→reset cycle |
| 19 | `drift --version` | 4 | Version/changelog output |
| 20 | Clarify `drift.ignore` examples | 4 | Concrete patterns, negation, comments |
| 21 | `drift auto-link` | 0 | Link by ID similarity for single-module projects |
| 22 | Example workflows in `drift skill` | 3 | Modify → todo → reset transcript |
| 23 | `drift validate` semantic pass | 1 | Token-overlap heuristic between spec and marker code |
| 24 | Relativize `drift.pin` paths | 1 | Store relative paths, document resolution rule |

### Design decisions (to be discussed before implementation)

> The following design questions need to be resolved before implementation begins. Each is tagged with the recommendation it affects.

#### D1: Hash model rework (recommendation #1)

The current model hashes the 10 lines following the marker line. Two failure modes:
- **Window bleed (run 3, corrected):** Markers in the drift-detection fixture are ~3 lines apart (`add_func` at line 9, `sub_func` at line 12, `mul_func` at line 15, `div_func` at line 18). The 10-line window means `sub_func` hashes lines 13–22, which extends into `div_func`'s body. When the subject modifies `div` to handle negative zero, `sub_func`'s and `mul_func`'s hashes change too — spurious drift on untouched code. *(Note: the original synthesis misattributed this to "import shifting the window." That was wrong — the scanner finds markers by regex and hashes content after the marker line, so the marker and its window move together. The real cause is overlapping windows.)*
- **Coarse placement / false "in sync" (run 5):** A marker placed at `game.go:170` (above the deal block) has a 10-line window covering lines 170–180. The actual `bettingRound` function is at line 252. Three real edits to betting logic at lines 215, 230, 254 all fall outside any marker's window. → Drift silently missed.

Options:

- **(A) Hash to end of enclosing block/function.** Scanner detects the end of the enclosing `{}` block (or equivalent indentation block for Python) and hashes from marker line to block end. Pros: captures the real implementation. Cons: language-specific parsing, complexity.
- **(B) Hash to next marker or EOF.** Hash from marker line to the next marker line (or end of file). Pros: language-agnostic, simple. Cons: one marker per function assumed; if two markers are in the same function, the first only gets a partial region.
- **(C) Explicit `fn=` annotation on markers.** Marker syntax: `// D! id=betting fn=bettingRound`. Scanner finds the function by name and hashes its full body. Pros: precise, user-controlled. Cons: requires language-specific function-finding, extra marker syntax.
- **(D) Configurable window size.** Let users set the window (e.g. `// D! id=betting lines=40`). Pros: simple, language-agnostic. Cons: doesn't solve coarse placement, just makes it tunable.
- **(E) Hybrid: default to next-marker-or-EOF, with optional `fn=` override.** Pros: simple default, escape hatch for precision. Cons: two modes to reason about.

**Questions**:
- Should we support multiple languages (Go, Python, JS) or Go-only for now?
- Is block-boundary detection feasible without a full parser/AST?
- Should the hash window be stored in `drift.pin` (so changes to the window are themselves drift)?

#### D2: `drift diff` — what to show (recommendation #2)

`drift diff <marker|spec>` should show what changed. Options for what to display:

- **(A) Old vs. new hash only.** Simple but not very useful.
- **(B) Old vs. new content (the hashed region).** Requires storing the old content or old hash + re-reading current. Since we only store the hash, we'd need to store the old content snapshot somewhere.
- **(C) A unified diff of the hashed region.** Requires storing old content.
- **(D) Just show the current hashed region + the current spec text side by side.** No historical comparison — "here's what the marker covers, here's what the spec says." Lets the agent judge alignment.

**Storage question**: If we want historical comparison (B/C), we need to store the old content in `drift.pin` (or a snapshot file). This bloats the state file. Alternatively, `drift diff` could just show current content (D) and let the agent compare to git.

#### D3: Spec text in `drift list` (recommendation #3)

Currently `drift list` shows: `marker → spec  [synced]  file:line`

Options:
- **(A) Add spec text as a third column.** Could be long; needs truncation.
- **(B) Add a `--verbose` flag that includes spec text.** Default stays compact.
- **(C) Replace `drift list` output entirely with a richer format.** Breaking change.

Should marker code context (first line after marker) also be shown?

#### D4: `line="0"` for specs (recommendation #4)

Specs are stored with `line="0"` and shown as `:0` in `drift list`. Options:
- **(A) Record the real `<spec>` element line number in the XML file.** Requires scanner to track line numbers for specs (it currently doesn't).
- **(B) Drop the `:line` suffix for specs entirely.** Specs don't have meaningful line numbers (their content is what matters, not their position).
- **(C) Show the spec file path without line number.** e.g. `core.pin.xml` instead of `core.pin.xml:0`.

#### D5: `drift lint` placement check (recommendation #5)

Warn when a marker's hashed window contains no `func`/`class`/`def` line. Options:
- **(A) Run automatically on `drift link`.** Warn at link time if placement looks wrong.
- **(B) Separate `drift lint` command.** Run on demand.
- **(C) Both**: warn on link + standalone command for full audit.

What heuristics?
- No `func`/`class`/`def`/`fn`/`function`/`def` keyword within the hashed window.
- Marker is more than N lines from the next declaration.
- Two markers within the same 10-line window (overlapping regions).

#### D6: `drift inspect` / `drift show` (recommendation #6)

A read-only command to inspect a single marker or spec. Options:
- **(A) `drift show <marker|spec>`** — print hashed region + linked spec text + link state.
- **(B) `drift inspect <marker|spec>`** — same.
- **(C) Fold into `drift list --verbose <marker|spec>`** — no new command.

Is this redundant with `drift diff` (D2)? If `drift diff` shows current content (option D), then `inspect` is basically `diff` when there's no drift.

#### D7: `--json` output (recommendation #7)

Add `--json` to `drift todo`, `drift list`, `drift link`. Questions:
- What schema? Need to design JSON structures for each command's output.
- Should `drift todo --json` include exit code semantics (non-zero on drift)?
- Should this be a global flag or per-command?

#### D8: `drift check` / `drift verify` (recommendation #8)

A command for CI / agent gating that asserts everything is healthy. Options:
- **(A) `drift check`** — asserts all linked, no orphans, all in sync. Non-zero exit on failure.
- **(B) `drift verify`** — same.
- **(C) Just make `drift todo` exit non-zero on drift.** No new command.

Is `drift check` different from `drift todo`? If `todo --json` has exit codes, maybe `check` is just `todo` with machine-readable exit semantics.

#### D9: `drift reset --all` (recommendation #9)

Bulk resolve all drifted edges. Safety question:
- **(A) `drift reset --all`** — resolves all drifted edges without prompting.
- **(B) `drift reset --all --dry-run`** — preview first, then confirm.
- **(C) Interactive prompt** — list all drifted edges and ask y/n.

Risk: bulk reset can mask drift that should be investigated. Should `--all` require that the agent has at least run `drift todo` first?

#### D10: `drift reset` confirmation (recommendation #10)

Currently `drift reset` emits nothing on success. Options:
- **(A) Print `✓ Resolved: <marker> → <spec>. Baseline updated.`**
- **(B) Print just `OK` or `Resolved.`**
- **(C) Print the updated drift.pin state summary.**

Also: clarify "collapses baselines" in `drift skill` — what does it actually mean? (Answer: rewrites the baseline hash to the current hash and clears the resolution entry.)

### Implementation order (proposed)

> To be finalized after design discussion.

1. **Hash model rework** (D1) — foundational, affects everything downstream
2. **`drift diff`** (D2) — depends on new hash model for content display
3. **Spec text in `drift list`** (D3) — independent
4. **Fix `line="0"`** (D4) — independent, small
5. **`drift lint`** (D5) — depends on hash model (needs to know what's in the window)
6. **`drift inspect`** (D6) — depends on hash model (shows hashed region)
7. **`--json` output** (D7) — independent
8. **`drift check`** (D8) — independent
9. **`drift reset --all` + confirmation** (D9, D10) — independent
10. **Document edge cases in `drift skill`** (D11) — after all above, captures the new behavior
11. Low-priority items (D12-D24) — batch after high/medium

## Future steel cables

### Steel cable 8: Ref-based drift

Parse `<ref>` elements in spec content. Implement dual-hash model:
- Self hash: hash of spec content excluding resolved refs
- Composite hash: hash including resolved ref content
- Markers link to composite hash
- Drift output distinguishes "you changed this" vs "a dependency changed"

### Steel cable 9: AST

Replace flat prose specs with structured AST nodes. Each node hashable independently. Markers link to specific AST nodes, not whole specs. See design discussion notes.
