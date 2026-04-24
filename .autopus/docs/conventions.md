# Code Conventions

## Go

### File Naming

- Detected pattern: **snake_case**
- Examples: `cmd\auto\main.go`, `cmd\generate-templates\main.go`, `content\embed.go`

### Naming

- Packages: `lowercase` (single word preferred)
- Exported: `PascalCase`
- Unexported: `camelCase`

### Error Handling

Detected patterns in this project:

- if err != nil guard
- fmt.Errorf without wrapping
- fmt.Errorf with %w wrapping

### Import Style

- grouped (stdlib / internal / external)

### Project Layout

- `cmd/` — CLI entry points
- `pkg/` — Public reusable libraries
- `internal/` — Private implementation

### Tooling

- **Linter:** golangci-lint
- **Formatter:** gofmt

