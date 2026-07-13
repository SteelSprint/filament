# Contributing to filament

## Getting started

1. Fork the repository
2. Clone your fork
3. Open in the devcontainer (or install Go 1.22+)
4. Run `go test ./...` to verify everything works

## Development workflow

<!-- #F id:8tf919gk versioning.source -->

filament follows a spec-first workflow:

1. **Spec first** — Edit `filament.spec.xml` to describe the new behavior
2. **Sync** — Run `filament sync` to refresh spec hashes in the state file
3. **Check** — Run `filament check` to see what drifted
4. **Implement** — Write the code
5. **Resolve** — Run `filament resolve --site <id>` for each changed marker
6. **Test** — Run `go test ./...`
7. **Check** — Run `filament check` again to verify

## Running tests

```bash
go test ./...
```

## Self-hosting

filament dogfoods itself. The spec, code, tests, and docs all use #F markers that trace to `filament.spec.xml`. Run `filament check` to verify the project is self-consistent.

<!-- #F id:cdi0ftqy versioning.amendments -->

## Pull requests

- One concern per PR
- Include tests for new functionality
- Run `filament check` before submitting
- Run `go vet ./...` before submitting
