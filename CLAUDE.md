# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is gottp?

A Postman/Insomnia-like TUI API client built in Go with Bubble Tea. Three-panel layout (sidebar, editor, response) with vim-style modal editing, collections stored as YAML, and 8+ theme support. Supports HTTP, GraphQL (including subscriptions), WebSocket, and gRPC (including streaming) protocols with environment variable interpolation, 7 auth methods (basic/bearer/apikey/oauth2/awsv4/digest/none), request history (SQLite), cURL/HAR/Postman/Insomnia/OpenAPI import/export, response diffing (line + word-level), pre/post-request JavaScript scripting, code generation (8 languages), request chaining/workflows, mock server, and a headless CLI runner.

## Build & Test Commands

```bash
make build            # Build to bin/gottp (with version ldflags)
make run              # Build and run
make test             # go test ./...
make test-race        # go test -race ./...
make test-cover       # Coverage report with atomic mode
make lint             # golangci-lint run
make bench            # Benchmarks for HTTP client, scripting, diff, collection loader
make fuzz             # Fuzz all import parsers (30s each)
make vulncheck        # govulncheck ./...
make install          # go install to $GOPATH/bin
make release-dry-run  # GoReleaser snapshot (test release build)

# Single test / package
go test ./internal/protocol/http/ -run TestClient_GET
go test ./internal/runner/ -v
go test ./internal/auth/digest/
go test ./internal/mock/
```

Launch TUI: `./bin/gottp --collection path/to/file.gottp.yaml`

Headless CLI: `./bin/gottp run collection.gottp.yaml --env Production --output json`

Mock server: `./bin/gottp mock collection.gottp.yaml --port 8080`

CLI subcommands: `run`, `init`, `validate`, `fmt`, `import`, `export`, `mock`, `completion`, `version`, `help`

Environment files: place `environments.yaml` next to the collection file. The first environment is auto-selected on startup.

## Architecture

**Bubble Tea MVU pattern**: All UI components implement `Update(msg) -> (Model, Cmd)` and `View() -> string`. The root model is `internal/app/app.go` which orchestrates panels, overlays, and message routing.

### Critical: Message Types Package

`internal/ui/msgs/msgs.go` is a **shared message types package** created to avoid import cycles between `app` and UI components. All `tea.Msg` types and enums (`AppMode`, `PanelFocus`) live here. Both `app` and all UI packages import `msgs` — never import `app` from UI packages. When adding a new message type, add it to `msgs.go`, not to `app` or UI packages.

### Key Packages

| Package | Role |
|---------|------|
| `internal/app/` | Root model, split into focused sub-modules (see below) |
| `internal/ui/msgs/` | Shared message types (breaks import cycles) |
| `internal/ui/panels/{sidebar,editor,response}/` | Three main panels |
| `internal/ui/components/` | Reusable: KVTable, TabBar, StatusBar, CommandPalette, Help, Modal, Toast, JumpOverlay |
| `internal/ui/theme/` | Theme catalog, lipgloss styles, custom YAML theme loader |
| `internal/ui/layout/` | Responsive three-panel layout calculator |
| `internal/protocol/` | Protocol interface, Registry, HTTP/GraphQL/WebSocket/gRPC clients |
| `internal/core/collection/` | YAML collection model, loader, saver |
| `internal/core/environment/` | Environment variables, `{{var}}` interpolation via `Resolve()`, AES-256-GCM encryption |
| `internal/core/{history,state,cookies,tls}/` | SQLite history, central state, cookie jar, mTLS config |
| `internal/export/` | curl/HAR/Postman/Insomnia export + `codegen/` (8 languages) |
| `internal/import/` | Format auto-detection + curl/Postman/Insomnia/OpenAPI/HAR importers |
| `internal/runner/` | Headless CLI runner, perf baselines, workflow execution |
| `internal/auth/{oauth2,awsv4,digest}/` | Auth implementations |
| `internal/diff/` | Myers diff (line + word-level via `DiffLinesWithWords()`) |
| `internal/scripting/` | JavaScript scripting via goja engine |
| `internal/mock/` | Mock HTTP server with CORS, latency/error simulation |
| `internal/templates/` | Pre-built request templates |
| `internal/config/` | App config from `~/.config/gottp/config.yaml` |

### app.go Sub-Module Structure

The `internal/app/` package is split into focused files:

| File | Contents |
|------|----------|
| `app.go` | `App` struct, `New()`, `Init()`, `Update()`, `View()`, `resizePanels()` |
| `app_keys.go` | `handleGlobalKey()`, `handlePanelKey()`, `updateEditorInsert()`, `cycleFocus()`, `updateFocus()` |
| `app_request.go` | `sendRequest()`, `handleRequestSent()`, `initiateOAuth2()`, introspection/reflection handlers |
| `app_overlays.go` | `handleSwitchTheme()`, `handleImportFile()`, `handleSetBaseline()`, `openExternalEditor()` |
| `app_tabs.go` | `syncTabs()`, `loadActiveRequest()`, `loadHistory()`, `handleRequestSelected()` |
| `app_save.go` | `saveCollection()`, `copyAsCurl()`, `importCurl()`, `handleGenerateCode()`, `handleInsertTemplate()` |
| `keymap.go` | `KeyMap` struct and `DefaultKeyMap()` |

### Message Routing in app.go

`Update()` priority: overlays (command palette > help > modal > jump) → editor insert mode → global keys (Ctrl+Enter send, Ctrl+K palette, etc.) → panel-specific keys. The `handleGlobalKey()` and `handlePanelKey()` methods in `app_keys.go` handle this dispatch.

### Protocol Interface

```go
type Protocol interface {
    Name() string
    Execute(ctx context.Context, req *Request) (*Response, error)
    Validate(req *Request) error
}
```

