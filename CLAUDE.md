# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is gottp?

A Postman/Insomnia-like TUI API client built in Go with Bubble Tea. Three-panel layout (sidebar, editor, response) with vim-style modal editing, collections stored as YAML, and multiple theme support (Catppuccin variants, Nord, Dracula, Gruvbox, Tokyo Night). Supports HTTP, GraphQL, WebSocket, and gRPC protocols with environment variable interpolation, auth (basic/bearer/apikey/oauth2/awsv4), request history (SQLite), cURL/HAR import/export, Postman/Insomnia/OpenAPI import, response diffing, pre/post-request JavaScript scripting, jump navigation, proxy/mTLS/cookie jar support, and a headless CLI runner (`gottp run`).

## Build & Test Commands

```bash
make build          # Build to bin/gottp (with version ldflags)
make run            # Build and run
make test           # go test ./...
make test-race      # go test -race ./...
make lint           # golangci-lint run
make release-dry-run  # GoReleaser snapshot (test release build)
go test ./internal/protocol/http/ -run TestClient_GET   # Single test
go test ./internal/runner/ -v                            # Runner tests
go test ./internal/core/cookies/                         # Cookie jar tests
go test ./internal/core/tls/                             # TLS config tests
go test ./internal/import/har/                           # HAR parser tests
go test ./internal/export/har/                           # HAR exporter tests
```

Launch TUI: `./bin/gottp --collection path/to/file.gottp.yaml`

Headless CLI: `./bin/gottp run collection.gottp.yaml --env Production --output json`

Environment files: place `environments.yaml` next to the collection file. The first environment is auto-selected on startup.

## Architecture

**Bubble Tea MVU pattern**: All UI components implement `Update(msg) -> (Model, Cmd)` and `View() -> string`. The root model is `internal/app/app.go` which orchestrates panels, overlays, and message routing.

### Critical: Message Types Package

`internal/ui/msgs/msgs.go` is a **shared message types package** created to avoid import cycles between `app` and UI components. All `tea.Msg` types (`SendRequestMsg`, `RequestSentMsg`, `SetModeMsg`, `CopyAsCurlMsg`, `ImportCurlMsg`, `OpenEditorMsg`, `HistorySelectedMsg`, etc.) and enums (`AppMode`, `PanelFocus`) live here. Both `app` and all UI packages import `msgs` — never import `app` from UI packages.

### Key Packages

| Package | Role |
|---------|------|
| `internal/app/` | Root model, split into focused sub-modules (see below) |
| `internal/ui/msgs/` | Shared message types (breaks import cycles) |
| `internal/ui/panels/sidebar/` | Collection tree + history list (two-section navigation) |
| `internal/ui/panels/editor/` | Multi-protocol request editor (HTTPForm, GraphQLForm, WebSocketForm, GRPCForm) with ProtocolSelector |
| `internal/ui/panels/response/` | Response viewer (body/headers/cookies/timing/diff/console for HTTP; messages/headers/timing for WebSocket) |
| `internal/ui/components/` | Reusable: KVTable, TabBar, StatusBar, CommandPalette, Help, Modal, Toast, JumpOverlay |
| `internal/ui/theme/` | Theme catalog (8+ themes), lipgloss style definitions, custom YAML theme loader |
| `internal/ui/layout/` | Responsive three-panel layout calculator |
| `internal/protocol/` | Protocol interface, Registry, HTTP/GraphQL/WebSocket/gRPC clients |
| `internal/core/collection/` | YAML collection model, loader, saver |
| `internal/core/environment/` | Environment variables, `{{var}}` interpolation via `Resolve()` |
| `internal/core/history/` | SQLite-backed request history (Entry, Store) at `~/.local/share/gottp/history.db` |
| `internal/core/state/` | Central state (tabs, active collection, active env) |
| `internal/core/cookies/` | Thread-safe cookie jar wrapping `net/http/cookiejar` |
| `internal/core/tls/` | mTLS config builder (client cert+key, CA bundles, skip-verify) |
| `internal/export/` | curl export (`AsCurl()`), HAR export |
| `internal/import/` | Format auto-detection, curl/Postman/Insomnia/OpenAPI/HAR importers |
| `internal/runner/` | Headless CLI runner for `gottp run` with text/JSON/JUnit output |
| `internal/auth/oauth2/` | OAuth2 flows (auth code w/ PKCE, client credentials, password grant) |
| `internal/auth/awsv4/` | AWS Signature v4 request signing |
| `internal/diff/` | Myers diff algorithm for response diffing |
| `internal/scripting/` | JavaScript pre/post-request scripting via goja engine |
| `internal/config/` | App config from `~/.config/gottp/config.yaml` |

