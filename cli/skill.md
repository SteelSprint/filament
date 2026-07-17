Drift is a spec-drift detection tool designed for LLM coding agents. It tracks the relationship between specification terms (specs) and the code that implements them (markers). When either side changes, `drift todo` surfaces the drift so the agent can verify alignment.

# Quick Start

```
drift init            # Initialize: creates .drift/ + a starter main.drift.xml
drift help            # Show command reference
drift skill           # Print this guide (pipe to a file or read into context)
```

# Workflow

1. **Initialize**: `drift init` — creates `.drift/` (state directory with `state.xml` and `baselines/`) and `main.drift.xml` (spec entry point template). Edit `main.drift.xml` to add your specs.

2. **Write specs**: Edit `*.drift.xml` files. Each file has a root `<module name="...">` (or `<main>` for the entry point). Specs are `<spec id="...">description</spec>` elements — they must be **direct children** of the root element, not nested inside a `<specs>` wrapper.

3. **Place markers**: Add `// D! id=<markerid> range-start` and `// D! id=<markerid> range-end` comment lines in your code, wrapping the code that implements a spec. The marker IDs are short unique strings you choose.

4. **Link markers to specs**: `drift link <marker> <module.spec>` — connects a marker to a spec. Spec IDs are module-qualified (e.g. `core.validate`).

5. **Check for drift**: `drift todo` — scans specs and markers, compares hashes against baselines, and reports any drift as a todo list. Each item includes a hint: `→ Run 'drift diff <marker> <spec>' to see what changed.`

6. **See what changed**: `drift diff <marker> <module.spec>` — shows a unified diff of both the spec and marker content against their baselines. This is the verify step before resolving.

7. **Resolve drift**: After verifying that code and specs are still aligned, run `drift reset <marker> <module.spec>` to mark the edge as resolved and collapse baselines.

# Spec Files

Specs live in `*.drift.xml` files. The entry point is `main.drift.xml` in the project root.

**main.drift.xml** (entry point — can be pure manifest or have direct specs):
```xml
<main>
  <import path="./core/core.drift.xml"/>
  <spec id="bootstrap">Initialize the project and load all modules</spec>
</main>
```

**Module files** (e.g. `core/core.drift.xml`):
```xml
<module name="core">
  <spec id="validate">Input must be validated before processing</spec>
  <spec id="parse">Parse input tokens into structured data</spec>
</module>
```

Spec IDs are qualified as `<module>.<specId>`. Specs in `main.drift.xml` use the `main.` prefix (e.g. `main.bootstrap`). Imports are relative to the importing file. Diamond imports are deduplicated by absolute path. Cycles are detected and reported with a trace.

**ID format invariants:** The local `id` attribute in a `<spec>` element must NOT contain a dot — dots are reserved for module qualification (e.g. `module.specid`). Marker shortcodes must NOT contain a dot either. This ensures every spec ID has exactly one dot (separating module from local ID) and marker IDs have none, enabling unambiguous disambiguation in CLI commands like `drift reset <id>`.

# Markers

Markers are comment lines in code files that come in pairs: `range-start` and `range-end`. They wrap the code region that implements a spec. The scanner hashes the lines between `range-start` and `range-end` (exclusive of both marker lines).

```go
// D! id=validate_input range-start
func handleRequest() {
    validateInput()
}
// D! id=validate_input range-end
```

**Rules:**
- Every `range-start` with ID X must have a matching `range-end` with ID X in the **same file**, appearing **after** the start.
- Old-style markers without `range-start` or `range-end` are rejected with an error.
- The scanner reports all unpaired markers at once (not fail-on-first).
- Nested and overlapping ranges are allowed.
- Before hashing, other marker declaration lines within a range are blanked (the `D!` declaration is stripped, leaving only the comment prefix like `// `). This makes markers invisible to each other's hashes.

Supported file extensions: `.go`, `.py`, `.js`, `.ts`, `.jsx`, `.tsx`, `.java`, `.c`, `.cpp`, `.h`, `.hpp`, `.rs`, `.rb`, `.php`, `.swift`, `.kt`, `.cs`, `.scala`, `.sh`, `.bash`, `.lua`, `.dart`, `.vue`, `.svelte`.

The marker pattern is a regex: `D!\s+id=(\S+)(?:\s+(range-start|range-end))?`. It can appear in any comment style (`//`, `#`, `--`, `/* */`, etc.).

# CLI Commands

| Command | Description |
|---|---|
| `drift init` | Create `.drift/` directory (state.xml + baselines/) and `main.drift.xml` template. |
| `drift todo` | Scan specs and markers, report drift. Exit 0 if clean, 1 if drift, 2 on error. Each item includes a hint to run `drift diff`. |
| `drift list [--verbose]` | Show all specs, markers, links, and sync state. `--verbose` adds spec text and marker content preview. Read-only. |
| `drift show <marker\|spec>` | Show current content of a spec or marker with filepath and line ranges. Linked specs/markers are also displayed. Read-only. |
| `drift diff <marker\|spec>` | Show unified diffs of spec and marker content vs baselines for all linked edges. Read-only. |
| `drift diff <marker> <module.spec>` | Show unified diff for a specific edge (spec side + marker side). Read-only. |
| `drift diff --all` | Show unified diffs for ALL drifted edges at once — every entry in `drift todo`. Forces review of every broken edge before resolving any of them. Read-only. |
| `drift link <marker> <module.spec>` | Connect a marker to a spec. Both must exist on disk. Writes baseline snapshots. |
| `drift unlink <marker> <module.spec>` | Remove a link between a marker and a spec. Also clears resolution state for that edge. |
| `drift reset <marker> <module.spec>` | Mark a drifted edge as resolved. Prints confirmation. Collapses baselines when all edges for a node are resolved. |
| `drift reset <id>` | Remove an orphaned (deleted, no links) spec/marker from state.xml. |
| `drift help` | Show command reference with examples. |
| `drift skill` | Print this guide (for LLM agents learning the tool). |

