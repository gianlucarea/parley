# parley

![Jack Sparrow GIF](https://media4.giphy.com/media/v1.Y2lkPTc5MGI3NjExdDE5dnM5cTdnZG15d2V5dmlqcTdmeHQ0Zzlxd2xkbWR4czN5M3p1MyZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/10X22vzgNamaiI/giphy.gif)

> A git-friendly TUI HTTP client. Collections are plain TOML files — diff them, commit them, review them.

---

## What is this?

🛠️ **Project Origin**: This project originally began as a highly successful AI-assisted engineering experiment designed to build a lightweight, specialized tool for personal development workflows.

Most HTTP clients store your requests in a proprietary binary format or a JSON blob that's painful to review in a pull request. Parley stores everything as plain TOML: one file per collection, readable by humans, diffable by Git.

![Demo GIF](https://media1.giphy.com/media/v1.Y2lkPTc5MGI3NjExbGh4cnFnMzd0cTJ5YmZyNmx0MDV1M3RucWl2OWFpYXpxbW1vMHduaCZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/nG19UJLcVRowjSG7E4/giphy.gif
)
---

## What's implemented

### Collection parser (`internal/collection`)

- Loads `.toml` collection files from a file path or an entire directory
- Supports multiple requests per file, each with a name, method, URL, headers, body, and auth
- Collection name is derived from the filename (no extension)

### Environment & variable substitution (`internal/env`)

- Parses a `.parley.env` file (key=value, optional quotes, `#` comments)
- Replaces `{{VAR}}` placeholders in URLs, headers, bodies, and auth fields
- Multiple named environments via `.parley.<name>.env` files, selected with `--env` or switched live with `e`
- Substitution precedence: **shell environment > `.parley.<name>.env` > `.parley.env`**

### HTTP executor (`internal/http`)

- Executes `GET`, `POST`, `PUT`, `PATCH`, `DELETE` requests
- Injects `Authorization: Bearer <token>` for `bearer` auth
- Injects `Authorization: Basic <b64>` for `basic` auth
- Auth field values also support `{{VAR}}` substitution
- Returns status code, status text, response headers, body, and elapsed time

### TUI (`internal/tui`)

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lipgloss](https://github.com/charmbracelet/lipgloss).

Three-pane layout:

| Pane | Description |
|------|-------------|
| **Sidebar** | Lists all collections and their requests. Method badges are color-coded (`GET` green, `POST` yellow, `PUT` blue, `PATCH` cyan, `DELETE` red). Built-in filter with `/`. |
| **Editor** | Shows the selected request's method, URL, headers, body, and auth. Press `Enter` to execute. |
| **Viewer** | Displays the response: status code, elapsed time, and a syntax-highlighted scrollable body (powered by [Chroma](https://github.com/alecthomas/chroma)). |

Keyboard shortcuts:

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus between panes |
| `↑` / `↓` | Navigate requests in sidebar |
| `Enter` | Select request (sidebar) / Run request (editor) |
| `/` | Filter requests |
| `Esc` | Clear filter |
| `e` | Cycle environment (`default` → named envs) |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

---

## Quickstart

```sh
# 1. Build from source
git clone https://github.com/gianlucarea/parley
cd parley
go build -o parley ./cmd/parley

# 2. Create a collection file
cat > api.toml << 'EOF'
[[requests]]
name   = "List Users"
method = "GET"
url    = "{{BASE_URL}}/users"

[requests.headers]
Accept = "application/json"

[[requests]]
name   = "Create User"
method = "POST"
url    = "{{BASE_URL}}/users"
body   = '{"name":"Alice"}'

[requests.auth]
type  = "bearer"
token = "{{API_TOKEN}}"
EOF

# 3. Create a .parley.env (keep this gitignored)
cat > .parley.env << 'EOF'
BASE_URL=https://api.example.com
API_TOKEN=my-secret-token
EOF

# 4. Launch
./parley .
```

---

## Collection format

One `.toml` file per logical collection. Put multiple collection files in the same directory — parley loads them all.

```toml
[[requests]]
name   = "Get User"
method = "GET"
url    = "{{BASE_URL}}/users/{{USER_ID}}"

[requests.headers]
Accept = "application/json"

[requests.auth]
type  = "bearer"
token = "{{API_TOKEN}}"

[[requests]]
name   = "Create Post"
method = "POST"
url    = "{{BASE_URL}}/posts"
body   = '{"title":"hello"}'

[requests.auth]
type     = "basic"
username = "{{USER}}"
password = "{{PASS}}"
```

**Supported methods:** `GET` `POST` `PUT` `PATCH` `DELETE`

**Auth types:**

| type | required fields |
|------|-----------------|
| `bearer` | `token` |
| `basic` | `username`, `password` |

---

## `.parley.env` reference

```dotenv
# Lines starting with # are comments
BASE_URL=https://api.example.com
API_TOKEN=super-secret
USER=alice
PASS=hunter2
```

Shell environment variables always win, so you can override per-run without editing any file:

```sh
API_TOKEN=other-token ./parley .
```

---

## Named environments

Put shared values in `.parley.env` and per-environment overrides in `.parley.<name>.env` (all in the directory you launch parley from):

```
.parley.env          # shared defaults (e.g. API_TOKEN)
.parley.dev.env      # BASE_URL=http://localhost:8080
.parley.staging.env  # BASE_URL=https://staging.example.com
.parley.prod.env     # BASE_URL=https://api.example.com
```

Pick one at startup, or press `e` in the TUI to cycle through them. The active environment is shown in the bottom bar.

```sh
./parley --env staging .   # or: ./parley -e staging .
```

Precedence: **shell environment > `.parley.<name>.env` > `.parley.env`**. Without `--env`, only `.parley.env` is used (`default`). An unknown name exits with the list of available environments. `.parley.*.env` files are gitignored.

---

## Possible future developments

- **Request history** — persist and browse previous responses per request
- **Export to curl** — copy the selected request as a ready-to-paste `curl` command
- **Import from OpenAPI / Swagger** — generate a collection from an API spec file or URL
- **Response assertions** — define expected status codes or body patterns and highlight failures
- **Request chaining** — extract values from a response (e.g. a token) and inject them into subsequent requests
- **Pre-built binaries & releases** — goreleaser pipeline for Linux / macOS / Windows (amd64 + arm64)
- **WebSocket / SSE support** — stream responses in the viewer pane
- **gRPC support** — proto-based collections alongside HTTP ones
- **Response diffing** — compare two runs of the same request side by side
