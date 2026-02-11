# gottp

A Postman/Insomnia-like TUI API client built in Go.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## Features

- **Three-panel layout** — sidebar (collections), request editor, response viewer
- **HTTP client** with GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- **Auth support** — Basic, Bearer token, API key
- **Key-value editors** for headers and query params with enable/disable toggles
- **Syntax-highlighted responses** with JSON pretty-printing
- **Collections** saved as readable `.gottp.yaml` files
- **Environment variables** with `{{variable}}` interpolation
- **Vim-style modal editing** — Normal/Insert modes, j/k navigation
- **Responsive layout** — adapts from single-panel to three-panel based on terminal width
- **Catppuccin Mocha** color theme
- **Export to curl**
- **Command palette** (Ctrl+K), help overlay (?), tab management

## Install

```bash
go install github.com/serdar/gottp/cmd/gottp@latest
```

Or build from source:

```bash
git clone https://github.com/sadopc/gottp.git
cd gottp
make build
./bin/gottp
```

## Usage

```bash
# Launch with auto-discovered collection in current directory
gottp

# Launch with a specific collection
gottp --collection my-api.gottp.yaml
```

### Key Bindings

| Key | Action |
|-----|--------|
| `i` | Enter insert mode (focus URL input) |
| `Esc` | Return to normal mode |
| `Enter` / `Ctrl+R` | Send request |
| `Tab` / `Shift+Tab` | Cycle panel focus |
| `b` | Toggle sidebar |
| `Ctrl+K` | Command palette |
| `Ctrl+N` | New request tab |
| `Ctrl+W` | Close tab |
| `Ctrl+S` | Save request |
| `[` / `]` | Previous / next tab |
| `?` | Help overlay |
| `Ctrl+C` | Quit |

**In editor (focus field = 2):**
| Key | Action |
|-----|--------|
| `h/l` or arrows | Switch sub-tab (Params, Headers, Auth, Body) |
| `1-4` | Jump to sub-tab |

**In KV tables (Params/Headers):**
| Key | Action |
|-----|--------|
| `j/k` | Navigate rows |
| `Tab` | Toggle key/value column |
| `Enter` | Edit cell |
| `a` | Add row |
| `d` | Delete row |
| `Space` | Toggle enabled |

## Collection Format

Collections are stored as `.gottp.yaml` files:

```yaml
name: My API
version: "1"
items:
  - folder:
      name: Users
      items:
        - request:
            name: List Users
            method: GET
            url: "https://api.example.com/users"
            params:
              - { key: page, value: "1", enabled: true }
            headers:
              - { key: Accept, value: application/json, enabled: true }
        - request:
            name: Create User
            method: POST
            url: "https://api.example.com/users"
            body:
              type: json
              content: '{"name": "test"}'
```

## Roadmap

- [ ] Environment system with variable interpolation
- [ ] Request history (SQLite-backed)
- [ ] GraphQL support (query editor, introspection)
- [ ] gRPC support (server reflection, dynamic calls)
- [ ] WebSocket support (connect/send/receive)
- [ ] OAuth2 / AWS Sig v4 auth
- [ ] Pre/post-request scripting (JavaScript)
- [ ] Import from Postman, Insomnia, OpenAPI, curl
- [ ] Response diffing
- [ ] Custom themes

## License

MIT
