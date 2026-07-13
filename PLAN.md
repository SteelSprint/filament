# Implementation Plan: `<ref>` elements, transitive coverage, and related changes

This plan is derived from FEEDBACK.md and the subsequent design discussion.
It addresses seven issues the agent reported and introduces a new explicit
reference model (`<ref>` elements) with transitive coverage semantics.

---

## Summary of changes

| # | Change | Files affected |
|---|--------|---------------|
| 1 | Replace implicit prose-token references with explicit `<ref>` elements | `parser.go`, `spec.go`, `filament.spec.xml`, all test data |
| 2 | Both terms and clauses can contain `<ref>` | `parser.go` |
| 3 | Transitive coverage for MISSING detection | `comment.go`, `filament.spec.xml` |
| 4 | Keep `no_forward_refs`, `no_cycles`, `no_self_references` — now apply to `<ref>` graph | `parser.go`, `filament.spec.xml` |
| 5 | New parser rules: `ref_content`, `ref_target_undefined`, `term_refs_terms` | `parser.go`, `filament.spec.xml` |
| 6 | Merge status/check logic (status calls `Check()`, exits 1 on findings) | `main.go`, `comment.go`, `filament.spec.xml` |
| 7 | Rule explanations: full text per violation, gated on `--quiet` | `parser.go`, `main.go` |
| 8 | Tooltip: move `--quiet` mention to first line | `comment.go` |
| 9 | Skill text: add SPEC XML SCHEMA section with examples | `skill.go` |
| 10 | Update all test data to use `<ref>` | `testdata/`, `cases/`, `*_test.go` |

---

## Design principle: self-explaining output

Every output filament produces must answer three questions for the LLM (or
human) reading it:

| Question | Example (SPEC_DRIFT) |
|----------|---------------------|
| **WHAT** happened | "The spec clause X changed since this marker was last reviewed" |
| **WHY** it matters | "The text at file:line traces to this clause — verify it still matches" |
| **HOW** to fix | "run 'filament resolve --spec marker'" |

This pattern is already mandated by the spec at `output.finding_prose` and
`output.result_prose`. It must be applied consistently to every new output
in this plan:

- **Parser violations** (`ruleExplanation` function): each explanation states
  what the rule requires, why it exists, and how to fix the violation. The
  violation detail line provides WHAT; the explanation provides WHY and HOW.
  Example: `"term backend refs storage which is a clause, not a term. Terms
  are vocabulary; clauses are requirements. Dependencies flow downward, not
  upward. Remedy: reword the term or move the dependency into a clause."`

