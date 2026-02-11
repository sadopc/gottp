# gottp

A Postman/Insomnia-like TUI API client built in Go.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## Features

### Multi-Protocol Support
- **HTTP** — GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- **GraphQL** — query editor, variables, schema introspection (Ctrl+I)
- **WebSocket** — connect, send/receive messages, real-time message log
- **gRPC** — server reflection, service browser, dynamic unary calls

### Request Editor
- **Three-panel layout** — sidebar (collections + history), request editor, response viewer
- **Protocol switching** — Ctrl+P to cycle between HTTP, GraphQL, WebSocket, gRPC
- **Key-value editors** for headers and query params with enable/disable toggles
- **Auth support** — Basic, Bearer, API Key, OAuth2 (auth code w/ PKCE, client credentials, password grant), AWS Signature v4
- **Pre/post-request scripting** — JavaScript (ES5.1+) with `gottp` API for assertions, env var mutation, logging
- **$EDITOR integration** — press `E` to edit request body in your preferred editor

### Response Viewer
- **Syntax-highlighted responses** with JSON pretty-printing
- **Response body search** — `/` or `Ctrl+F` to search, `n`/`N` to navigate matches
- **Response cookies tab** — parsed `Set-Cookie` headers
- **Performance waterfall** — DNS, TCP, TLS, TTFB, and transfer timing breakdown
- **Response diffing** — set a baseline, compare subsequent responses with Myers diff
- **Script console** — view pre/post-script logs and test results
- **WebSocket message log** — color-coded sent/received messages

### Collections & Import/Export
- **Collections** saved as readable `.gottp.yaml` files
- **Environment variables** with `{{variable}}` interpolation and environment switching (Ctrl+E)
- **Request history** — SQLite-backed, searchable, displayed in sidebar
- **cURL import/export** — copy requests as curl, import from clipboard
- **HAR import/export** — import from browser DevTools, export request/response pairs
- **Import from Postman** (v2.1), **Insomnia** (v4), **OpenAPI** (3.0)

### Networking
- **Proxy support** — HTTP, HTTPS, and SOCKS5 proxies (global or per-request)
- **Cookie jar** — automatic cookie handling across requests within a collection
- **mTLS** — client certificate + key, CA bundles, skip TLS verification

### Navigation & Themes
- **Vim-style modal editing** — Normal/Insert/Jump/Search modes, j/k navigation
- **Jump mode** — press `f` for quick keyboard navigation to any panel/field
- **Command palette** (Ctrl+K), help overlay (?), tab management
- **8 built-in themes** — Catppuccin (Mocha, Latte, Frappe, Macchiato), Nord, Dracula, Gruvbox, Tokyo Night
- **Custom themes** — load YAML theme files from `~/.config/gottp/themes/`
- **Responsive layout** — adapts from single-panel to three-panel based on terminal width

### CLI Mode (Headless)
- **`gottp run`** — run requests from the terminal without the TUI
- **CI/CD integration** — JSON and JUnit XML output formats
- **Scripting in CI** — pre/post-request scripts execute in CLI mode too
- **Exit codes** — 0 = all pass, 1 = test assertion failure, 2 = request error

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

### TUI Mode (Interactive)

```bash
# Launch with auto-discovered collection in current directory
gottp

# Launch with a specific collection
gottp --collection my-api.gottp.yaml
```

### CLI Mode (Headless)

