# Public API

<!-- #F id:c9d0e1fg public_api.subcommands public_api.check public_api.status public_api.file_walk -->

Filament exposes ten subcommands. Every command prints prose output describing what happened and why. Use `--quiet` to suppress the tooltip preamble.

## check

<!-- #F id:pgbp65gt public_api.check -->

```
filament check [file-or-dir]...
```

Verify that every #F marker is in sync with the spec. Exits 1 if any drift, missing, orphan, or malformed marker is found. Exits 0 if all markers are in sync.

Default file-or-dir is the current directory. File and directory arguments are walked recursively; non-text files are skipped silently.

Checks performed:
- Parser rule violations on the spec XML
- State file presence
- Spec drift per marker-clause pair (current spec hash vs. reviewed spec hash)
- Site drift per marker (current content hash vs. stored content hash)
- Missing clauses (in spec, no marker in any file)
- Orphan markers (marker references clause not in spec)
- Malformed markers (invalid syntax)
- Not-in-state markers (marker not in .filament)

## status

<!-- #F id:t2s5fcx8 public_api.status -->

```
filament status [file-or-dir]...
```

Show every marker and its drift state, including OK markers. Detects every condition that check detects. Prints a coverage summary stating how many clauses have markers and how many do not. Exits 1 if any finding is found, 0 otherwise.

## init

<!-- #F id:si1coe00 public_api.init -->

```
filament init [file-or-dir]...
```

Create `.filament` from the current spec and source markers. If `.filament` already exists, prints an error and exits 1 to prevent destroying review state. Otherwise computes all spec hashes, scans all markers, computes content hashes, and writes the state file with all three sections populated.

## add

<!-- #F id:1hle9m0l public_api.add -->

```
filament add <clause_id> [clause_id]...
```

Print a #F marker line with a new marker id. Exit 0 if all clause ids are defined in the spec, 1 otherwise.

## resolve

<!-- #F id:se20locb public_api.resolve -->

```
filament resolve --spec <marker_id> [marker_id]...
filament resolve --site <marker_id> [marker_id]...
```

Clear drift for specific markers. Requires exactly one of `--spec` or `--site`.

With `--spec`: updates the reviewed spec hash for all clauses referenced by each marker. Clears spec drift for those markers only; other markers referencing the same clauses remain flagged.

With `--site`: recomputes the content hash for each marker from the current source. Clears site drift for those markers only.

## sync

<!-- #F id:et9oll6g public_api.sync -->

```
filament sync
```

Refresh the `[spec]` section from the current spec XML. Does not modify `[site]` or `[state]`. Run this after editing the spec, before running `filament check`.

## migrate

<!-- #F id:d3o49dxl public_api.migrate -->

```
filament migrate [file-or-dir]...
```

Convert old `filament:hash` comments to #F markers and create the state file. Groups adjacent `filament:hash` lines into a single #F marker per group, preserving the comment prefix.

## skill

<!-- #F id:u0i1vjkh public_api.skill -->

```
filament skill
```

Print the full usage guide for LLMs and new users. Self-contained — covers marker format, state file, drift model, commands, and the spec-first philosophy.

## Global options

```
--spec=<path>    Path to spec XML (default: ./filament.spec.xml)
--quiet          Suppress the tooltip preamble
```
