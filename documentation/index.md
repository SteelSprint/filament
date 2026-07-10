# Filament documentation

<!-- #F id:j7k3m2nf tool.name tool.location tool.language -->

## Get started

- [Tutorial](tutorial.md) — the mental model, with worked examples of spec drift, site drift, and adding clauses.
- [For LLMs](for-llms.md) — how to read the spec and code together when the docs are not enough.

## Reference

- [Marker format](marker-format.md) — the #F directive, marker ids, clause ids, and comment-prefix agnosticism.
- [State file](state-file.md) — the .filament file structure: [spec], [site], and [state] sections.
- [Public API](public-api.md) — the ten subcommands, their arguments, exit codes, and prose output.

## How to

- [Check a codebase](how-to/check-a-codebase.md) — validate a spec and verify every marker in a project.
- [Add a new clause](how-to/add-a-clause.md) — extend an existing spec with a new clause and keep code in sync.
- [Debug a drift finding](how-to/debug-mismatch.md) — what it means when filament reports a finding, and how to fix it.