```bash
# Run all requests in a collection
gottp run api.gottp.yaml

# Run a single request by name
gottp run api.gottp.yaml --request "Get Users"

# Run all requests in a folder
gottp run api.gottp.yaml --folder "Auth"

# Use a specific environment
gottp run api.gottp.yaml --env Production

# JSON output for scripting
gottp run api.gottp.yaml --output json

# JUnit XML for CI pipelines
gottp run api.gottp.yaml --output junit > results.xml

# Verbose output with response bodies
gottp run api.gottp.yaml --verbose
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
| `Ctrl+P` | Switch protocol (HTTP/GraphQL/WebSocket/gRPC) |
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
| `1-6` | Jump to sub-tab |

**Response:**
| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `1-6` | Switch tabs (Body, Headers, Cookies, Timing, Diff, Console) |
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
  - folder:
      name: GraphQL
      items:
        - request:
            name: Countries
            method: POST
            url: "https://countries.trevorblades.com/graphql"
            graphql:
              query: "{ countries { name code } }"
              variables: "{}"
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

## Configuration

App config lives at `~/.config/gottp/config.yaml`:

```yaml
theme: catppuccin-mocha
vim_mode: true
default_timeout: 30s
editor: ""           # defaults to $EDITOR
script_timeout: 5s

# Proxy (HTTP/HTTPS/SOCKS5)
proxy_url: "http://proxy.example.com:8080"
no_proxy: "localhost,127.0.0.1"

# mTLS
tls:
  cert_file: "/path/to/client.pem"
  key_file: "/path/to/client-key.pem"
  ca_file: "/path/to/ca.pem"
  insecure_skip_verify: false
```

Proxy and TLS settings can also be overridden per-request in collection YAML. The HTTP client respects `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables as fallback.

## Pre/Post-Request Scripting

Add JavaScript scripts to requests. Pre-scripts can modify the request before it's sent; post-scripts can assert on the response.

```javascript
// Pre-script: set dynamic headers
gottp.request.setHeader("X-Request-ID", gottp.uuid());
gottp.request.setHeader("X-Timestamp", Date.now().toString());

// Post-script: validate response
gottp.test("Status is 200", function() {
  gottp.assert(gottp.response.statusCode === 200);
});

gottp.test("Response has data", function() {
  var body = JSON.parse(gottp.response.body);
  gottp.assert(body.data !== undefined, "Missing data field");
});

// Environment variable manipulation
gottp.setEnvVar("token", JSON.parse(gottp.response.body).token);
```

Available scripting API: `gottp.setEnvVar()`, `gottp.getEnvVar()`, `gottp.log()`, `gottp.test()`, `gottp.assert()`, `gottp.base64encode()`, `gottp.base64decode()`, `gottp.sha256()`, `gottp.uuid()`.

## Themes

Switch themes via the command palette (Ctrl+K > "Switch Theme"). Built-in themes:

- Catppuccin Mocha (default), Latte, Frappe, Macchiato
- Nord
- Dracula
- Gruvbox Dark
- Tokyo Night

Custom themes can be added as YAML files in `~/.config/gottp/themes/`.

## Roadmap

- [x] Environment system with variable interpolation
- [x] Request history (SQLite-backed)
- [x] Auth UI (Basic, Bearer, API Key)
- [x] cURL import/export
- [x] Response body search
- [x] Response cookies tab
- [x] Jump mode navigation
- [x] $EDITOR integration
- [x] Custom themes (8 built-in + custom YAML)
- [x] OAuth2 auth (auth code w/ PKCE, client credentials, password grant)
- [x] AWS Signature v4 auth
- [x] Import from Postman, Insomnia, OpenAPI
- [x] Response diffing
- [x] GraphQL support (query editor, introspection)
- [x] WebSocket support (connect/send/receive, message log)
- [x] gRPC support (server reflection, dynamic unary calls)
- [x] Pre/post-request scripting (JavaScript)
- [x] CLI mode (`gottp run` for headless/CI execution)
- [x] HAR import/export
- [x] Performance waterfall (DNS/TCP/TLS/TTFB/Transfer timing)
- [x] Proxy support (HTTP/HTTPS/SOCKS5)
- [x] Cookie jar (automatic cookie handling)
- [x] Certificate management (mTLS)
- [x] CI/CD pipeline (GitHub Actions + GoReleaser)
- [ ] Streaming gRPC (server/client/bidi)
- [ ] GraphQL subscriptions
- [ ] Request chaining / workflows

## License

MIT
