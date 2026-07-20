# Drift — Agent Guide

Drift is a spec-drift detection tool for LLM coding agents. Specs describe behavior; markers wrap the code that implements each spec. Specs also cite each other via `<ref>` tags — those citations are tracked too, so editing a spec surfaces drift on every spec that transitively cites it. When any side changes, `drift todo` derives **closures** (per-seed drift sets) so the agent can verify alignment before resolving.

## Spec discipline workflow (MUST follow)

1. **`drift todo`** — see which closures drifted (each with an 8-character hash)
2. **`drift diff <hash>`** — review every node in the closure
3. **For each closure:** decide whether the *code* is wrong (fix the code), the *spec* is wrong (update the spec), or the *citation* is wrong (fix the `<ref>` target)
4. **`drift reset <hash>`** — resolve ONE closure at a time, only after reviewing it

**NEVER batch-reset.** There is no `drift reset --all`. This friction is the point — blind reset defeats the tool.

**`drift todo` exit 1 means unfinished work.** Exit 0 requires both (a) all markers linked and (b) no closures derived. Unlinked markers are actionable drift.

**Exit codes** (apply to every command):
- `0` — clean success.
- `1` — drift present / unlinked markers / todo pending.
- `2` — error (bad args, corrupt state, I/O failure).
- `3` — dry-run preview (no state mutation). Used by `drift reset --dry-run`, `drift link --dry-run`, `drift unlink --dry-run`. LLMs should treat exit 3 as "I haven't changed anything yet" — a successful preview, not a no-op.

## Critical rules

- **Specs and markers are symmetric nodes** in a directed citation graph. Both can drift; drift propagates along the citer chain (cited → citer), transitive to fixpoint. Markers cannot be cited, so drift through a marker stops there — the single retained asymmetry.
- **Spec IDs have exactly one dot** (module separator): `main.bootstrap`, `orch.link`. Marker shortcodes have no dot. Never put a dot in a `<spec id="...">` local ID.
- **Markers wrap the implementation region** with `// D! id=<shortcode> range-start` and `// D! id=<shortcode> range-end`. The scanner hashes the lines between the markers.
- **Refs (`<ref spec="module.localid">label</ref>`) declare spec-spec edges.** The scanner parses them from spec content; they are stored in `state.xml` as baseline edges. Direction records who-cited-whom (used for cycle detection and provenance propagation). Renaming a referenced spec ID does NOT invalidate the referrer's hash — refs are stripped from spec content before hashing.
- **No directed cycles among spec-spec edges.** `$1 → $2 → $1` is rejected by validation. The scanner reports all cycles in one pass.
- **Closures are derived per-seed.** Each drift event has a seed node (the citer-side party of the change). Closure membership = seed + transitive citers (plus, for marker seeds, the linked specs so reviewers can verify the marker still implements them). Closure identity is the first 8 hex chars of SHA1(sorted node IDs + sorted undirected edge keys) — stable across drift-state changes, changes only when membership changes.
- **Closures are strictly disjoint.** Two seeds produce two closures, even if they share non-seed citers. A non-seed citer that cites multiple drifted specs appears in each spec's closure independently.
- **Reset is per-closure, per-seed events.** `drift reset <hash>` syncs the closure's seed events to baseline (NODE_CHANGED → set hash, EDGE_ADDED → add edge, EDGE_REMOVED → remove edge, NODE_REMOVED → remove node). Broken-edge events are no-ops on reset and persist until the user fixes the scan. Citers' state is never modified by reset — only the seed's events sync.
- **Commit `.drift/state.xml` and `.drift/baselines.bin` to git.** They are shared baselines, not local artifacts. Do NOT commit `.drift/user-settings.xml` or `.drift/state.lock` (both gitignored).
- **State file locking is built in.** Concurrent `drift link`/`unlink`/`reset` calls are safe — `internal/fileio` acquires an exclusive advisory lock (flock on Unix, LockFileEx on Windows) on `.drift/state.lock` for the entire CLI invocation via `fileio.Begin`; all state/baseline I/O routes through the resulting `Session`. Safe to batch these in parallel tool calls.

## Build / test / lint

```sh
make build                              # build + drift gate (preferred)
go build -o drift ./cmd/drift           # build only, skip gate
go test -race -count=1 ./...            # full suite with race detector
GOOS=windows go build -o /dev/null ./statestore/   # verify Windows compiles
```

- Module path is `drift`, Go 1.26.
- One external dependency: `golang.org/x/sys` (for cross-platform file locking in `internal/fileio/`). Do not add dependencies without strong justification.
- The race test (`cli/race_test.go`) runs on every `go test ./...` — it is a regression guard for concurrent state mutations, not optional.
- `make build` runs `./drift todo` as a spec-drift gate. The build fails if any drift is detected. On each successful rebuild the prior binary is backed up to `bak/drift-<UTC-timestamp>` (gitignored). Roll back with `cp bak/drift-<ts> drift`.
- State.xml v4 is the provenance-closure format (baseline only, no per-edge resolutions). Pre-v4 files are refused with a clear error directing the user to re-init.

## Repo layout

