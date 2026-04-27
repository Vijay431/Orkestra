# Orkestra

Self-hosted MCP ticket server for autonomous LLM agents. No cloud, no rate limits, no per-request costs. All data stays in a local SQLite file inside a ~20 MB Docker image.

Solves the three LinearMCP pain points:

| Pain Point                 | Orkestra Solution                     |
| -------------------------- | ------------------------------------- |
| API cost & rate limits     | Local process, zero external calls    |
| Token-heavy JSON responses | TOON format — 60–70% fewer tokens     |
| Cloud dependency           | Single Docker container, runs offline |

---

## Quick Start

```bash
# 1. Clone and onboard (auto-detects Claude Code, Cursor, Copilot, Windsurf, Zed)
git clone https://github.com/vijay431/orkestra
PROJECT_ID=myapp ./install.sh

# 2. Verify
curl http://localhost:8080/health
# → {"status":"ok","project":"myapp","db_ok":true,...}

# 3. Claude Code picks up the MCP server automatically after install
```

Or start manually:

```bash
PROJECT_ID=myapp docker compose up -d
claude mcp add orkestra-myapp --transport http http://localhost:8080/sse
```

---

## Local Development

No Docker required for development. You need Go 1.22+.

```bash
# Run the server directly against a temp database
PROJECT_ID=dev DB_PATH=/tmp/dev.db go run ./cmd/server

# The server starts on :8080 by default
curl http://localhost:8080/health
```

**All environment variables with their defaults:**

| Env var           | Default             | Description                                              |
| ----------------- | ------------------- | -------------------------------------------------------- |
| `PROJECT_ID`      | **(required)**      | Ticket ID prefix (`myapp`) and scope filter              |
| `DB_PATH`         | `/data/orkestra.db` | SQLite file path                                         |
| `PORT`            | `8080`              | HTTP listen port                                         |
| `BIND_ADDR`       | `0.0.0.0`           | Listen address (use `127.0.0.1` to bind localhost only)  |
| `MCP_TOKEN`       | _(unset)_           | Bearer token for `/sse` and `/message` (optional)        |
| `LOG_LEVEL`       | `info`              | `debug` \| `info` \| `warn` \| `error`                   |
| `BACKUP_DIR`      | `/data/backups`     | Backup destination directory                             |
| `BACKUP_INTERVAL` | `1h`                | Backup frequency (Go duration string)                    |
| `BACKUP_KEEP`     | `24`                | Number of backup files to retain                         |

**Browsing the database:**

```bash
# SQLite CLI
sqlite3 /tmp/dev.db ".tables"
sqlite3 /tmp/dev.db "SELECT id, title, status, priority FROM tickets WHERE archived_at IS NULL;"

# Or use DB Browser for SQLite (GUI): https://sqlitebrowser.org
```

---

## TOON Format

All tool responses use TOON (Tokens Object Oriented Notation) — a compact typed shorthand that cuts average ticket representation from ~400 tokens (JSON) to ~120 tokens.

```
TOON/1 T{id:myapp-001,t:"Fix auth bug",s:ip,p:h,typ:bug,lbl:[auth,sec],ca:2024-01-15,ua:2024-01-15T10:00:00Z}
```

**Field aliases:**

| Full name    | TOON  | Full name     | TOON  |
| ------------ | ----- | ------------- | ----- |
| `id`         | `id`  | `exec_mode`   | `em`  |
| `title`      | `t`   | `exec_order`  | `ord` |
| `status`     | `s`   | `parent_id`   | `par` |
| `priority`   | `p`   | `children`    | `ch`  |
| `type`       | `typ` | `description` | `d`   |
| `labels`     | `lbl` | `comments`    | `cmt` |
| `assignee`   | `as`  | `created_at`  | `ca`  |
| `updated_at` | `ua`  | `links`       | `lnk` |

**Enum codes:**

