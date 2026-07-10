# For LLMs

<!-- #F id:e4f5g6hi tool.name tool.language tool.design -->
<!-- #F id:ph3dtn9i output.finding_prose -->
<!-- #F id:yqxay09q output.result_prose -->

This document explains how an LLM should read and use filament when working in a codebase that uses it.

## The quick path

Run `filament skill` for the comprehensive usage guide. This covers everything: marker format, state file, drift model, commands, and the spec-first philosophy. It is self-contained and requires no external files.

## The spec is the source of truth

Every codebase that uses filament has a spec XML file (e.g., `filament.spec.xml`). This file defines the clauses that the code must implement. The spec is the control plane — the code, tests, and docs are implementations of it.

When the spec changes, every file location referencing the changed clause is flagged for review. When a file changes, the spec clauses referenced by markers near the change are flagged for review. Nothing changes silently.

## Markers trace code to spec

#F markers in source files create a traceable link from spec to code:

```go
// # F id:example1 tool.name tool.binary
func main() { ... }
```

This means: "the code near this marker implements spec clauses `tool.name` and `tool.binary`." If either clause changes, or if the code changes, filament flags the divergence.

## The state file tracks drift

The `.filament` file stores three things:
- Current spec clause hashes
- Content hashes for each marker's surroundings
- Reviewed spec hashes for each marker-clause pair

When a spec clause changes, the reviewed spec hash for each marker referencing it goes stale → spec drift.
When content near a marker changes, the content hash goes stale → site drift.

Both must be reviewed and cleared independently.

## Workflow for an LLM

1. **Orient**: Run `filament status` to see all markers and their state.
2. **Check**: Run `filament check` to verify everything is in sync. If it exits 0, you're good.
3. **When the spec changes**: Run `filament sync` to refresh spec hashes, then `filament check` to see what drifted. Review each flagged site, then `filament resolve --spec <id>` for each.
4. **When code changes**: Run `filament check` to see what drifted. Read the spec clauses the marker traces to, compare against the new content, then `filament resolve --site <id>`.
5. **Adding new code**: Run `filament add <clause_id>` to get a marker line, paste it into the file, then `filament init` or `filament resolve --site <id>` to register it.

## The "why" matters

Filament enforces that spec and code stay traceable to each other. When the spec changes, every implementation site must be re-checked. When code changes, the spec clauses it claims to implement must be re-read. This prevents silent drift — the gap between what the spec says and what the code does.

An LLM should always review the spec clause before clearing drift. Running `filament resolve` without reading the spec defeats the purpose.