```
cmd/drift/       # main() entry point
cli/             # CLI dispatch, command structs, output layer (Plain/Color/JSON)
  commands/      # one struct per subcommand (init, todo, link, reset, …)
  output/        # presenters, themes, tokenizer, user settings
core/            # core algorithm (Closure, DriftEvent, DeriveClosures, EvaluateState)
scanner/         # file scanner — specs and refs from *.drift.xml, markers from code
statestore/      # FileStateStore (state.xml v4), BaselineStore (baselines.bin packfile)
orchestrator/    # wires scanner + statestore + core; mutating methods receive a fileio.Session from the caller
eval/            # eval harness (subjects an LLM to a drift fixture, judges result)
internal/        # diff, testutil
business/        # product spec hierarchy (goals → modules → intent → impl)
model.drift.xml  # CONCEPTUAL SPEC — model.provenance (above all impls)
```

## Specs in this repo

The drift codebase is self-hosting on drift. Specs live in `*.drift.xml` files next to the code they describe:

- `model.drift.xml` — `model.provenance`: notation, axioms, algorithm
- `cli/cli.drift.xml` — CLI command contracts
- `core/core.drift.xml` — core algorithm contracts (validate, todo_action, reset_action, scan_coverage, provenance_closure)
- `orchestrator/orchestrator.drift.xml` — orchestrator method contracts
- `cli/output/output.drift.xml` + `output_impl.drift.xml` — output layer
- `statestore/statestore.drift.xml` — state.xml v4 + baseline store
- `business/` — product-level goal hierarchy

Current state: 132 specs, 71 markers, 153 edges. `drift todo` should report clean on a resting tree.

## Editing code that drift tracks

When you change code inside a `// D! id=… range-start … range-end` region:

1. Run `drift todo` — the marker drifts and seeds a closure
2. Run `drift diff <hash>` — see the code delta + deltas of every other node in the closure
3. Read the linked spec and decide: does the spec still describe the new code?
4. If yes → `drift reset <hash>` (baseline syncs)
5. If no → update the spec text in the `*.drift.xml` file, then reset

When you change a spec's wording in a `*.drift.xml` file:

1. Run `drift todo` — the spec drifts and seeds a closure
2. Read the linked marker region in the code (visible via `drift diff <hash>`)
3. Decide: does the code still implement the new spec?
4. If yes → `drift reset <hash>`
5. If no → fix the code, then reset

If the spec you changed is cited by other specs (via `<ref>`), every spec that transitively cites it appears in the same closure (provenance propagation). Every marker linked to those specs also appears in the closure. Reset the closure when verified.

## Adding new specs

1. Add `<spec id="localid">description</spec>` to the relevant `*.drift.xml` module file (local ID must NOT contain a dot)
2. Wrap the implementing code region with `// D! id=<shortcode> range-start` / `range-end`
3. `drift link <shortcode> <module.localid>`
4. `drift todo` — should report clean

## Citing other specs

1. Add `<ref spec="module.localid">label text</ref>` (or self-closing `<ref spec="module.localid" />`) inside a `<spec>` element's content. The label text is preserved in the canonical hash; the `<ref>` tag is stripped.
2. `drift todo` — first time, this surfaces as a closure with an EDGE_ADDED event (a new spec-spec edge). Review and run `drift reset <hash>` to baseline it.
3. Future changes to the *cited* spec will propagate along the citer chain: every transitively-citing spec appears in the closure, plus every marker linked to those specs.

## Closure properties

- **Identity**: 8-character hash of sorted node IDs + sorted undirected edge keys. Stable across drift-state changes; changes only when nodes/edges are added or removed.
- **Ephemeral**: closures exist for the current `drift todo` run; they are NOT stored in state.xml.
- **Per-seed**: each closure has one seed (the citer-side party of the change). Reset syncs only the seed's events. Non-seed citers' state is untouched.
- **Strictly disjoint**: two seeds produce two closures, even when sharing non-seed citers. Resetting one closure never affects another.
- **Broken edges persist**: closures with broken-edge events survive reset (the broken edge event is a no-op on reset). The user must fix the scan (add the missing spec or remove the ref) to clear the broken edge.

## Eval harness

`eval/` runs an LLM ("subject") against a drift fixture workspace, then a judge LLM scores the result. Used to validate that agents can use drift correctly and that drift itself doesn't have UX footguns.

```sh
go run ./eval --battery --repeat 10 --subject <model> --judge <model>
```

Per-prompt overrides via `<name>-subject.md` and `<name>-judge.md` files alongside `<name>.md`. The `--repeat N` flag runs the same prompt N times in parallel for a statistical baseline.

## Output modes

Every command supports three output modes:

- **Plain** (default when piped) — stable text, no ANSI
- **Color** (default in TTY) — themed ANSI + syntax highlighting
- **JSON** (`--json`) — structured output for programmatic consumption

For scripting or LLM consumption, use `--json` or `--no-color`.

## Themes

`drift config theme <name>` sets a per-user preference (stored in `.drift/user-settings.xml`, not committed). 12 built-in themes. Project-level custom theme via `.drift/theme.xml` (committed, full override of all 18 elements).

## Quick reference

| Task | Command |
|---|---|
| What drifted? | `drift todo` |
| Show closure diffs | `drift diff <hash>` |
| Show all closures' diffs | `drift diff --all` |
| Resolve one closure | `drift reset <hash>` |
| List everything | `drift list --verbose` |
| Show one entity | `drift show <marker\|spec>` |
| Full guide | `drift skill` |
| Command reference | `drift help` |
| Structured output | `drift todo --json` |