- **Coverage summary** (`runStatus`): states how many clauses are covered,
  why uncovered clauses matter (filament can't detect drift for them), and
  how to fix (5-step procedure: check → find → add → paste → init/resolve).

- **Rule explanations** (`ruleExplanation`): each of the 16 rules has a
  one-sentence explanation stating what the rule requires and how to fix the
  violation. Gated on `--quiet`; when quiet, a single line tells the LLM
  to remove `--quiet` for explanations.

- **`term_refs_terms` violation**: explains that terms are vocabulary and
  clauses are requirements, dependencies flow downward, and suggests rewording
  or moving the dependency.

The LLM reading the output should never need external context to understand
what went wrong, why it matters, or what to do next. If the output doesn't
answer all three questions, it's incomplete.

---

## Part 1: `<ref>` element support (parser + spec model)

### 1a. `spec.go` — Element struct and reference model

**`Element` struct (line 40):** Add a `Refs []string` field:

```go
type Element struct {
    Kind          ElementKind
    ID            string
    Label         string
    Text          string
    Refs          []string  // IDs from <ref> elements, in order of appearance
    Parent        *Element
    Kids          []*Element
    InDefinitions bool
}
```

**`ReferencesInOrder` (line 163):** Rewrite to return `e.Refs` filtered by
`defined` and deduplicated, preserving order.

**Remove dead code:** `findRefTokens` (line 142), `isRefChar` (line 126),
`stripTrailingNonID` (line 130), `tokenPos` struct (line 121).

**`ComputeAllHashes` (line 219):** No change. Iterates in document order;
`no_forward_refs` guarantees refs point to earlier elements, so `hashes[r]`
is populated when needed.

**`ComputeHash` (line 197):** No change. Still calls `ReferencesInOrder`,
which now returns `e.Refs`.

### 1b. `parser.go` — Mixed-content parsing for clauses and terms

**New function `readClauseBody(dec, endName) (text string, refs []string, err error)`:**
Reads mixed CharData + `<ref>` elements inside `<clause>` and `<term>`.
Concatenates all text (including ref content) into the returned `Text` string
(inlined — the hash input includes what a human reads). Collects ref IDs into
`Refs`. Rejects any nested element other than `<ref>`.

**New function `readRefContent(dec) (string, error)`:**
Reads the content of a `<ref>` element. Allows only CharData (text). Rejects
any nested `StartElement`. Returns the text content (the referenced ID).

**`parseStartElement` (line 121) changes:**

- `"clause"` case (line 155): Call `readClauseBody` instead of
  `readCharDataOnly`. Populate `Text` and `Refs` fields.
- `"term"` case (line 138): Same.
- `"description"` case (line 130): Keep `readCharDataOnly` — description is
  prose-only, no refs allowed.
- No new `"ref"` case needed — `<ref>` elements are consumed inside
  `readClauseBody`. If `<ref>` appears outside a clause/term, it hits the
  default case and returns `"unexpected element: ref"`.

### 1c. `parser.go` — New parser rules

Add to `validate()` (line 196):

- **`checkRefContent`:** For each element with `Refs`, verify that no ref
  is empty string. A ref to an empty string means `<ref></ref>` — the element
  must contain non-empty text. (Nested elements inside `<ref>` are already
  rejected at parse time by `readRefContent` — no validation function needed
  for that.)
- **`checkRefTargetUndefined`:** For each ref in `e.Refs`, verify
  `defined[ref]` is true. A ref to a nonexistent ID is a violation.
- **`checkTermRefsTerms`:** For each term element, iterate `e.Refs`. Look up
  the ref target's kind via a `map[string]ElementKind` helper (built from
  `spec.All()`). If the target is not `KindTerm`, report violation:
  `"term %s refs %s which is a %s, not a term. Terms are vocabulary;
  clauses are requirements. A term may only reference another term because
  vocabulary must not depend on requirements — dependencies flow downward
  (clauses → terms), not upward. Remedy: reword the term to not reference
  the clause, or move the dependency into a clause instead."`

### 1d. `parser.go` — Update violation messages

Change "cites" to "refs" in violation messages:

- `no_forward_refs` (line 284): `"%s refs %s which is not defined earlier"`
- `no_self_references` (line 299): `"%s refs itself"`

`no_cycles` (line 341) stays as-is — it doesn't use "cites".

---

## Part 2: Transitive coverage

### 2a. `comment.go` — `Check()` function (lines 223-235)

Replace the current MISSING detection (direct-only) with transitive coverage:

1. Build reference graph from `spec.All()`: for each non-section element,
   map `e.ID → e.Refs` (filtered by `defined`).
2. Start with `covered` = set of clause IDs that have markers (the existing
   `referenced` map).
3. BFS queue: for each covered element, follow its refs. If a ref points to
   a defined ID not yet covered, add it to `covered` and enqueue it.
4. A clause is MISSING only if `defined[e.ID] && !covered[e.ID]`.

This follows refs through terms: if clause B has a marker, B refs term T,
and T refs clause A, then A is covered.

---

## Part 3: Status/Check merge

### 3a. `main.go` — `runStatus` (line 164)

Rewrite to call `Check()` internally:

1. Call `Check(spec, lock, paths, windowSize)` to get all findings.
2. Build a set of `(markerID, status)` pairs from findings.
3. Walk files, scan markers. For each marker:
   - If it appears in findings, print the finding prose via `FormatFinding`.
   - If it doesn't appear in findings, print `OK  marker=...  clause=...  file:line`
     (reuse `FormatStatusResult` for OK lines).
4. Print all non-marker findings (MISSING, STATE_FILE_MISSING) as prose.
5. Print coverage summary (always, regardless of gap):
   - Count: direct markers, transitively covered (via ref graph), total clauses, uncovered.
   - `Coverage: N/M clauses covered (D with markers, T transitively). K clauses are uncovered.`
6. If there are uncovered clauses, print explanatory prose (see Part 3b).
7. Exit 1 if any findings, 0 otherwise.

### 3b. Coverage summary prose

**Without `--quiet`:**

```
Coverage: 20/36 clauses covered (14 with markers, 6 transitively). 16 clauses are uncovered.

Uncovered clauses are not traced to any implementation. This means
filament cannot detect drift between the spec's intent and the workspace's
actual behavior for those clauses. To fix this:

  1. Run 'filament check' to see the full list of uncovered clauses.
  2. For each uncovered clause, find the file location that implements it.
  3. Run 'filament add <clause_id>' to generate a marker.
  4. Paste the marker above the relevant content in the file.
  5. Run 'filament init' (if no state file) or 'filament resolve --site <marker_id>'.

A clause is also considered covered if another covered clause or term
references it via a <ref> element. You may not need a marker for every
clause — only for leaf implementations that nothing else depends on.
```

**With `--quiet`:**

```
Coverage: 20/36 clauses covered (14 with markers, 6 transitively). 16 clauses are uncovered.
```

### 3c. `main.go` — `runCheck` (line 108)

No change to logic. Still calls `Check()`, prints findings only, exits 1
if any.

### 3d. `comment.go` — `FormatStatusResult` (line 303)

Keep for OK-line formatting. `status` uses `FormatFinding` for findings and
`FormatStatusResult` for OK markers.

### 3e. `filament.spec.xml` — `public_api.status` (line 108)

Change from:
> "The tool MUST always exit 0, regardless of whether any findings are found."

To:
> "The tool MUST exit 1 if any finding is found, 0 otherwise. The status
> subcommand MUST display every marker and its drift state, including OK
> markers, and MUST print a coverage summary."

---

## Part 4: Rule explanations

### 4a. `parser.go` — New `ruleExplanation(rule string) string`

Maps each `parser_rules.*` name to a sentence stating what the rule requires
and how to fix it. Covers all rules:

| Rule | Explanation |
|------|-------------|
| `description_ignored` | A description element must appear at most once and only at the top level. Remedy: remove duplicate description elements. |
| `path_group` | A child element's id must extend its parent section's id. Remedy: rename the child id or move it under the correct parent. |
| `no_forward_refs` | A `<ref>` must point to an element defined earlier in the spec. Remedy: reorder so the referenced element appears before the referencing one. |
| `no_cycles` | The `<ref>` reference graph must be acyclic. Remedy: break the cycle by removing a `<ref>` that creates a circular dependency. |
| `no_self_references` | An element must not `<ref>` itself. Remedy: remove the self-referencing `<ref>`. |
| `no_text_on_sections` | A section must not contain non-whitespace text. Remedy: move text into a child clause. |
| `no_empty_clauses` | A clause must contain non-whitespace text. Remedy: add content or remove the empty clause. |
| `no_empty_sections` | A section must contain at least one sub-clause or sub-section. Remedy: add a child element or remove the empty section. |
| `no_empty_terms` | A term must contain non-whitespace text. Remedy: add content or remove the empty term. |
| `unique_ids` | Each id must be unique within the spec. Remedy: rename the duplicate element. |
| `valid_id_format` | Each id segment must match [a-z0-9_]+. Remedy: rename using only lowercase letters, digits, and underscores. |
| `single_definitions` | A spec must contain at most one definitions block. Remedy: merge the definitions blocks into one. |
| `terms_in_definitions` | A term must appear inside a definitions block. Remedy: move the term inside a `<definitions>` element. |
| `ref_content` | A `<ref>` element must contain non-empty text content matching a defined id. Remedy: add text content (the referenced id) inside the `<ref>` element. |
| `ref_target_undefined` | A `<ref>` element's text content must match a defined clause, section, or term id. Remedy: fix the id or define the missing element. |
| `term_refs_terms` | A `<ref>` inside a term must reference another term, not a clause or section. Terms are vocabulary; clauses are requirements — dependencies flow downward, not upward. Remedy: reword the term to not reference the clause, or move the dependency into a clause. |