| Domain   | Codes                                                                                    |
| -------- | ---------------------------------------------------------------------------------------- |
| Status   | `bk`=backlog · `td`=todo · `ip`=in_progress · `dn`=done · `bl`=blocked · `cl`=cancelled |
| Priority | `cr`=critical · `h`=high · `m`=medium · `l`=low                                          |
| Type     | `bug` · `ft`=feature · `tsk`=task · `ep`=epic · `chr`=chore                              |
| ExecMode | `par`=parallel · `seq`=sequential                                                        |
| LinkType | `blk`=blocks · `rel`=relates · `dup`=duplicates                                          |

**Comment encoding (inside `cmt` array):** `C{a:author,t:"body",ts:2024-01-15T10:04}`
**Link encoding (inside `lnk` array):** `L{f:from_id,t:to_id,k:blk}`

**Error envelopes:**

```
TOON/1 ERR{code:not_found,msg:"myapp-999 does not exist"}
TOON/1 ERR{code:conflict,msg:"myapp-002 already claimed"}
TOON/1 ERR{code:seq_blocked,msg:"myapp-022 blocked: ord=1 not done"}
TOON/1 ERR{code:invalid,msg:"exec_order must be unique within parent"}
```

---

## MCP Tools Reference

### Ticket Lifecycle

| Tool             | Required args | Optional args                                                                                     | Returns             | Notes                                                     |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------- | ------------------- | --------------------------------------------------------- |
| `ticket_create`  | `title`       | `type` `priority` `description` `labels` `parent_id` `exec_mode` `exec_order`                    | `TOON/1 T{...}`     | Lands in `bk`; `exec_order` required for `seq` children   |
| `ticket_get`     | `id`          | —                                                                                                 | `TOON/1 T{...}`     | Full ticket: comments, links, child IDs                   |
| `ticket_claim`   | `id`          | —                                                                                                 | `TOON/1 T{...}`     | Atomic CAS → `ip`; `ua` field is your etag                |
| `ticket_update`  | `id`          | `etag` `title` `status` `priority` `type` `description` `assignee` `labels` `exec_mode` `exec_order` | `TOON/1 T{...}` | Supply `etag` (ua field) for optimistic locking          |
| `ticket_archive` | `id`          | —                                                                                                 | `TOON/1 {ok:true}`  | Soft delete — sets `archived_at`                          |

### Discovery

| Tool              | Required | Key optional args                                              | Returns                                  | Notes                                              |
| ----------------- | -------- | -------------------------------------------------------------- | ---------------------------------------- | -------------------------------------------------- |
| `ticket_list`     | —        | `status` `priority` `type` `labels` `limit` `include_archived` | `TOON/1 [...]`                           | All tickets; ordered by `created_at DESC`          |
| `ticket_backlog`  | —        | `priority` `type` `labels` `limit`                             | `TOON/1 [...]`                           | Priority-ordered `bk` tickets (cr→h→m→l)           |
| `ticket_board`    | —        | `type` `labels`                                                | `TOON/1 BOARD{bk:[],td:[],ip:[],dn:[]}` | Kanban view; empty buckets omitted                 |
| `ticket_search`   | `query`  | `include_archived`                                             | `TOON/1 [...]`                           | FTS5 ranked full-text search                       |
| `ticket_children` | `id`     | `recursive` `depth` (1–10)                                     | `TOON/1 [...]`                           | Sorted by `exec_order`; `recursive` flattens tree  |
| `ticket_diagram`  | `id`     | `depth` (1–10)                                                 | Mermaid flowchart                        | Parallel → ⚡ subgraph; sequential → 🔗 subgraph   |

### Collaboration

| Tool             | Required args                   | Optional args | Returns                           |
| ---------------- | ------------------------------- | ------------- | --------------------------------- |
| `ticket_comment` | `id`, `body`                    | `author`      | `TOON/1 T{...}` (updated ticket)  |
| `ticket_link`    | `from_id`, `to_id`, `link_type` | —             | `TOON/1 {ok:true}`                |

---

## Workflows

### Core loop

```
ticket_backlog          → pick highest-priority item
ticket_claim id=X       → atomically move to ip (save ua as etag)
(do the work)
ticket_update id=X s=dn etag=<ua>  → mark done
```

