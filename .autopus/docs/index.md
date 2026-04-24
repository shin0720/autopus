# autopus-adk-main

## Tech Stack

- **Go** 1.26

## Directory Overview

```
cmd/  # CLI entry points
configs/
content/  # Content assets
demo/
docs/  # Documentation
e2e/
internal/  # Private implementation packages
pkg/  # Public reusable libraries
scripts/  # Build and utility scripts
templates/  # Template files
```

## Key Entry Points

- `cmd\auto\main.go` — auto CLI entry point
- `cmd\generate-templates\main.go` — generate-templates CLI entry point

## Documentation

- [Commands](commands.md) — Build, test, lint commands
- [Structure](structure.md) — Directory structure and roles
- [Conventions](conventions.md) — Code conventions with examples
- [Boundaries](boundaries.md) — Constraints (Always / Ask / Never)
- [Architecture](architecture.md) — Architecture decisions and rationale
- [Testing](testing.md) — Test patterns and coverage