### 4b. `main.go` — Update all 7 violation-printing sites

At each site (`runCheck:117`, `runStatus` (new), `runInit:267`, `runAdd:346`,
`runResolve:409`, `runSync:510`, `runMigrate:560`):

```go
for _, v := range violations {
    fmt.Fprintf(os.Stderr, "PARSER_VIOLATION  %s: %s\n", v.Rule, v.Detail)
    if !quiet {
        fmt.Fprintf(os.Stderr, "  %s\n", ruleExplanation(v.Rule))
    }
}
if quiet && len(violations) > 0 {
    fmt.Fprintln(os.Stderr, "Run without --quiet for full parser rule explanations.")
}
```

---

## Part 5: Tooltip discoverability

### 5a. `comment.go` — `Tooltip` constant (line 329)

Move the `--quiet` mention to the first line:

```
filament tracks whether a workspace's files stay aligned with their spec.
(Suppress this preamble with --quiet.) Specs are the source of truth; #F
markers in workspace files trace to spec clauses. Drift means a clause and
the content referencing it may have diverged — each finding requires review,
not just a command. Read the full guide with 'filament skill'.
```

The spec at `output.tooltip` mandates: alignment tracking, source of truth,
drift meaning, skill recommendation, `--quiet` suppression. This reorganization
includes all mandated content; it just leads with `--quiet` for discoverability.
No spec change needed.

---

## Part 6: Skill text — spec XML schema section

### 6a. `skill.go` — `SkillText` (line 3)

Insert a new `SPEC XML SCHEMA` section between `THE MARKER FORMAT` (ends
line 38) and `THE STATE FILE` (line 41). Content:

**Element reference table** — the seven valid elements:

| Element | Attributes | Content | Parent |
|---------|-----------|---------|--------|
| `<spec>` | `name` (required) | description, definitions, sections, clauses | root |
| `<description>` | none | prose only | `<spec>` |
| `<definitions>` | none | `<term>` elements | `<spec>` |
| `<term>` | `text` (required, = id) | prose + optional `<ref>` | `<definitions>` |
| `<section>` | `id` (required), `label` (required) | sections, clauses | `<spec>` or `<section>` |
| `<clause>` | `id` (required) | prose + optional `<ref>` | `<spec>` or `<section>` |
| `<ref>` | none | text only (the referenced id) | `<clause>` or `<term>` |

**ID rules:**
- Dotted-path identifiers, segments match `[a-z0-9_]+`
- Child element's id must extend parent section's id (e.g., parent `operations` → child `operations.create`)
- Each id must be unique across the entire spec

**Reference rules:**
- `<ref>id</ref>` creates an explicit reference inside a clause or term
- Prose words are NOT references — only `<ref>` elements are
- `<ref>` must contain only text (the referenced id), no nesting
- Forward references, self-references, and cycles are violations
- A ref to an undefined id is a violation
- Transitive coverage: if clause B `<ref>`s clause A and B has a marker, A is covered

**Examples** (three, in increasing complexity):

**Minimal:**
```xml
<spec name="simple">
  <clause id="first">A single leaf clause.</clause>
  <clause id="second">Another leaf clause.</clause>
</spec>
```

