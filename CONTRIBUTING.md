# Contributing to gottp

Thank you for your interest in contributing to gottp! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/gottp.git`
3. Create a feature branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Push and open a pull request

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make

### Build & Run

```bash
make build          # Build to bin/gottp
make run            # Build and run
make test           # Run all tests
make lint           # Run linter
make test-race      # Run tests with race detector
make test-cover     # Generate coverage report
```

### Running a Single Test

```bash
go test ./internal/protocol/http/ -run TestClient_GET
```

## Project Structure

- `internal/app/` — Root Bubble Tea model and key handling
- `internal/ui/` — Panels, components, themes, layout
- `internal/protocol/` — HTTP, GraphQL, WebSocket, gRPC clients
- `internal/core/` — Collection, environment, history, state
- `internal/export/` — Export formats and code generation
- `internal/import/` — Import format detection and parsers
- `internal/runner/` — Headless CLI runner
- `internal/scripting/` — JavaScript scripting engine
- `internal/mock/` — Mock HTTP server

## Guidelines

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Run `make lint` before submitting
- Value receivers for `Update()` and `View()`, pointer receivers for mutating methods
- Sub-components return `(Model, Cmd)` from `Update`, not `tea.Model`

### Commit Messages

- Use clear, descriptive commit messages
- Prefix with type: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `ci:`

### Pull Requests

- Keep PRs focused on a single change
- Include tests for new functionality
- Update documentation if relevant
- Ensure all tests pass: `make test`
- Ensure linter passes: `make lint`

### Adding a New Protocol

1. Implement the `protocol.Protocol` interface in `internal/protocol/<name>/`
2. Register it via `registry.Register(client)` in `app.New()`
3. Add a protocol-specific form in `internal/ui/panels/editor/`
4. Update `LoadRequest()` for auto-detection

### Adding a New Auth Type

1. Update `authTypes` slice in `internal/ui/panels/editor/auth_section.go`
2. Add input fields and update `BuildAuth()` / `LoadAuth()` / `View()` / `maxCursor()`

### Adding a New Import Format

1. Add format detection logic in `internal/import/detect.go`
2. Create an importer in `internal/import/<format>.go`
3. Add tests

## Reporting Issues

- Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) for bugs
- Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) for new ideas
- Check existing issues before creating a new one

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