### Epic with parallel swarm

```
ticket_create typ=ep t="Auth system"
ticket_create parent_id=<epic> t="JWT middleware"   # exec_mode=par by default
ticket_create parent_id=<epic> t="OAuth provider"
ticket_create parent_id=<epic> t="Session store"
# All 3 children can be claimed simultaneously by different agents
ticket_diagram id=<epic>  → Mermaid flowchart
```

### Sequential pipeline

```
ticket_create typ=tsk t="Deploy pipeline" em=seq
ticket_create parent_id=<pipeline> t="Run tests"        em=seq ord=1
ticket_create parent_id=<pipeline> t="Build image"      em=seq ord=2
ticket_create parent_id=<pipeline> t="Push to registry" em=seq ord=3
# Claiming ord=2 while ord=1 is bk/ip → ERR{code:seq_blocked}
```

### Etag optimistic locking

```
ticket_get id=myapp-001       → ...ua:2024-01-15T10:05:22.123456789Z...
ticket_update id=myapp-001 etag=2024-01-15T10:05:22.123456789Z s=dn
# If another agent updated first → ERR{code:conflict} → re-read and retry
```

### Concurrent-agent coordination

Multiple agents working from the same backlog compete safely via atomic claim:

```
# Agent A                           # Agent B
ticket_backlog                      ticket_backlog
→ [myapp-003, myapp-004, ...]       → [myapp-003, myapp-004, ...]

ticket_claim id=myapp-003           ticket_claim id=myapp-003
→ T{s:ip,...}  ✓                    → ERR{code:conflict}  ← already claimed
                                    ticket_claim id=myapp-004  ← next item
                                    → T{s:ip,...}  ✓
```

---

## Architecture

```
HTTP request
    │
    ▼
internal/mcp/server.go        — HTTP+SSE server, bearer auth, /health, /skill
    │
    ▼
internal/mcp/tools.go         — 13 MCP tool handlers (mark3labs/mcp-go)
    │  parses args, calls service
    ▼
internal/ticket/service.go    — thin business logic wrapper, backup goroutine
    │
    ▼
internal/ticket/store.go      — SQLite CRUD, FTS5 search, etag CAS, soft delete
    │  modernc.org/sqlite (no CGO)
    ▼
SQLite WAL database           — single writer, concurrent reads, FTS5 content table

Response path:
    store → service → tools → internal/toon/encoder.go → TOON/1 string → HTTP
```

**Package responsibilities:**

| Package             | Responsibility                                                     |
| ------------------- | ------------------------------------------------------------------ |
| `cmd/server/`       | Entrypoint, DB init, SQL migrations via `go:embed`, graceful shutdown |
| `internal/ticket/`  | Domain types, SQLite store (CRUD + FTS5), service layer, backup goroutine |
| `internal/toon/`    | TOON encoder, Mermaid diagram generator                            |
| `internal/mcp/`     | MCP server, all 13 tool handler registrations                      |

**Key design decisions:**

- **SQLite WAL** — `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;` with `SetMaxOpenConns(1)` for single-writer guarantee
- **Etag = updated_at** — optimistic locking without a dedicated column; `ticket_update` appends `WHERE updated_at = ?`
- **Soft delete** — `archived_at DATETIME NULL`; all queries default to `AND archived_at IS NULL`
- **Sequential ordering** — `UNIQUE(parent_id, exec_order)` at DB level; `ticket_claim` checks all lower `exec_order` siblings have `status = 'dn'` in a transaction
- **PROJECT_ID scoping** — all SQL queries append `AND project_id = ?`; ticket IDs embed the project slug (`myapp-001`) for unambiguous cross-project references

---

## Adding a New Tool

Follow these four steps. Example: adding `ticket_reopen`.

**Step 1 — Add domain logic to `internal/ticket/store.go`**