**With definitions and refs:**
```xml
<spec name="with_refs">
  <definitions>
    <term text="backend">The storage engine.</term>
  </definitions>
  <clause id="storage">The storage layer uses the <ref>backend</ref>.</clause>
</spec>
```

**Nested sections:**
```xml
<spec name="nested">
  <definitions>
    <term text="example">A term used in clauses.</term>
  </definitions>
  <section id="1" label="First section">
    <clause id="1.1">A leaf clause referencing <ref>example</ref>.</clause>
    <clause id="1.2">A second leaf referencing <ref>1.1</ref>.</clause>
    <section id="1.3" label="Subsection">
      <clause id="1.3.1">A deeply nested leaf.</clause>
    </section>
  </section>
  <section id="2" label="Second section">
    <clause id="2.1">References <ref>1.2</ref> and <ref>1.3.1</ref>.</clause>
  </section>
</spec>
```

---

## Part 7: Spec XML updates (`filament.spec.xml`)

### 7a. Definitions section

**Update `reference` term (line 15):**

Before: "A maximal run of [a-z0-9_.]+ in a clause's or term's prose content
that, after stripping any trailing characters not in [a-z0-9_], exactly matches
the id of a defined clause, section, or term in the same spec."

After: "An inline `<ref>` element inside a clause or term. The text content
of the ref is the id of a defined clause, section, or term in the same spec."

**New `ref` term:** "An inline element inside a clause or term that creates a
reference to another element. The text content of a ref must be a defined id."

**New `covered` term:** "An element is covered if it has a well_formed_marker
(clauses only), or if a covered element references it via a ref. Coverage is
transitive through both clauses and terms. Only clauses can be MISSING; terms
and sections are never reported regardless of coverage."

**Update `missing` term (line 29):**

Before: "A clause in the spec that has no well_formed_marker in any scanned
file. Only clauses are subject to this condition; sections and terms are
structural or vocabulary and are never reported."

After: "A clause in the spec that is not covered. Only clauses are subject
to this condition; sections and terms are structural or vocabulary and are
never reported."

### 7b. Hash format section

**Update `hash.input.references` (line 54):**

Before: "The references are the hash_outputs of every clause, section, or
term cited by this clause or term. Sections have no references because they
have no text content."

After: "The references are the hash_outputs of every clause, section, or
term referenced by `<ref>` elements inside this clause or term. Sections
have no references because they have no text content."

### 7c. Drift model section

**Update `drift.missing` (line 89):**

Before: "A clause in the spec that has no well_formed_marker in any scanned
file MUST be reported as a finding with the status MISSING."

After: "A clause in the spec that is not covered MUST be reported as a finding
with the status MISSING."

**New `drift.transitive_coverage` clause:** "A clause is covered if it has a
well_formed_marker in any scanned file, or if a covered clause or term
references it via a ref. The tool MUST compute the transitive closure of
coverage starting from all clauses that have well_formed_markers."

### 7d. Public API section

**Update `public_api.status` (line 108):**

Before: "The status subcommand MUST take zero or more file or directory paths.
The tool MUST display every marker and its drift state. The tool MUST always
exit 0, regardless of whether any findings are found."

After: "The status subcommand MUST take zero or more file or directory paths.
The tool MUST display every marker and its drift state, including OK markers.
The tool MUST detect every condition that check detects. The tool MUST print
a coverage summary stating how many clauses have markers and how many do not.
The tool MUST exit 1 if any finding is found, 0 otherwise."

**Update `public_api.check` (line 107):** Add "The tool MUST apply transitive
coverage when detecting missing conditions."

### 7e. Parser rules section

**Update `parser_rules.no_forward_refs` (line 145):**

Before: "A clause or term MUST cite only ids that have been defined earlier
in the spec."

After: "A `<ref>` element MUST reference only ids that have been defined
earlier in the spec."

