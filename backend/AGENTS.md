# Backend Guidelines

## Structure

`backend/` is the Go module `example.com/uptime/backend`. The server entry
point is `cmd/server/main.go`. Put application-private packages in `internal/`
and reusable packages in `pkg/`; keep package responsibilities narrow.

## Development Commands

Run these commands from `backend/`:

```bash
go run ./cmd/server  # run the service
go test ./...        # run all Go tests
gofmt -w .           # format Go source before committing
```

## Style & Testing

Follow idiomatic Go. Use short, lowercase package names; use `PascalCase` for
exported identifiers and `camelCase` for unexported identifiers. Format all Go
files with `gofmt`; do not hand-format generated output.

Write tests with the standard `testing` package, colocated with the code under
test and named `*_test.go`. Prefer table-driven tests for multiple input cases.
Cover success paths, validation failures, and boundary cases for new behavior.

## Configuration & Safety

Read configuration from documented environment variables rather than embedding
credentials. Validate request input at the handler or service boundary, return
safe errors to clients, and never log secrets.
