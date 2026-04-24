# Testing

## Framework

- **Framework:** go test
- **Command:** `go test -race ./...`
- **Coverage:** Enabled

## Test Locations

- `e2e/`
- `pkg\cost/`
- `pkg\selfupdate/`
- `pkg\sigmap/`
- `pkg\worker\auth/`
- `pkg\worker\reaper/`
- `pkg\adapter\claude/`
- `pkg\adapter/`
- `pkg\docs/`
- `pkg\worker\a2a/`
- `pkg\worker\mcp/`
- `pkg\worker\poll/`
- `pkg\worker\security/`
- `pkg\worker\stream/`
- `internal\cli/`
- `pkg\worker\budget/`
- `pkg\hint/`
- `pkg\template/`
- `pkg\worker\audit/`
- `pkg\worker\compress/`
- `pkg\worker\parallel/`
- `pkg\lore/`
- `pkg\search/`
- `pkg\worker\net/`
- `pkg\adapter\gemini/`
- `pkg\adapter\opencode/`
- `pkg\telemetry/`
- `pkg\worker\workspace/`
- `pkg\lsp/`
- `internal\cli\tui/`
- `pkg\adapter\codex/`
- `pkg\worker\qa/`
- `pkg\worker\scheduler/`
- `pkg\worker\tui/`
- `pkg\browse/`
- `pkg\experiment/`
- `pkg\learn/`
- `pkg\setup/`
- `pkg\worker\adapter/`
- `templates/`
- `pkg\constraint/`
- `pkg\content/`
- `pkg\issue/`
- `pkg\pipeline/`
- `pkg\connect/`
- `pkg\spec/`
- `pkg\version/`
- `pkg\worker\daemon/`
- `pkg\e2e/`
- `pkg\terminal/`
- `pkg\worker/`
- `pkg\worker\routing/`
- `pkg\arch/`
- `pkg\detect/`
- `pkg\orchestra/`
- `pkg\worker\knowledge/`
- `pkg\worker\mcpserver/`
- `pkg\worker\pidlock/`
- `pkg\worker\setup/`
- `pkg\config/`

## Patterns

### Table-Driven Tests

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name string
        input string
        want  string
    }{
        {"basic", "input", "expected"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := DoSomething(tt.input)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Conventions

- Test files: `*_test.go` (same package)
- Use `t.Parallel()` for independent tests
- Race detection: `go test -race ./...`