# How Drift Detection Works

`drift` SHA1-hashes spec content (the text inside `<spec>` elements) and marker content (the lines between `range-start` and `range-end`, with other marker declarations blanked). These hashes are stored as baselines in `.drift/state.xml`. On each `drift todo`, current hashes are compared against baselines:

- **No drift**: All hashes match → "No changes detected. N specs, M markers, K links in sync."
- **Marker changed**: The code near a marker was modified. Check if it still matches the spec.
- **Spec changed**: The spec text was modified. Check if the code still implements it.
- **Both changed**: Both sides changed. Verify alignment on both sides.

Drift is per-edge (one marker ↔ one spec). If 1 spec is linked to 3 markers and the spec changes, that's 3 todo items. `drift reset <marker> <module.spec>` resolves one edge. When all edges for a node are resolved, the baseline collapses to the current hash.

# Diffs

`drift diff` shows what changed between the baseline and current content. This is the verify step: instead of re-reading whole files and guessing against a hash, you see a unified diff.

- `drift diff <marker> <module.spec>` — shows both the spec and marker diffs for one edge.
- `drift diff <marker|spec>` — auto-expands to all linked edges.
- `drift diff --all` — shows the diff for EVERY drifted edge (every entry in `drift todo`) in one pass. Use this to review all broken edges before resolving them.

Each side shows:
- Entity ID, filepath, and line range (markers only)
- Status: `in sync`, `no baseline snapshot`, or `deleted from disk`
- If changed: a unified diff with `--- baseline` / `+++ current` headers

When there's no baseline snapshot (e.g. pre-migration or content-addressed miss), the diff shows "Status: no baseline snapshot (hash X)" — informational, not an error.

# Why no bulk reset?

**Drift intentionally provides NO command to reset multiple edges at once.** There is no `drift reset --all`, no glob, no multi-arg form. Every drifted edge must be individually reviewed and individually resolved with `drift reset <marker> <module.spec>`.

This friction is deliberate and is the whole point of the tool. A bulk reset would let you (or an LLM agent) blindly mark everything as reviewed without actually reading the changes — which would make drift useless as a spec-code sync tool.

The intended workflow when drift exists:

1. `drift todo` — see what broke (which edges drifted).
2. `drift diff --all` — review every broken edge's changes in one pass.
3. `drift reset <marker> <module.spec>` — resolve ONE edge at a time, after reviewing it.

If you find yourself wanting a bulk reset, that is a signal that you are not actually reviewing the drift — which is exactly the failure mode drift is designed to prevent.

# .drift/ directory

`.drift/` is the state directory at the project root. It contains:

- `state.xml` — XML state file storing baseline hashes, links, and resolution state. Tool-managed — do not edit by hand. Commit to git.
- `baselines/` — content-addressed baseline files. Each file is named by its SHA1 hash (`sha1(content) == filename`). Written on `link` and `reset`. Dedup'd automatically. Orphaned files (from collapsed baselines) are harmless. Commit to git.

# drift.ignore

A `.gitignore`-style file at the project root. Patterns exclude files/directories from marker scanning. Directory patterns end with `/`. Comments start with `#`.

# Edge Cases

- **Unpaired markers:** A `range-start` without a matching `range-end` (or vice versa) in the same file causes a scanner error. All unpaired markers are reported at once.
- **Old-style markers:** Markers without `range-start` or `range-end` suffix are rejected. This enforces the range model.
- **Nested ranges:** An outer range can contain inner ranges. Inner marker declarations are blanked from the outer range's hash, so changing an inner marker's ID does not affect the outer marker's hash.
- **Overlapping ranges:** Ranges that partially overlap (neither fully contains the other) are allowed.
- **Empty ranges:** A `range-start` immediately followed by `range-end` (no content between them) hashes an empty string. This is allowed but not useful.
- **Deleted specs:** When a spec is removed from a `.drift.xml` file but still in `state.xml`, it is treated as drift (not an error). `drift todo` shows a deletion-specific message. Resolve with `drift reset <marker> <spec>`. After resolution, the deleted spec and its links are pruned from `state.xml`.
- **Deleted markers:** Same deletion-as-drift model as specs. The marker's hash becomes empty, triggering drift on all linked edges.
- **Orphaned entries (deleted, no links):** Shown with `[deleted]` tag in `drift list`. Cleaned via `drift reset <id>` (single-arg: dot = spec, no dot = marker).
- **`drift reset` semantics:** Rewrites the baseline hash to the current hash and clears the resolution entry. Prints "Resolved: MARKER → SPEC. Baseline updated." on success.
- **`drift todo` exit codes:** 0 = clean, 1 = drift exists, 2 = error. Use in CI: `drift todo && echo "clean"`.
