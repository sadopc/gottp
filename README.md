# gottp

A Postman/Insomnia-like TUI API client built in Go.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## Features

- **Three-panel layout** — sidebar (collections + history), request editor, response viewer
- **HTTP client** with GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- **Auth support** — Basic, Bearer token, API key (full interactive UI)
- **Key-value editors** for headers and query params with enable/disable toggles
- **Syntax-highlighted responses** with JSON pretty-printing
- **Response body search** — `/` or `Ctrl+F` to search, `n`/`N` to navigate matches
- **Response cookies tab** — parsed `Set-Cookie` headers with Name, Value, Domain, Path, HttpOnly, Secure
- **Collections** saved as readable `.gottp.yaml` files
- **Environment variables** with `{{variable}}` interpolation and environment switching (`Ctrl+E`)
- **Request history** — SQLite-backed, searchable, displayed in sidebar
- **cURL import/export** — copy requests as curl, import from clipboard
- **Jump mode** — press `f` for quick keyboard navigation to any panel/field
- **$EDITOR integration** — press `E` to edit request body in your preferred editor
- **Vim-style modal editing** — Normal/Insert/Jump/Search modes, j/k navigation
- **Responsive layout** — adapts from single-panel to three-panel based on terminal width
- **Catppuccin Mocha** color theme
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

**General:**
| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit |
| `Ctrl+K` | Command palette |
| `?` | Help overlay |
| `Tab` / `Shift+Tab` | Cycle panel focus |
| `Ctrl+Enter` | Send request |
| `S` | Send request (normal mode) |
| `Ctrl+N` | New request tab |
| `Ctrl+W` | Close tab |
| `Ctrl+S` | Save request |
| `Ctrl+E` | Switch environment |
| `[` / `]` | Previous / next tab |
| `f` | Jump mode (quick navigation) |
| `E` | Edit body in $EDITOR |

**Sidebar:**
| Key | Action |
|-----|--------|
| `b` | Toggle sidebar |
| `j` / `k` | Move cursor down / up |
| `Enter` | Open selected request |
| `/` | Search collections |

**Editor:**
| Key | Action |
|-----|--------|
| `i` | Enter insert mode |
| `Esc` | Return to normal mode |
| `h/l` or arrows | Switch sub-tab |
| `1-4` | Jump to sub-tab (Params, Headers, Auth, Body) |

**KV Tables (Params/Headers):**
| Key | Action |
|-----|--------|
| `j/k` | Navigate rows |
| `Tab` | Toggle key/value column |
| `Enter` | Edit cell |
| `a` | Add row |
| `d` | Delete row |
| `Space` | Toggle enabled |

**Response:**
| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `1-4` | Switch tabs (Body, Headers, Cookies, Timing) |
| `/` / `Ctrl+F` | Search in response body |
| `n` / `N` | Next / previous search match |
| `w` | Toggle word wrap |

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

## Environment Variables

Create an `environments.yaml` file alongside your collection:

```yaml
environments:
  - name: Development
    variables:
      base_url: "http://localhost:3000"
      api_key: "dev-key-123"
  - name: Production
    variables:
      base_url: "https://api.example.com"
      api_key: "prod-key-456"
```

Use `{{variable}}` syntax in URLs, headers, params, and body. Switch environments with `Ctrl+E` or the command palette.

## Roadmap

- [x] Environment system with variable interpolation
- [x] Request history (SQLite-backed)
- [x] Auth UI (Basic, Bearer, API Key)
- [x] cURL import/export
- [x] Response body search
- [x] Response cookies tab
- [x] Jump mode navigation
- [x] $EDITOR integration
- [ ] GraphQL support (query editor, introspection)
- [ ] gRPC support (server reflection, dynamic calls)
- [ ] WebSocket support (connect/send/receive)
- [ ] OAuth2 / AWS Sig v4 auth
- [ ] Pre/post-request scripting (JavaScript)
- [ ] Import from Postman, Insomnia, OpenAPI
- [ ] Response diffing
- [ ] Custom themes

## License

MIT
