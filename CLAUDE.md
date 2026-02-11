# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is gottp?

A Postman/Insomnia-like TUI API client built in Go with Bubble Tea. Three-panel layout (sidebar, editor, response) with vim-style modal editing, collections stored as YAML, and Catppuccin Mocha theme. Supports environment variable interpolation, auth (basic/bearer/apikey), request history (SQLite), cURL import/export, response search, and jump navigation.

## Build & Test Commands

```bash
make build          # Build to bin/gottp (with version ldflags)
make run            # Build and run
make test           # go test ./...
make test-race      # go test -race ./...
make lint           # golangci-lint run
go test ./internal/protocol/http/ -run TestClient_GET   # Single test
go test ./internal/import/curl/                         # cURL parser tests
go test ./internal/core/history/                        # History store tests
```

Launch with a collection: `./bin/gottp --collection path/to/file.gottp.yaml`

Environment files: place `environments.yaml` next to the collection file. The first environment is auto-selected on startup.

## Architecture

**Bubble Tea MVU pattern**: All UI components implement `Update(msg) -> (Model, Cmd)` and `View() -> string`. The root model is `internal/app/app.go` which orchestrates panels, overlays, and message routing.

### Critical: Message Types Package

`internal/ui/msgs/msgs.go` is a **shared message types package** created to avoid import cycles between `app` and UI components. All `tea.Msg` types (`SendRequestMsg`, `RequestSentMsg`, `SetModeMsg`, `CopyAsCurlMsg`, `ImportCurlMsg`, `OpenEditorMsg`, `HistorySelectedMsg`, etc.) and enums (`AppMode`, `PanelFocus`) live here. Both `app` and all UI packages import `msgs` — never import `app` from UI packages.

### Key Packages

| Package | Role |
|---------|------|
| `internal/app/` | Root model, global keybindings, orchestration |
| `internal/ui/msgs/` | Shared message types (breaks import cycles) |
| `internal/ui/panels/sidebar/` | Collection tree + history list (two-section navigation) |
| `internal/ui/panels/editor/` | Request editor (HTTPForm with method/URL/params/headers/auth/body) |
| `internal/ui/panels/response/` | Response viewer (body with syntax highlighting + search, headers, cookies, timing) |
| `internal/ui/components/` | Reusable: KVTable, TabBar, StatusBar, CommandPalette, Help, Modal, Toast, JumpOverlay |
| `internal/ui/theme/` | Catppuccin Mocha styles, lipgloss style definitions |
| `internal/ui/layout/` | Responsive three-panel layout calculator |
| `internal/protocol/` | Protocol interface + HTTP client implementation |
| `internal/core/collection/` | YAML collection model, loader, saver |
| `internal/core/environment/` | Environment variables, `{{var}}` interpolation via `Resolve()` |
| `internal/core/history/` | SQLite-backed request history (Entry, Store) at `~/.local/share/gottp/history.db` |
| `internal/core/state/` | Central state (tabs, active collection, active env) |
| `internal/export/` | curl export (`AsCurl()`) |
| `internal/import/curl/` | curl import parser (`ParseCurl()`) with shell tokenizer |
| `internal/config/` | App config from `~/.config/gottp/config.yaml` |

### Message Routing in app.go

`Update()` priority: overlays (command palette > help > modal > jump) → editor insert mode → global keys (Ctrl+Enter send, Ctrl+K palette, etc.) → panel-specific keys. The `handleGlobalKey()` and `handlePanelKey()` methods handle this dispatch.

### Protocol Interface

```go
type Protocol interface {
    Name() string
    Execute(ctx context.Context, req *Request) (*Response, error)
    Validate(req *Request) error
}
```

Currently only HTTP is implemented. GraphQL, gRPC, WebSocket are planned.

### UI Modes

`ModeNormal` (navigate with j/k, Tab between panels) → `ModeInsert` (typing in text fields, Esc to return) → `ModeJump` (f key, type label to jump) → `ModeSearch` (/ in response body). `i` enters insert mode, `Esc` exits. Components track their own editing state via `Editing() bool`.

### Environment Variable Resolution

`sendRequest()` calls `environment.Resolve()` on URL, header values, param values, body, and auth fields before executing. Resolution priority: env vars > collection vars > OS env.

### Save Flow

`saveCollection()` syncs form state back to `store.ActiveRequest()` (method, URL, params, headers, body, auth) before writing YAML.

### Auth Section

`AuthSection` in `editor/auth_section.go` supports none/basic/bearer/apikey. Vim-style j/k navigation, h/l or space to cycle type, enter to edit fields. `BuildAuth()` returns `*protocol.AuthConfig`, `LoadAuth()` populates from `*collection.Auth`.

### History

SQLite store at `~/.local/share/gottp/history.db` via `modernc.org/sqlite` (pure Go, no CGO). Entries saved after each successful request. Sidebar displays recent 20 with relative timestamps. Selecting a history entry opens it in a new tab.

## Conventions

- Value receivers for `Update()` and `View()`, pointer receivers for mutating helpers (`startEditing`, `commitEdit`, `SetSize`, `SetPairs`)
- Sub-components (KVTable, HTTPForm, AuthSection) return `(Model, Cmd)` from Update, not `tea.Model` — they use concrete types
- Responsive layout breakpoints: <60 cols = single panel, <100 cols = two panel, >=100 = three panel
- Collection files use `.gottp.yaml` extension
- Environment files use `environments.yaml` (placed next to collection)
- HTTP method colors: GET=green, POST=yellow, PUT=blue, PATCH=peach, DELETE=red (Catppuccin palette)
- `app.New()` signature: `New(col *collection.Collection, colPath string, cfg config.Config) App`

## Known Gotchas

- KVTable View() must build the cursor prefix (`"> "` / `"  "`) separately before joining with styled content. Slicing into styled strings (e.g., `row[2:]`) breaks ANSI escape sequences.
- Response body search uses plain text (not syntax-highlighted) for match highlighting to avoid ANSI escape interference.
- Command palette has dynamic mode: `OpenEnvPicker()` replaces commands temporarily; `ResetCommands()` restores defaults on close.
