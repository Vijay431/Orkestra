# Acknowledgments

Orkestra exists because of the work of many open-source authors. This page is a thank-you note.

---

## 🌟 Direct Dependencies

These libraries do real work in Orkestra's request path. Without them, there's no project.

| Library | License | What It Does for Us |
|---------|---------|---------------------|
| 🛠️ **[mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)** | MIT | The MCP protocol implementation — JSON-RPC transport, tool registration, SSE streaming. Every Orkestra tool is one `mcp.NewTool` call. |
| 🗄️ **[modernc.org/sqlite](https://gitlab.com/cznic/sqlite)** | BSD-3-Clause | Pure-Go SQLite driver. The reason Orkestra ships as a single 20 MB scratch container with no CGo, no `libsqlite3.so`, no native build chain. |

---

## 🧱 Indirect Dependencies (the foundation)

Pulled in transitively, but each one solves a real problem:

| Library | License | Role |
|---------|---------|------|
| [google/uuid](https://github.com/google/uuid) | BSD-3-Clause | Stable ticket ID generation |
| [google/jsonschema-go](https://github.com/google/jsonschema-go) | Apache-2.0 | JSON schema validation inside mcp-go |
| [yosida95/uritemplate](https://github.com/yosida95/uritemplate) | BSD-3-Clause | URI template parsing for MCP resources |
| [dustin/go-humanize](https://github.com/dustin/go-humanize) | MIT | Human-readable durations and sizes |
| [spf13/cast](https://github.com/spf13/cast) | MIT | Type-safe coercion of MCP tool arguments |
| [mattn/go-isatty](https://github.com/mattn/go-isatty) | MIT | Terminal detection for log formatting |
| [ncruces/go-strftime](https://github.com/ncruces/go-strftime) | MIT | Date formatting compatibility for SQLite |
| [modernc.org/libc](https://gitlab.com/cznic/libc) · [mathutil](https://gitlab.com/cznic/mathutil) · [memory](https://gitlab.com/cznic/memory) | BSD-3-Clause | The pure-Go runtime that makes the SQLite port possible |
| [remyoudompheng/bigfft](https://github.com/remyoudompheng/bigfft) | BSD-3-Clause | Fast big-int multiplication in modernc's runtime |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | BSD-3-Clause | OS syscall bindings |

---

## 🛡️ Tooling We Lean On

These don't ship in the binary, but they keep the project healthy:

- **[Go](https://go.dev)** — the language and standard library
- **[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)** — call-graph-aware CVE scanning on every CI run
- **[staticcheck](https://staticcheck.dev)** — static analysis catching the bugs `go vet` misses
- **[Dependabot](https://github.com/dependabot)** — weekly dependency PRs

---

## 💡 Inspiration

Ideas borrowed (with respect) from:

- **[Linear](https://linear.app)** — the workflow shape (backlog → in-progress → done) and ticket ID format
- **[Mermaid](https://mermaid.js.org)** — the diagram language `ticket_diagram` emits
- **[Conventional Commits](https://www.conventionalcommits.org)** — our commit style
- **[Keep a Changelog](https://keepachangelog.com)** — our changelog format
- **TOON** — the compact notation idea grew out of conversations about token economy when LLMs read structured data

---

## 🙏 Special Thanks

To everyone who reports a bug, files an issue, sends a PR, or just stars the repo — you're the reason this exists.

Found something you maintain that should be on this list? Open a PR.
