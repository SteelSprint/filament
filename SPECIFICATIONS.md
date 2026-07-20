# Drift Specifications

This document defines the conventions for drift's own specs. It is the
"constitution" for spec authoring: any new spec or spec edit SHOULD
conform to the conventions below.

## Format

Every implementation spec (one describing a code region) MUST have this
shape:

```
<spec id="module.localid">
Overview: one-paragraph summary of what the spec covers.

Defined terms:
  Term — definition anchored to glossary entries where applicable

Requirements:
  R1. <MUST|SHOULD|MAY> ...
  R2. ...

Platform behavior:   (optional — only if behavior differs by OS)
  Unix:    ...
  Windows: ...

Examples:            (optional — only if non-obvious)
  ...

See also: related.spec.ids, glossary.entries.
</spec>
```

Intent / contract / conceptual specs (no marker, cited via `<ref>`)
follow the same shape but typically omit Platform behavior and
Examples. Glossary entries (`glossary.*`) are pure definitions; they
do not follow this shape.

## Capitalized keywords (RFC 2119)

The keywords **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**,
**MAY**, and **REQUIRED** are used with the meanings defined in
RFC 2119:

- **MUST** — absolute requirement. Violations are bugs.
- **MUST NOT** — absolute prohibition.
- **SHOULD** — recommended; deviations must have documented rationale.
- **SHOULD NOT** — not recommended; same.
- **MAY** — optional, at implementer discretion.

Always capitalized when used as a keyword. Lowercase "must" / "should"
in prose is fine when not asserting a requirement.

## Requirement numbering

Each requirement within a spec is numbered `R1`, `R2`, ..., reset to
`R1` at the start of each spec. Numbering is stable: inserting a new
requirement between R3 and R4 renumbers R4 → R5, R5 → R6, etc., and a
corresponding spec-text drift is expected (the spec hash changes).

Cross-references to a specific requirement use the form
`<spec-id>.R<n>`, e.g. `fileio.session.R3` or `pinstore.baselines.R2`.

## Defined terms

When a spec introduces jargon or terms of art, define them in a
`Defined terms:` block at the top. Terms already defined in
`glossary.drift.xml` SHOULD be referenced via
`<ref spec="glossary.X">X</ref>` rather than redefined.

The glossary is the single source of truth for cross-cutting
terminology (Spec, Marker, Edge, Closure, Seed, Citer, etc.).
Spec-local terms (specific to one spec's domain) may be defined inline.

## Marker placement

Markers wrap the code region that implements the spec:

```go
// D! id=<shortcode> range-start
func Foo() { ... }
// D! id=<shortcode> range-end
```

Conventions:

- Marker shortcodes contain NO dot (reserved for spec ID qualification).
- One spec → one marker. Avoid splitting a single spec across multiple
  markers unless the implementation is genuinely split across
  non-contiguous regions.
- A single code file may host multiple markers (one per spec it
  implements).
- Markers may be nested; the scanner blank inner-marker declarations
  before hashing so they don't interfere.
- Markers SHOULD wrap the smallest region that fully implements the
  spec. Wrapping a whole file when only one function is relevant makes
  drift noisy on unrelated changes.

## Intent specs vs. implementation specs

**Implementation spec** — describes a specific code region. Has a
marker. Tracks drift on that code.

**Intent spec** — describes a cross-cutting contract, requirement, or
concept. Has NO marker. Cited via `<ref>` from one or more
implementation specs.

Both kinds live in `*.drift.xml` files. The difference is whether a
marker is linked. Drift's `unlinked` warning applies to markers
without specs, not specs without markers — so intent specs are a
first-class pattern, not a gap.

## Conceptual specs

`thesis.drift.xml`, `model.drift.xml`, `principles.drift.xml`, and
`business/` specs describe the project's mission, conceptual model,
and product hierarchy. These are intentionally unlinked — they sit
above all implementation specs and are cited via `<ref>` from
everywhere. Changes here have far-reaching consequences (the spec
text itself says so).

## Spec ID conventions

- Spec IDs have exactly one dot: `<module>.<localid>`.
- Module name matches the package or directory (e.g. `fileio`,
  `scanner`, `cli`).
- Local ID uses `snake_case` (e.g. `closure_event_ordering`, not
  `ClosureEventOrdering`).
- Local ID MUST NOT contain a dot.

## When to add a new spec

Add a new spec when:

1. You add a new code file that implements distinct behavior
   (currently unmarked code).
2. You add a new behavioral contract that spans multiple code regions
   and isn't captured by any existing spec.
3. You observe a real failure mode (in eval runs, in production, in
   manual testing) that no spec currently prevents.

Do NOT add a new spec when:

1. The behavior is already covered by an existing spec — extend that
   spec instead.
2. The behavior is implementation detail with no observable contract
   (refactor-only changes that don't affect outputs).
3. The behavior is a one-off test case or temporary workaround.

## Reviewing drift

When `drift todo` reports a closure:

1. Run `drift diff <hash>` — see both the spec text delta and the
   code delta side by side.
2. Decide: does the new code still satisfy the spec's R1..Rn
   requirements?
3. If yes → `drift reset <hash>` (syncs baseline).
4. If no → fix the code, OR update the spec text to match the new
   contract, OR fix the citation (if the spec is in the wrong place).

The reviewer's job is to verify R1..Rn, not to rubber-stamp. A green
`drift todo` certifies content alignment only — see
`cli.todo_alignment_vs_correctness`.
