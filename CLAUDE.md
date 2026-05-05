# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Orkestra — Project Context

Orkestra is a self-hosted MCP ticket server written in Go. It exposes 13 MCP tools to LLM agents for creating, tracking, and resolving work tickets — all locally, with no cloud dependency or rate limits.

See `skill/SKILL.md` for the LLM operator guide (tool selection, TOON format, workflows). Full tool parameter reference in `skill/references/api-guide.md`.

## Architecture

- `cmd/server/` — entrypoint, DB init via `go:embed 001_init.sql`, graceful shutdown
- `internal/ticket/` — domain types (`types.go`), SQLite store with FTS5 (`store.go`), service with backup lifecycle (`service.go`)
- `internal/toon/` — TOON encoder (`encoder.go`), Mermaid diagram generator (`diagram.go`), schema constants (`schema.go`)
- `internal/mcp/` — MCP server (`server.go`), all 13 tool definitions + handlers (`tools.go`)
- `internal/web/` — embedded HTTP Kanban UI on port 7777 (go:embed)
- `internal/testutil/` — shared test helpers: `NewTestDB`, `NewTestStore`, `NewTestService`, `NopLogger`, `FreePort`

## Key Design Decisions

- **TOON format**: compact ticket notation, ~60-70% fewer tokens than JSON; every tool returns TOON/1-prefixed output
- **SQLite**: single-writer (`db.SetMaxOpenConns(1)`), WAL mode, FTS5 for full-text search, soft delete via `archived_at`
- **PROJECT_ID**: mandatory env var; all store queries are scoped to this project; ticket IDs are `{PROJECT_ID}-{seq}`
- **Etag = updated_at**: `UpdateInput.Etag` is `updated_at` as RFC3339Nano — must be passed for optimistic locking
- **ticket_claim**: atomic CAS; moves ticket to `ip` and enforces `exec_mode=seq` ordering (prior seq ticket must be `dn`/`cl`)
- **Schema sync**: `internal/testutil/db.go` `Schema` const must stay in sync with `migrations/001_init.sql` (used by in-memory test DBs)
- **MCP transport**: SSE over HTTP; the server exposes `/sse` and `/message` for MCP clients, plus public `/health` and `/skill` endpoints; `srv.Start(ctx)` in `main.go` blocks until shutdown
- **healthcheck subcommand**: `./orkestra healthcheck <url>` — exits 0 on HTTP 200, 1 on unhealthy, 2 on bad args; used by the Docker `HEALTHCHECK` directive
- **Module path**: `github.com/vijay431/orkestra` — use this for internal imports

## Getting Started

End users install Orkestra via `install.sh`, which starts the Docker container, registers the MCP server with all detected AI tools, and installs the agent skill globally.

**Remote (no clone needed):**
```bash
curl -fsSL https://raw.githubusercontent.com/Vijay431/Orkestra/main/install.sh | PROJECT_ID=myapp bash
```

**Local clone:**
```bash
git clone https://github.com/Vijay431/Orkestra && cd Orkestra
PROJECT_ID=myapp ./install.sh
```

The `go run` command below is for developing Orkestra itself, not for using it.

## Commands

```bash
# Run the server (dev mode — no Docker)
PROJECT_ID=myapp go run ./cmd/server

# Run all non-e2e tests
go test ./internal/... ./cmd/...
# or via make
make test

# Run a single test
go test ./internal/ticket/... -run TestStore_Create

# Test categories
make test-unit          # ticket + toon packages only
make test-integration   # mcp + cmd/server packages
make test-e2e           # requires -tags e2e build tag, 60s timeout
make test-all           # all of the above

# Coverage report
make cover              # writes coverage.html

# Docker
PROJECT_ID=myapp docker compose up -d
```

## Environment Variables

| Variable          | Default     | Notes                                      |
|-------------------|-------------|--------------------------------------------|
| `PROJECT_ID`      | (required)  | Scopes all tickets; part of ticket ID      |
| `PORT`            | `8080`      | MCP server port                            |
| `BIND_ADDR`       | `0.0.0.0`   |                                            |
| `MCP_TOKEN`       | (none)      | Bearer token for MCP auth                  |
| `DB_PATH`         | `orkestra.db` |                                          |
| `BACKUP_DIR`      | `backups`   |                                            |
| `BACKUP_INTERVAL` | `1h`        | Go duration string                         |
| `BACKUP_KEEP`     | `24`        | Number of backup files to retain           |
| `WEB_ENABLED`     | `true`      | Set to `false` to disable Kanban UI        |
| `WEB_ADDR`        | `127.0.0.1:7777` | Kanban UI bind address               |
| `LOG_LEVEL`       | `info`      | `debug`, `info`, `warn`, `error`           |

## The 13 MCP Tools

`ticket_create`, `ticket_get`, `ticket_claim`, `ticket_update`, `ticket_archive`, `ticket_list`, `ticket_comment`, `ticket_link`, `ticket_search`, `ticket_children`, `ticket_backlog`, `ticket_board`, `ticket_diagram`

All registered in `internal/mcp/tools.go:RegisterTools`.
