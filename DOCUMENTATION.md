# Drift — Documentation

## What is drift?

Drift detects divergence between specifications and code. **Specs** (`*.drift.xml` files) declare intended behavior in plain English. **Markers** (`// D! id=... range-start` / `range-end` comment pairs in code files) wrap the lines that implement each spec. Both specs and markers are content-hashed and tracked in `.drift/state.xml`. When a hash changes, drift derives **closures** (per-seed drift sets) so a human or LLM agent can review before the baseline is re-established.

Specs also cite each other via `<ref spec="module.localid">label</ref>` tags. Those citations are tracked as edges in a directed graph. When a spec changes, every spec that transitively cites it is flagged for review — that's provenance propagation.

## The citation graph

Two kinds of edges:

- **Link edges** (`marker → spec`) — created by `drift link <marker> <module.spec>`. The marker stores the edge to the spec it implements.
- **Ref edges** (`spec → spec`) — auto-parsed from `<ref>` tags in spec content. The source spec stores the edge to the spec it references.

Both share a single `<edges>` section in `state.xml`. Direction records who-cited-whom (used for cycle detection); drift propagation is along the citer chain (cited → citer), transitive to fixpoint.

## Closures

A **closure** is the unit of review. Each closure is derived from one seed (the node that drifted). Closure membership = seed + transitive citers (plus, for marker seeds, the linked specs). Closure identity is the first 8 hex chars of `SHA1(sorted node IDs + sorted undirected edge keys)` — stable across drift-state changes; changes only when membership changes.

Properties:

- **Per-seed**: each closure has one seed. Reset syncs only the seed's events.
- **Strictly disjoint**: two seeds produce two closures, even if they share non-seed citers.
- **Ephemeral**: closures exist for the current `drift todo` run; not stored in state.xml.
- **Broken edges persist**: closures with broken-edge events survive reset (the broken-edge event is a no-op). The user must fix the scan.

## Commands

| Command | Description |
|---|---|
| `drift init` | Initialize `.drift/` + starter `main.drift.xml`. Idempotent. |
| `drift todo` | Scan specs and markers; derive closures; report drift. Exit 0 when clean, 1 when drift exists, 2 on error. |
| `drift list [--verbose]` | List all specs, markers, edges, and sync state. `--verbose` adds previews. |
| `drift show <marker\|spec>` | Show one entity's content with linked counterparts. |
| `drift diff <hash>` | Show unified diffs for every node in the closure. |
| `drift diff --all` | Show diffs for all closures in one pass. |
| `drift link <marker> <module.spec>` | Create a link edge (marker → spec). |
| `drift unlink <marker> <module.spec>` | Remove a link edge. |
| `drift reset <hash>` | Resolve a closure by syncing its seed events into baseline. |
| `drift config theme <name>` | Set per-user theme preference. |
| `drift config theme` | Show current theme. |
| `drift help` | Show command reference. |
| `drift skill` | Print comprehensive guide for LLM agents. |
| `drift version` | Show version. |

### Global flags

- `--json` — structured JSON output
- `--no-color` — disable ANSI colors
- `--color={auto,always,never}` — control color mode (default: auto)
- `--help, -h` — show command-specific help

### Reset semantics

`drift reset <hash>` re-derives closures, finds the one with matching hash, and applies each event:

| Event | Reset action |
|---|---|
| `NODE_CHANGED` | baseline hash → scan hash |
| `NODE_ADDED` | already in reconciled baseline (reconciler sentinel); reset establishes hash |
| `NODE_REMOVED` | node removed from baseline; edges filtered |
| `EDGE_ADDED` | edge added to baseline |
| `EDGE_REMOVED` | edge removed from baseline |
| `EDGE_BROKEN` | no-op (user must fix the scan) |

Closures containing ONLY broken-edge events are refused.

## Drift events

Every closure contains one or more drift events:

| Event | Trigger |
|---|---|
| `NODE_CHANGED` | baseline node's hash differs from scan hash |
| `NODE_ADDED` | new node in scan, not in baseline (reconciled with sentinel empty hash) |
| `NODE_REMOVED` | baseline node not in scan (Deleted=true) |
| `EDGE_ADDED` | new spec-spec edge in scan (link edges are user-curated and never appear in scan) |
| `EDGE_REMOVED` | spec-spec edge in baseline but not scan |
| `EDGE_BROKEN` | scan edge whose To endpoint doesn't exist |

## Provenance propagation

When a node drifts, drift walks the citer chain (cited → citer) to derive the closure. Worked examples:

- `$2 → $1`, `$1 drifts` → `$2` is in the closure (citer of `$1`).
- `$2 → $1`, `$2 drifts` → `$1` NOT in closure via citer walk. `$1`'s text didn't change.
- `#A → $1`, `$1 drifts` → `#A` is in the closure (citer of `$1`).
- `#A → $1`, `#A drifts` → closure includes `#A` and its outgoing-edge targets (`$1`) so reviewers can verify the marker still implements the spec. Then walks citers of `$1`.