**Update `parser_rules.no_self_references` (line 147):**

Before: "A clause or term MUST NOT cite its own id."

After: "A clause or term MUST NOT contain a `<ref>` to its own id."

**Update `parser_rules.no_cycles` (line 146):**

Before: "The reference graph MUST be acyclic."

After: "The `<ref>` reference graph MUST be acyclic."

**New `parser_rules.ref_content`:** "A `<ref>` element MUST contain
non-empty text content. A `<ref>` element MUST NOT contain nested elements."

**New `parser_rules.ref_target_undefined`:** "A `<ref>` element's text
content MUST match a defined clause, section, or term id in the spec."

**New `parser_rules.term_refs_terms`:** "A `<ref>` element inside a term
MUST reference another term. A `<ref>` inside a term MUST NOT reference
a clause or section. Terms are vocabulary; clauses are requirements.
Dependencies flow downward (clauses → terms), not upward."

### 7f. Convert all clauses in filament.spec.xml to use `<ref>`

Every clause and term in filament.spec.xml that intentionally references
another defined element must use `<ref>`. Requires a full audit of the spec.

Examples:
- `hash.input.references` references `hash_output` → `<ref>hash_output</ref>`
- `marker_format.syntax` references `well_formed_marker`, `marker_id` → `<ref>` elements
- `public_api.check` references `well_formed_marker`, `spec_drift`, `site_drift`, etc.
- etc.

All hashes will change. `.filament` must be regenerated.

---

## Part 8: Test data updates

### 8a. `testdata/golden.spec.xml`

```xml
<spec name="golden">
  <clause id="x">a</clause>
  <clause id="y">b cites <ref>x</ref>.</clause>
  <clause id="z">c cites <ref>y</ref> and <ref>x</ref>.</clause>
</spec>
```

### 8b. `testdata/fixture_new_valid.go`

Content stays the same — ref content is inlined in prose, so the text reads
identically.

### 8c. `testdata/fixture_new_valid.filament`

Recompute hashes. Since content text is identical (inlined ref content) and
references are the same, hashes should be unchanged. Verify during
implementation.

### 8d. Invalid test cases — update to use `<ref>`

| File | Before (implicit) | After (explicit `<ref>`) |
|------|-------------------|--------------------------|
| `f2_forward_reference.spec.xml` | `<clause id="1">...cites 2...</clause>` | `<clause id="1">...refs <ref>2</ref>...</clause>` |
| `f3_cycle_length_2.spec.xml` | `cites 2` / `cites 1` | `refs <ref>2</ref>` / `refs <ref>1</ref>` |
| `f4_self_reference.spec.xml` | `references itself` | `refs <ref>1</ref>` |
| `f8_cycle_length_3.spec.xml` | `cites 2` / `cites 3` / `cites 1` | `refs <ref>2</ref>` / `refs <ref>3</ref>` / `refs <ref>1</ref>` |
| `f11_term_self_reference.spec.xml` | `loopy appears in text` | `<term text="loopy">refs <ref>loopy</ref>.</term>` |
| `f12_term_term_cycle.spec.xml` | `cites beta` / `cites alpha` | `refs <ref>beta</ref>` / `refs <ref>alpha</ref>` |
| `f13_term_clause_cycle.spec.xml` | `cites 1` / `cites external` | `refs <ref>1</ref>` / `refs <ref>external</ref>` |

### 8e. New invalid test cases

| File | Violation |
|------|-----------|
| `f15_ref_nesting.spec.xml` | `<ref>` with a nested element inside → `ref_content` |
| `f16_ref_target_undefined.spec.xml` | `<ref>nonexistent</ref>` → `ref_target_undefined` |
| `f17_term_refs_clause.spec.xml` | `<term text="backend">uses <ref>storage</ref></term>` where `storage` is a clause → `term_refs_terms` |

### 8f. Valid test cases

