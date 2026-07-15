# Plan

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  CLI (main.go)                                       │
│  drift init | drift todo | drift reset <m>:<s>      │
├──────────────────────────────────────────────────────┤
│  Orchestrator                                        │
│  load pin → scan → build ctx+action → core → save   │
├────────────────────────┬─────────────────────────────┤
│  PinStore              │  Scanner                    │
│  read/write drift.pin  │  walk fs, hash specs +      │
│  (XML codec)           │  markers, produce Scan      │
├────────────────────────┴─────────────────────────────┤
│  Core (core.go)  ✓ done                              │
│  pure, stateless                                     │
│  EvaluateState(ctx) → EvaluatedState                 │
└──────────────────────────────────────────────────────┘
```

## Decisions

| Decision | Choice |
|---|---|
| drift.pin format | XML (stdlib `encoding/xml`, zero deps) |
| Hash function | SHA1 hex-encoded |
| Missing drift.pin | `drift init` required first |
| CLI output | Match DOCUMENTATION.md exactly |
| Test doubles | Hand-written fakes |
| File location | Project root |
| Testing | Red/green, exhaustive arity, clamped validations |
| Build approach | Walking skeleton / steel cable, end-to-end per iteration |

## File structure

```
core.go              # done - pure algorithm
core_test.go         # done - exhaustive tests
pin_file.go          # PinState, PinStore interface, filePinStore (XML)
pin_file_test.go     # exhaustive arity, clamped validations
scanner.go           # Scanner interface, fileScanner (fs walk + hash)
scanner_test.go      # exhaustive arity, clamped validations
orchestrator.go      # Orchestrator: Init(), Todo(), Reset()
orchestrator_test.go # exhaustive arity with hand-written fakes
main.go              # CLI entry point
cli_test.go          # E2E steel cable tests
```

## XML format for drift.pin

```xml
<drift>
  <specs>
    <spec id="validate_input" hash="S98YH3T2T32..." filepath="main.pin.xml" line="37"/>
  </specs>
  <markers>
    <marker id="4hy7fh3h" hash="JHIO34YU..." filepath="src/main.go" line="15"/>
  </markers>
  <links>
    <link specId="validate_input" markerId="4hy7fh3h"/>
  </links>
  <resolutions>
    <resolution specId="validate_input" markerId="4hy7fh3h" currentSpecHash="..." currentMarkerHash="..."/>
  </resolutions>
</drift>
```

## Steel cable iterations

Each iteration is end-to-end (test → implement → green → commit).

### Steel cable 1: `drift init` → `drift todo` → "No changes detected."

**Red (tests first):**

`cli_test.go` — E2E test in temp dir:
- `drift init` → assert `drift.pin` exists with valid empty XML
- `drift todo` → assert output is "No changes detected." and exit code 0
- `drift todo` without `drift init` first → assert error

`pin_file_test.go` — exhaustive unit tests:
- Load empty drift.pin → empty PinState, no error
- Save empty PinState → valid XML round-trip
- Load missing drift.pin → error
- Load malformed XML → error
- Arity: 0/1/many specs, 0/1/many markers, 0/1/many links, 0/1/many resolutions
- Round-trip: save → load → assert equal, across all arity shapes

`orchestrator_test.go` — exhaustive unit tests with fakes:
- Todo() with empty PinState + empty Scan → 0 todos, "No changes detected."
- Arity: all topology shapes from core_test.go (0×0, 1×0, 0×1, 1×1, many×many)
- Reset() on nonexistent edge → error
- Clamped validations: mismatched scan/pin specs, unknown IDs, etc.

**Green (implement):**
- `pin_file.go` — `PinState`, `PinStore` interface, `filePinStore` (XML read/write, init)
- `scanner.go` — `Scanner` interface, `fileScanner` (returns empty for now)
- `orchestrator.go` — `Orchestrator` with `Init()`, `Todo()`, `Reset()`
- `main.go` — CLI: parse `init`/`todo`/`reset`, call orchestrator, format output

### Steel cable 2: `drift reset m1:s1` on empty → error

Reset path + CLI arg parsing for `drift reset <marker>:<spec>`.

### Steel cable 3: 1 spec + 1 marker, no drift → clean

Scanner discovers + hashes specs/markers, pin stores real data, links established.

### Steel cable 4: Marker changed → 1 todo surfaced

Drift detection E2E, todo output formatting matching DOCUMENTATION.md.

### Steel cable 5: `drift reset m1:s1` → edge resolved

Reset + collapse + save E2E.

### Steel cable 6+: Many-to-many topologies

Partial resolution (1×2, 2×1, 2×2, 3×3), progressive collapse, matrix state tracking.
