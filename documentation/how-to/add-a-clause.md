# Add a new clause

<!-- #F id:g0h1i2jk public_api.add parser_rules.no_forward_refs -->

## Add a clause to the spec

Edit your spec XML to add a new clause:

```xml
<clause id="tool.name">The tool is named myproject.</clause>
```

## Generate a marker

```
filament add tool.name
```

This prints a #F marker line with a new 8-character marker id:

```
#F id:a1b2c3d4 tool.name
```

Paste it into your file above the code that implements the clause.

## Register the marker

If you don't have a state file yet:

```
filament init
```

If you already have a state file:

```
filament resolve --site a1b2c3d4
```

This computes the content hash for the marker's surroundings and adds it to the state file.

## Verify

```
filament check
```

If the new clause has a marker and the state file is up to date, the check passes.

## Multiple clauses

Generate a marker for multiple clauses at once:

```
filament add tool.name tool.language tool.binary
```

This prints a single marker line with all three clause ids. One marker can reference multiple clauses — they are tracked independently for drift detection.