```go
func (s *Store) Reopen(ctx context.Context, projectID, id string) (*Ticket, error) {
    res, err := s.db.ExecContext(ctx,
        `UPDATE tickets SET status='bk', updated_at=? WHERE id=? AND project_id=? AND archived_at IS NULL`,
        time.Now().UTC(), id, projectID)
    if err != nil {
        return nil, err
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return nil, ErrNotFound
    }
    return s.Get(ctx, projectID, id)
}
```

**Step 2 — Expose it via `internal/ticket/service.go`**

```go
func (s *Service) Reopen(ctx context.Context, id string) (*Ticket, error) {
    return s.store.Reopen(ctx, s.projectID, id)
}
```

**Step 3 — Register the MCP tool in `internal/mcp/tools.go`**

```go
func toolTicketReopen(svc *ticket.Service) (mcp.Tool, mcp.HandlerFunc) {
    tool := mcp.NewTool("ticket_reopen",
        mcp.WithDescription("Move a done or cancelled ticket back to backlog."),
        mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
    )
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        id := req.Params.Arguments["id"].(string)
        t, err := svc.Reopen(ctx, id)
        if err != nil {
            return mcp.NewToolResultText(toon.EncodeError(toon.TicketErrCode(err), err.Error())), nil
        }
        return mcp.NewToolResultText(toon.Encode(t)), nil
    }
    return tool, handler
}
```

Then register it in `NewServer` alongside the other tools:

```go
s.mcpServer.AddTool(toolTicketReopen(svc))
```

**Step 4 — Write a test in `internal/mcp/tools_test.go`**

```go
func TestTicketReopen(t *testing.T) {
    svc := setupTestService(t)
    // create → claim → update done → reopen → verify status=bk
}
```

```bash
go test ./...
```

---

## Multi-Project Setup

Run one container per project on separate ports:

```yaml
# docker-compose.yml
services:
    orkestra-auth:
        build: .
        ports: ['8080:8080']
        volumes: [orkestra-auth:/data]
        environment: { PROJECT_ID: auth }

    orkestra-payments:
        build: .
        ports: ['8081:8080']
        volumes: [orkestra-payments:/data]
        environment: { PROJECT_ID: payments }
```

```bash
claude mcp add orkestra-auth     --transport http http://localhost:8080/sse
claude mcp add orkestra-payments --transport http http://localhost:8081/sse
```

The LLM sees two namespaced tool sets (`orkestra-auth__ticket_create`, `orkestra-payments__ticket_backlog`) and ticket IDs self-identify their project (`auth-001`, `payments-042`).

---

## Data Safety

Three layers:

1. **WAL mode** — `PRAGMA journal_mode=WAL` prevents corruption under concurrent reads
2. **Periodic backup** — background goroutine runs `VACUUM INTO` every `BACKUP_INTERVAL` (default 1h); keeps last `BACKUP_KEEP` (default 24) backups in `BACKUP_DIR`
3. **Docker named volume** — survives `docker compose down`; destroyed only by `docker compose down -v`

---

## Testing

```bash
# Run all tests
go test ./...

# With coverage report
go test -cover ./...

# Verbose output for a specific package
go test -v ./internal/ticket/...
go test -v ./internal/toon/...
```

**24 tests across 4 packages (~200ms):**

| Package            | Tests | What it covers                                          |
| ------------------ | ----- | ------------------------------------------------------- |
| `internal/ticket`  | 9     | CRUD, claim CAS, sequential enforcement, etag, FTS5     |
| `internal/toon`    | 15    | Encoding, special chars, board format, error envelopes  |
| `internal/mcp`     | 6     | Tool registration, TOON encoding, conflict, seq_blocked |
| `cmd/server`       | 3     | HTTP /health, /skill endpoint, bearer auth              |

---

## Building

```bash
# Local binary
go build -o orkestra ./cmd/server

# Docker (production — scratch image, ~20 MB)
docker build -t orkestra .

# Docker (debug — alpine with shell + sqlite CLI)
docker build --target debug -t orkestra:debug .
```

**Development loop:**

```bash
# Tail container logs
docker compose logs -f

# Check health + backup status
curl http://localhost:8080/health | jq .

# Inspect skill document served by the running server
curl http://localhost:8080/skill
```