### app.go Sub-Module Structure

The `internal/app/` package is split into focused files:

| File | Contents |
|------|----------|
| `app.go` | `App` struct, `New()`, `Init()`, `Update()`, `View()`, `resizePanels()`, layout helpers |
| `app_keys.go` | `handleGlobalKey()`, `handlePanelKey()`, `updateEditorInsert()`, `cycleFocus()`, `updateFocus()`, `activateJumpMode()` |
| `app_request.go` | `sendRequest()`, `handleRequestSent()`, `initiateOAuth2()`, `handleOAuth2Token()`, introspection/reflection handlers, script result conversion |
| `app_overlays.go` | `handleSwitchTheme()`, `handleImportFile()`, `handleImportComplete()`, `handleSetBaseline()`, `openExternalEditor()` |
| `app_tabs.go` | `syncTabs()`, `loadActiveRequest()`, `loadHistory()`, `handleRequestSelected()`, `handleHistorySelected()` |
| `app_save.go` | `saveCollection()`, `authConfigToCollection()`, `copyAsCurl()`, `importCurl()` |
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

`cmd/gottp/main.go` routes to either TUI mode (default) or `runCmd()` when `os.Args[1] == "run"`. The runner in `internal/runner/` loads the collection, resolves env vars, executes requests with scripting, and outputs results. Exit codes: 0=pass, 1=test assertion failure, 2=request error. Output formats: text (human-readable), json (structured), junit (CI XML).

### UI Modes

`ModeNormal` (navigate with j/k, Tab between panels) → `ModeInsert` (typing in text fields, Esc to return) → `ModeJump` (f key, type label to jump) → `ModeSearch` (/ in response body). `i` enters insert mode, `Esc` exits. Components track their own editing state via `Editing() bool`.

### Environment Variable Resolution

`sendRequest()` calls `environment.Resolve()` on URL, header values, param values, body, and auth fields before executing. Resolution priority: env vars > collection vars > OS env.

### Save Flow

`saveCollection()` syncs form state back to `store.ActiveRequest()` (method, URL, params, headers, body, auth) before writing YAML.

### Multi-Protocol Editor

`editor.Model` wraps four protocol-specific forms (`HTTPForm`, `GraphQLForm`, `WebSocketForm`, `GRPCForm`) and a `ProtocolSelector` widget. `Ctrl+P` cycles protocols. All form access goes through delegation methods on `editor.Model`:
- `BuildRequest()`, `GetParams()`, `GetHeaders()`, `GetBodyContent()`, `SetBody()`, `BuildAuth()`, `FocusURL()` — each delegates to the active form
- `LoadRequest()` auto-detects protocol from collection request fields (GraphQL/GRPC/WebSocket config)
- **Never use `editor.Form()` directly** — use the delegation methods instead

### Response Panel Modes

The response panel has two display modes:
- **HTTP mode**: tabs = Body, Headers, Cookies, Timing, Diff, Console
- **WebSocket mode**: tabs = Messages, Headers, Timing

`SetMode(proto)` switches between them. The Console tab shows JavaScript script output (logs + test results). The Timing tab shows a waterfall visualization (DNS/TCP/TLS/TTFB/Transfer) when `TimingDetail` is available in the response.

### HTTP Client Infrastructure