Update `simple.spec.xml`, `nested.spec.xml`, `alphabetic.spec.xml` to use
`<ref>` where they have intentional references. Currently these use implicit
prose references (e.g., "references example" in nested.spec.xml line 10).
Change to `<ref>example</ref>`.

### 8g. `hash_test.go`

Verify expected hash values. With inlined ref content, the content text is
identical to the prose with words in place. If hashes are unchanged, no test
updates needed. If hashes change (e.g., due to different whitespace handling),
update expected values.

### 8h. `parser_test.go`

Add tests for:
- `TestParserRules_RefContent` — validates `f15_ref_nesting.spec.xml`
- `TestParserRules_RefTargetUndefined` — validates `f16_ref_target_undefined.spec.xml`
- `TestParserRules_TermRefsTerms` — validates `f17_term_refs_clause.spec.xml`

Existing tests should pass after updating the invalid case files.

### 8i. `cli_test.go`

- Update `TestCLI_SkillQuiet` if needed (tooltip text changed).
- If any tests expect `status` to exit 0 with findings, update to expect exit 1.
- Verify self-hosting test still passes (filament.spec.xml hashes will change).

### 8j. `.filament` (filament's own state file)

Regenerate by running `filament init` after all spec changes. All hashes will
change because the spec itself now uses `<ref>` elements.

---

## Part 9: Implementation order

| Step | What | Depends on |
|------|------|-----------|
| 1 | `spec.go`: Add `Refs` field to `Element`, rewrite `ReferencesInOrder`, remove dead code | — |
| 2 | `parser.go`: Add `readClauseBody`, `readRefContent`; update clause/term parsing | Step 1 |
| 3 | `parser.go`: Add new validation rules (`ref_content`, `ref_target_undefined`, `term_refs_terms`) | Step 2 |
| 4 | `parser.go`: Update violation messages ("cites" → "refs") | Step 2 |
| 5 | `comment.go`: Transitive coverage in `Check()` | Step 1 |
| 6 | `parser.go` + `main.go`: Rule explanations (`ruleExplanation` function, print at violation sites) | — |
| 7 | `comment.go`: Tooltip change (move `--quiet` to first line) | — |
| 8 | `main.go`: Status/check merge (`runStatus` calls `Check()`, coverage summary) | Step 5 |
| 9 | `skill.go`: SPEC XML SCHEMA section with examples | — |
| 10 | `filament.spec.xml`: Update all clauses to use `<ref>`, add new terms/clauses | Steps 1-4 |
| 11 | Test data: Update all test cases, add new ones | Steps 1-10 |
| 12 | Regenerate `.filament` | Step 11 |
| 13 | Run `go test ./...` to verify | Step 12 |
| 14 | Run `go vet ./...` and any lint/typecheck commands | Step 13 |

Steps 6, 7, 9 are independent and can be done in parallel with steps 1-5.

---

## Decisions confirmed

| Decision | Choice |
|----------|--------|
| Replace or supplement implicit references? | **Replace** — prose words are no longer references |
| Terms get `<ref>`? | **Yes** — both terms and clauses |
| Transitive coverage? | **Yes** — b refs a + b has marker → a covered |
| Keep `no_forward_refs`? | **Yes** — LLMs work in passages; ordering helps |
| Hash input: inlined or excluded? | **Inlined** — prose reads naturally, hash captures what a human reads |
| `status` exit behavior? | **Exit 1 on findings** — merge logic with `check` |
| Coverage summary prose? | **Always print**, explanatory prose gated on `--quiet` |
| `<ref>` in sections/descriptions? | **No** — parser-level enforcement only (no named rule needed) |
| Undefined ref targets? | **Violation** — `parser_rules.ref_target_undefined` |
| Empty `<ref>` elements? | **Violation** — `parser_rules.ref_content` (non-empty text required) |
| Terms ref clauses/sections? | **No** — `parser_rules.term_refs_terms` (terms are vocabulary, clauses are requirements — dependencies flow downward) |