Markers cannot be cited, so drift through a marker stops there — the single retained asymmetry between specs and markers.

## State.xml v4

`.drift/state.xml` is the shared baseline. v4 is the provenance-closure format:

```xml
<drift version="4">
  <specs>
    <spec id="..." hash="..." filepath="..." line="..." />
  </specs>
  <markers>
    <marker id="..." hash="..." filepath="..." line="..." endline="..." />
  </markers>
  <edges>
    <edge from="..." to="..." />
  </edges>
</drift>
```

Pre-v4 files are refused on load with a clear error directing the user to re-init. No migration path.

The previous `<edgeResolutions>` section is gone. Reset = sync baseline to scan for the closure's seed events. There is no partial acknowledgement; closures reset as a unit.

## The `.drift/` directory

- `state.xml` — baseline (v4). Tool-managed. Commit to git.
- `baselines.bin` — gob-encoded packfile of content-addressed snapshots of every spec and marker hash. Commit to git.
- `theme.xml` — optional project-level custom theme. Commit to git.
- `user-settings.xml` — per-user theme preference. Do NOT commit (gitignored).
- `state.lock` — runtime lock file. Do NOT commit (gitignored).

## Output modes

Every command supports three modes:

- **Plain** (default when piped) — stable text, no ANSI.
- **Color** (default in TTY) — themed ANSI + syntax highlighting. 12 built-in themes.
- **JSON** (`--json`) — structured output for programmatic consumption. Never contains ANSI codes.

## Why no bulk reset?

`drift reset <hash>` accepts exactly ONE closure per invocation. There is no `--all`, no glob, no multi-arg form, and none will be added. This friction is the point: a bulk reset would let an LLM blindly mark everything as reviewed without actually reviewing the changes. The intended workflow is:

1. `drift todo` — see which closures drifted.
2. `drift diff --all` — review every closure's changes in one pass.
3. `drift reset <hash>` — resolve ONE closure at a time, after reviewing it.

The one-closure-at-a-time reset is the enforcement point. See `principles.friction`.

## Decision tree (cheat sheet)

Drift is a deterministic signal — it reports that a hash changed but does not judge whether the change is semantically consistent with the spec. Apply this rubric before resetting:

| Event kind                | Question to ask                                                       | If yes                                   | If no                                              |
| ------------------------- | -------------------------------------------------------------------- | ---------------------------------------- | -------------------------------------------------- |
| `NODE_CHANGED` on marker  | Does the code still implement the spec?                              | Reset the closure                        | Fix the code so it does, then reset              |
| `NODE_CHANGED` on spec    | Does the spec still describe what the code does?                      | Reset the closure                        | Fix the spec text (or the code, whichever is wrong), then reset |
| `NODE_ADDED`              | Is this node supposed to be tracked here?                             | Reset the closure                        | Delete or relink the file/entity, then reset      |
| `NODE_REMOVED`            | Was this deletion intentional?                                       | Reset the closure                        | Restore the file, then reset                      |
| `EDGE_ADDED`              | Is the new ref intentional?                                          | Reset the closure                        | Remove the ref from the spec text, then reset     |
| `EDGE_REMOVED`            | Was the removal intentional?                                         | Reset the closure                        | Restore the ref in the spec text, then reset      |
| `EDGE_BROKEN`             | (always — drift can't auto-fix)                                       | Fix the scan: add the missing spec, or remove the ref | — |

## Marker placement during refactors

When you extract helpers or move code across a marker boundary, keep the marker on the public entry-point function that the spec describes. Helpers stay outside the marked range unless they warrant their own spec + marker. Drift is content-addressed — a marker reports changes to the bytes inside its range, so refactors that move code across a marker boundary without changing behavior still produce `NODE_CHANGED` events. There is no "this was just a refactor" annotation; the reviewer's job is to confirm the move didn't change semantics, then reset.

## Dry-run and change summaries

`drift reset`, `drift link`, and `drift unlink` accept `--dry-run`. The dry-run prints the same per-event change summary it would print after applying (node hash changes truncated to 8 chars, edge add/remove), prefixed by a "Preview — no changes written" banner, and exits with code `3` instead of `0` so LLM consumers can distinguish a successful preview from a successful mutation. The post-apply path prints the same change summary without the banner and exits with `0`. JSON mode includes a `preview: true` flag and a structured `summary` field; hashes are truncated to 8 chars in all presenters.

## Build gate

`make build` runs `drift todo` before declaring the build complete. The build fails if any drift is detected. On each successful rebuild the prior binary is backed up to `bak/drift-<UTC-timestamp>` (gitignored). Roll back with `cp bak/drift-<ts> drift`.

## Self-hosting

Drift tracks its own specs and markers. `drift todo` must be clean before any commit — this is a hard gate. The project is its own primary test case: if drift can't track itself correctly, it can't track anything.

## See also

- `drift skill` — comprehensive LLM agent guide.
- `AGENTS.md` — agent workflow summary.
- `PLAN.md` — design history for the closure-driven model.
