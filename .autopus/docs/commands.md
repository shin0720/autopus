# Commands

## makefile (`Makefile`)

### Build

```bash
go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/auto
```

```bash
cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY)
```

### Test

```bash
go test -race -count=1 -tags integration ./...
```

```bash
go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

### Lint / Format

```bash
go vet ./...
```

### Clean / Deploy

```bash
rm -rf bin/ coverage.out
```

### Other

- `test-e2e`: `AUTOPUS_TEST_BINARY=./bin/auto go test -race -count=1 -tags e2e ./e2e/...`
- `test-all`: `go test -race -count=1 -tags 'integration e2e' ./...`
- `update-golden`: `go test -race -count=1 -tags e2e -update ./e2e/...`
- `LDFLAGS`: `make LDFLAGS`
- `generate-templates`: `go run ./cmd/generate-templates`
- `BINARY`: `make BINARY`
- `test-unit`: `go test -race -count=1 ./...`
- `test-integration`: `go test -race -count=1 -tags integration ./...`

## go.mod (`go.mod`)

### Build

```bash
go build ./...
```

### Test

```bash
go test ./...
```

### Lint / Format

```bash
go vet ./...
```