Implemented: HTTP, GraphQL, WebSocket, gRPC. The `Registry` dispatches requests by `req.Protocol` field. Register new protocols via `registry.Register(client)` in `app.New()`.

### CLI Runner (`gottp run`)

`cmd/gottp/main.go` routes to TUI mode (default) or subcommands. The runner in `internal/runner/` loads the collection, resolves env vars, executes requests with scripting, and outputs results. Supports workflow execution (`--workflow`), performance baselines (`--perf-save`/`--perf-baseline`). Exit codes: 0=pass, 1=test assertion failure, 2=request error. Output formats: text, json, junit.

### UI Modes

`ModeNormal` (navigate with j/k, Tab between panels) → `ModeInsert` (typing in text fields, Esc to return) → `ModeJump` (f key, type label to jump) → `ModeSearch` (/ in response body). `i` enters insert mode, `Esc` exits. Components track their own editing state via `Editing() bool`.

### Environment Variable Resolution

`sendRequest()` calls `environment.Resolve()` on URL, header values, param values, body, and auth fields before executing. Resolution priority: env vars > collection vars > OS env.

### Multi-Protocol Editor

`editor.Model` wraps four protocol-specific forms (`HTTPForm`, `GraphQLForm`, `WebSocketForm`, `GRPCForm`) and a `ProtocolSelector` widget. `Ctrl+P` cycles protocols. All form access goes through delegation methods on `editor.Model`:
- `BuildRequest()`, `GetParams()`, `GetHeaders()`, `GetBodyContent()`, `SetBody()`, `BuildAuth()`, `FocusURL()`
- `LoadRequest()` auto-detects protocol from collection request fields
- **Never use `editor.Form()` directly** — use the delegation methods instead

### Auth Section

`AuthSection` in `editor/auth_section.go` supports none/basic/bearer/apikey/oauth2/awsv4/digest. `BuildAuth()` returns `*protocol.AuthConfig`, `LoadAuth()` populates from `*collection.Auth`. When adding a new auth type: update `authTypes` slice, add input fields, update `BuildAuth()`/`LoadAuth()`/`View()`/`maxCursor()`.

### Scripting Engine

`internal/scripting/` provides a JavaScript (ES5.1+) engine via `goja`. Pre-scripts can mutate the request; post-scripts have read-only access to the response. The `gottp` global object provides: `setEnvVar()`, `getEnvVar()`, `log()`, `test(name, fn)`, `assert()`, `base64encode/decode()`, `sha256()`, `md5()`, `hmacSha256()`, `uuid()`, `timestamp()`, `timestampMs()`, `randomInt()`, `sleep()`, `readFile()`. Each execution uses a fresh runtime with configurable timeout (default 5s).

### HTTP Client Infrastructure

The HTTP client (`internal/protocol/http/client.go`) supports:
- **Proxy**: `SetProxy(proxyURL, noProxy string)` — HTTP/HTTPS/SOCKS5 proxy with per-request override
- **Cookie Jar**: `SetCookieJar(jar)` — auto-captures `Set-Cookie` and sends cookies
- **mTLS**: `SetTLSConfig(cfg)` — client cert+key, CA bundles, skip-verify
- **Timing**: `net/http/httptrace` for DNS/TCP/TLS/TTFB/Transfer breakdown
- **Digest Auth**: Automatically retries on 401 with `WWW-Authenticate: Digest` header

### CI/CD

GitHub Actions in `.github/workflows/`:
- `ci.yml` — test, lint, build on push/PR (ubuntu + macos, Go 1.25.x)
- `release.yml` — GoReleaser on tag push (v*) for cross-platform binaries

## Conventions

- Value receivers for `Update()` and `View()`, pointer receivers for mutating helpers (`startEditing`, `commitEdit`, `SetSize`, `SetPairs`)
- Sub-components return `(Model, Cmd)` from Update, not `tea.Model` — they use concrete types
- Responsive layout breakpoints: <60 cols = single panel, <100 cols = two panel, >=100 = three panel
- Collection files use `.gottp.yaml` extension; environment files use `environments.yaml`
- HTTP method colors: GET=green, POST=yellow, PUT=blue, PATCH=peach, DELETE=red (Catppuccin palette)
- Sub-models needing theme colors take both `theme.Theme` and `theme.Styles` in constructors; store `th` for colors and `styles` for pre-computed lipgloss styles
- Config fields use `yaml:"...,omitempty"` tags and zero-value defaults
- Import format detection lives in `internal/import/detect.go` — add new format checks there when adding importers
- `saveCollection()` syncs form state back to `store.ActiveRequest()` before writing YAML
- `authConfigToCollection()` in `app_save.go` maps `protocol.AuthConfig` → `collection.Auth` for persistence

## Known Gotchas

- KVTable View() must build the cursor prefix (`"> "` / `"  "`) separately before joining with styled content. Slicing into styled strings breaks ANSI escape sequences.
- Response body search uses plain text (not syntax-highlighted) for match highlighting to avoid ANSI escape interference.
- Command palette has dynamic mode: `OpenEnvPicker()`/`OpenThemePicker()` replace commands temporarily; `ResetCommands()` restores defaults on close.
- The HTTP client creates a new `http.Transport` per-request when proxy/TLS config is present to support per-request proxy overrides. Without custom config, it reuses the default transport.
- The `internal/core/tls` package is imported as `gotls` in config to avoid conflict with `crypto/tls`.
- Digest auth performs a transparent 401 retry — the first request intentionally gets a 401, then the response's `WWW-Authenticate` challenge is used to compute the `Authorization` header for the retry.
- The `diff.DiffLinesWithWords()` function first computes line-level diff, then enriches consecutive Removed+Added pairs with word-level detail. The `Words` field is nil for pure additions/removals.
