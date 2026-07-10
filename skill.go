package main

const SkillText = `WHAT IS FILAMENT?

filament tracks whether files stay aligned with their spec. It works on
any plaintext file — source code, documentation, configuration, SQL, HTML,
plain text. Markers (#F) in your files trace to clauses in a spec XML.
filament detects two kinds of drift:

  - Spec drift: the spec changed. Every file location referencing the
    changed clause is flagged so you can verify the content still matches.
  - Site drift: the content near a marker changed. The clause(s) it traces
    to are flagged so you can verify the spec still describes what the
    content does.

Both directions force review. This is the point: spec and files must stay
traceable to each other, and changes in either direction must be
consciously acknowledged, not silently merged.


THE MARKER FORMAT

Markers are placed in any text file, in any comment style:

  Go:       // # F id:example1 tool.name tool.binary
  Python:   # # F id:example1 tool.name tool.binary
  SQL:      -- # F id:example1 tool.name tool.binary
  HTML:     <!-- # F id:example1 tool.name tool.binary -->
  Markdown: <!-- # F id:example1 tool.name tool.binary -->

The format is: #F id:<marker_id> <clause_id> <clause_id> ...

The marker_id is an 8-character identifier (lowercase letters and digits).
The clause_ids are dotted-path identifiers from the spec XML.

The tool matches the #F directive as a substring, regardless of the
comment character that precedes it. This allows markers to work in any
text file with any comment style.


THE STATE FILE (.filament)

The .filament file stores three sections:

  [spec]    — current spec clause hashes
  [site]    — per-marker content hashes
  [state]   — per-marker-clause reviewed spec hashes

The spec section stores the current hash of each clause. The site section
stores the hash of the content near each marker. The state section stores
the spec hash that was in effect when each marker was last reviewed against
each clause it references.

The state file is auto-generated. Do not edit it manually.


DRIFT DETECTION

filament detects two kinds of drift:

  SPEC_DRIFT — the spec clause changed since the marker was last reviewed.
    The code/content at the marker's location may no longer match the
    spec's intent. Review the content, then run:
    filament resolve --spec <marker_id>

  SITE_DRIFT — the content near the marker changed since the marker was
    last stored. The spec clause(s) it traces to may no longer describe
    what the content does. Read the spec clause(s), compare against the
    content, then run:
    filament resolve --site <marker_id>

Both drifts are independent. A marker can have neither, one, or both.
When both are drifted, two separate findings are reported.


COMMANDS

  filament check [file-or-dir]...
    Verify that every #F marker is in sync with the spec. Exits 1 if any
    drift, missing, orphan, or malformed marker is found. Use in CI/CD as
    a failure gate. Default is current directory.

  filament status [file-or-dir]...
    Show every marker and its drift state. Always exits 0.

  filament init [file-or-dir]...
    Create .filament from the current spec and source markers.

  filament add <clause_id> [clause_id]...
    Print a #F marker line with a new marker id. Paste it into your file
    above the content that covers these clauses.

  filament resolve --spec <marker_id> [marker_id]...
    Clear spec drift for the given marker(s). Use after you've reviewed
    the spec changes and confirmed the content still implements them.

  filament resolve --site <marker_id> [marker_id]...
    Clear site drift for the given marker(s). Use after you've reviewed
    the content changes and confirmed they still match the spec.

  filament sync
    Refresh the [spec] section from the current spec XML. Run this after
    editing the spec, before running 'filament check'.

  filament migrate [file-or-dir]...
    Convert old filament:hash comments to #F markers and generate the
    state file. Run this once when upgrading from the old format.

  filament skill
    Print this guide.


THE SPEC-FIRST PHILOSOPHY

The spec is the control plane. The code, tests, and docs are
implementations of it. If the spec is vague, the implementation is forced
to make decisions that silently become de-facto spec — invisible,
untestable, and irreversible.

filament enforces this by requiring every implementation site to be
traceable to a spec clause via a #F marker. When the spec changes,
every site is flagged for review. When a site changes, the spec clauses
it references are flagged for review. Nothing changes silently.


WORKFLOW: UPDATING A SPEC CLAUSE

  1. Edit the spec XML (change clause prose)
  2. filament sync
  3. filament check — reports SPEC_DRIFT for every marker referencing
     the changed clause(s)
  4. Review each flagged marker's content against the new spec wording
  5. filament resolve --spec <marker_id> for each reviewed site
  6. filament check — should pass


WORKFLOW: CONTENT CHANGED NEAR A MARKER

  1. Edit the file (content near a #F marker changes)
  2. filament check — reports SITE_DRIFT for that marker
  3. Read the spec clause(s) the marker traces to
  4. Compare against the new content
  5. filament resolve --site <marker_id>
  6. filament check — should pass


WORKFLOW: ADDING A NEW MARKER

  1.   filament add tool.name tool.binary
     — prints: # F id:example1 tool.name tool.binary
  2. Paste into the file above the relevant content
  3. filament init (if no state file) or
     filament resolve --site example1


WORKFLOW: CI/CD

  - run: filament check
  - Exit 0 = pass, exit 1 = fail
  - Prose output goes to stderr; CI captures exit code


OPTIONS

  --spec=<path>    Path to spec XML (default: ./filament.spec.xml)
  --quiet          Suppress the tooltip preamble
`