The HTTP client (`internal/protocol/http/client.go`) supports:
- **Proxy**: `SetProxy(proxyURL, noProxy string)` — HTTP/HTTPS/SOCKS5 proxy with per-request override via `req.ProxyURL`. Respects `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` env vars as fallback.
- **Cookie Jar**: `SetCookieJar(jar)` — auto-captures `Set-Cookie` and sends cookies on subsequent requests within a collection.
- **mTLS**: `SetTLSConfig(cfg)` — client cert+key, CA bundles, skip-verify via `internal/core/tls/`.
- **Timing**: Uses `net/http/httptrace` to capture DNS/TCP/TLS/TTFB/Transfer breakdown, returned as `protocol.TimingDetail` in responses.

### Auth Section

`AuthSection` in `editor/auth_section.go` supports none/basic/bearer/apikey/oauth2/awsv4. Vim-style j/k navigation, h/l or space to cycle type, enter to edit fields. `BuildAuth()` returns `*protocol.AuthConfig`, `LoadAuth()` populates from `*collection.Auth`.

### Scripting Engine

`internal/scripting/` provides a JavaScript (ES5.1+) engine via `goja`. Pre-scripts can mutate the request (URL, headers, params, body); post-scripts have read-only access to the response. The `gottp` global object provides `setEnvVar()`, `getEnvVar()`, `log()`, `test(name, fn)`, `assert()`, `base64encode/decode()`, `sha256()`, `uuid()`. Each execution uses a fresh runtime with configurable timeout (default 5s).

### History

SQLite store at `~/.local/share/gottp/history.db` via `modernc.org/sqlite` (pure Go, no CGO). Entries saved after each successful request. Sidebar displays recent 20 with relative timestamps. Selecting a history entry opens it in a new tab.

### CI/CD

GitHub Actions workflows in `.github/workflows/`:
- `ci.yml` — test, lint, build on push/PR (ubuntu + macos, Go 1.25.x)
- `release.yml` — GoReleaser on tag push (v*) for cross-platform binaries

`.goreleaser.yml` builds for linux/darwin (amd64/arm64) + windows (amd64) with version ldflags.

## Conventions

- Value receivers for `Update()` and `View()`, pointer receivers for mutating helpers (`startEditing`, `commitEdit`, `SetSize`, `SetPairs`)
- Sub-components (KVTable, HTTPForm, AuthSection) return `(Model, Cmd)` from Update, not `tea.Model` — they use concrete types
- Responsive layout breakpoints: <60 cols = single panel, <100 cols = two panel, >=100 = three panel
- Collection files use `.gottp.yaml` extension
- Environment files use `environments.yaml` (placed next to collection)
- HTTP method colors: GET=green, POST=yellow, PUT=blue, PATCH=peach, DELETE=red (Catppuccin palette)
- `app.New()` signature: `New(col *collection.Collection, colPath string, cfg config.Config) App`
- Sub-models that need theme colors (e.g., DiffModel, ConsoleModel, WSLogModel) take both `theme.Theme` and `theme.Styles` in their constructors; store `th` for colors and `styles` for pre-computed lipgloss styles
- Protocol-specific editor forms (GraphQLForm, etc.) follow the same patterns as HTTPForm: sub-tabs, `BuildRequest()`, `LoadRequest()`, `Editing() bool`
- Config fields for new infra (proxy, TLS) use `yaml:"...,omitempty"` tags and zero-value defaults
- Import format detection lives in `internal/import/detect.go` — add new format checks there when adding importers

## Known Gotchas

- KVTable View() must build the cursor prefix (`"> "` / `"  "`) separately before joining with styled content. Slicing into styled strings (e.g., `row[2:]`) breaks ANSI escape sequences.
- Response body search uses plain text (not syntax-highlighted) for match highlighting to avoid ANSI escape interference.
- Command palette has dynamic mode: `OpenEnvPicker()` replaces commands temporarily; `ResetCommands()` restores defaults on close.
- The HTTP client creates a new `http.Transport` per-request when proxy/TLS config is present to support per-request proxy overrides. Without custom config, it reuses the default transport.
- The `internal/core/tls` package is imported as `gotls` in config to avoid conflict with `crypto/tls`.
