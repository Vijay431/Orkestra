# Orkestra — Project Context

Orkestra is a self-hosted MCP ticket server written in Go. It exposes 13 MCP tools to LLM agents for creating, tracking, and resolving work tickets — all locally, with no cloud dependency or rate limits.

See `skill/SKILL.md` for the LLM operator guide (tool selection, TOON format, workflows). Full tool parameter reference in `skill/references/api-guide.md`.

## Architecture

- `cmd/server/` — entrypoint, DB init, migrations via `go:embed`
- `internal/ticket/` — domain types, store (SQLite CRUD + FTS5), service (business logic + backup)
- `internal/toon/` — TOON encoder, Mermaid diagram generator
- `internal/mcp/` — MCP server, all 13 tool handlers
- `internal/web/` — embedded HTTP server; read-only Kanban UI on port 7777 (go:embed)

## Key Design Decisions

- **TOON format**: compact ticket notation, ~60-70% fewer tokens than JSON
- **SQLite**: single-writer, WAL mode, FTS5 for search, soft delete via `archived_at`
- **PROJECT_ID**: mandatory env var; all queries are scoped to this project
- **Etag = updated_at**: optimistic locking for concurrent updates
- **ticket_claim**: atomic CAS; enforces sequential ordering when `exec_mode=seq`

## Running Locally

```bash
PROJECT_ID=myapp go run ./cmd/server
```

## Running Tests

```bash
go test ./...
```

## Docker

```bash
PROJECT_ID=myapp docker compose up -d
```
