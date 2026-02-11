# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is gottp?

A Postman/Insomnia-like TUI API client built in Go with Bubble Tea. Three-panel layout (sidebar, editor, response) with vim-style modal editing, collections stored as YAML, and Catppuccin Mocha theme.

## Build & Test Commands

```bash
make build          # Build to bin/gottp (with version ldflags)
make run            # Build and run
make test           # go test ./...
make test-race      # go test -race ./...
make lint           # golangci-lint run
go test ./internal/protocol/http/ -run TestClient_GET   # Single test
```

Launch with a collection: `./bin/gottp --collection path/to/file.gottp.yaml`

## Architecture

**Bubble Tea MVU pattern**: All UI components implement `Update(msg) -> (Model, Cmd)` and `View() -> string`. The root model is `internal/app/app.go` which orchestrates panels, overlays, and message routing.

### Critical: Message Types Package

`internal/ui/msgs/msgs.go` is a **shared message types package** created to avoid import cycles between `app` and UI components. All `tea.Msg` types (`SendRequestMsg`, `RequestSentMsg`, `SetModeMsg`, etc.) and enums (`AppMode`, `PanelFocus`) live here. Both `app` and all UI packages import `msgs` — never import `app` from UI packages.

### Key Packages

| Package | Role |
|---------|------|
| `internal/app/` | Root model, global keybindings, orchestration |
| `internal/ui/msgs/` | Shared message types (breaks import cycles) |
| `internal/ui/panels/sidebar/` | Collection tree + history |
| `internal/ui/panels/editor/` | Request editor (HTTPForm with method/URL/params/headers/body) |
| `internal/ui/panels/response/` | Response viewer (body with syntax highlighting, headers, timing) |
| `internal/ui/components/` | Reusable: KVTable, TabBar, StatusBar, CommandPalette, Help, Modal, Toast |
| `internal/ui/theme/` | Catppuccin Mocha styles, lipgloss style definitions |
| `internal/ui/layout/` | Responsive three-panel layout calculator |
| `internal/protocol/` | Protocol interface + HTTP client implementation |
| `internal/core/collection/` | YAML collection model, loader, saver |
| `internal/core/environment/` | Environment variables, `{{var}}` interpolation |
| `internal/core/state/` | Central state (tabs, active collection) |
| `internal/export/` | curl export |
| `internal/config/` | App config from `~/.config/gottp/config.yaml` |

### Message Routing in app.go

`Update()` priority: overlays (command palette > help > modal) → editor insert mode → global keys (Ctrl+R send, Ctrl+K palette, etc.) → panel-specific keys. The `handleGlobalKey()` and `handlePanelKey()` methods handle this dispatch.

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

`ModeNormal` (navigate with j/k, Tab between panels) → `ModeInsert` (typing in text fields, Esc to return). `i` enters insert mode, `Esc` exits. Components track their own editing state via `Editing() bool`.

## Conventions

- Value receivers for `Update()` and `View()`, pointer receivers for mutating helpers (`startEditing`, `commitEdit`, `SetSize`, `SetPairs`)
- Sub-components (KVTable, HTTPForm) return `(Model, Cmd)` from Update, not `tea.Model` — they use concrete types
- Responsive layout breakpoints: <60 cols = single panel, <100 cols = two panel, >=100 = three panel
- Collection files use `.gottp.yaml` extension
- HTTP method colors: GET=green, POST=yellow, PUT=blue, PATCH=peach, DELETE=red (Catppuccin palette)

## Known Gotcha

KVTable View() must build the cursor prefix (`"> "` / `"  "`) separately before joining with styled content. Slicing into styled strings (e.g., `row[2:]`) breaks ANSI escape sequences.
