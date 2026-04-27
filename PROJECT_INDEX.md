# Project Index: Orkestra

Generated: 2026-04-27 · Go 1.26.2 · ~3.3K LOC (Go)

Self-hosted MCP ticket server. Exposes 13 MCP tools to LLM agents over stdio/HTTP, backed by SQLite (WAL + FTS5). Uses TOON encoding for ~60-70% token reduction vs JSON.

## Project Structure

```
cmd/server/         entrypoint, env config, DB init, migrations (go:embed)
internal/ticket/    domain types, SQLite store, service layer, backup loop
internal/toon/      TOON encoder, Mermaid diagram generator, error schema
internal/mcp/       MCP server bootstrap, 13 tool registrations + handlers
internal/testutil/  shared test DB helper
test/e2e/           end-to-end MCP tests
migrations/         001_init.sql (embedded)
skill/              SKILL.md operator guide + references/
docs/               quickstart, architecture, workflows, toon, data-safety
```

## Entry Points

- `cmd/server/main.go:26` — `main()`: reads env (PROJECT_ID required), opens SQLite, runs migrations, starts MCP server, optional backup loop
- `internal/mcp/server.go:43` — `NewServer(cfg, svc, log)`: builds MCP server (stdio or HTTP)
- `internal/mcp/server.go:73` — `Start(ctx)`: dispatches stdio vs HTTP transport
- `internal/mcp/tools.go:18` — `RegisterTools(s, svc)`: wires all 13 ticket_* MCP tools

## Core Modules

### internal/ticket — domain + persistence
- `types.go` — `Ticket`, `Comment`, `Link`, `CreateInput`, `UpdateInput`, `ListFilter`; enums `Status`, `Priority`, `TicketType`, `ExecMode`, `LinkType`
- `store.go` (578 lines) — SQLite CRUD: `Create`, `Get`, `Update` (etag CAS), `Claim` (atomic CAS), `Archive`, `List`, `Backlog`, `Board`, `Children`/`ChildrenDeep`, `AddComment`, `AddLink`, `Search` (FTS5), `Backup`
- `service.go` — thin wrapper over `Store` + `RunBackupLoop` / `pruneBackups` / `LastBackup`

### internal/toon — encoding + diagrams
- `encoder.go` — `Encode`, `EncodeSummary`, `EncodeBoard`, `EncodeError`, `EncodeOK`, `EtagOf`, `TicketErrCode`
- `diagram.go` — `GenerateDiagram(root, ChildLoader, maxDepth)` → Mermaid graph
- `schema.go` — `ErrCode` constants

### internal/mcp — MCP transport + tools
- `server.go` — stdio + HTTP transports, `/health`, `/skill`, optional bearer-token auth middleware
- `tools.go` — 13 tools: `ticket_create`, `_get`, `_claim`, `_update`, `_archive`, `_list`, `_comment`, `_link`, `_search`, `_children`, `_backlog`, `_board`, `_diagram`

## Key Design Decisions

- **PROJECT_ID** mandatory env var; every query scoped to it
- **Etag = updated_at** for optimistic concurrency on `ticket_update`
- **ticket_claim** atomic CAS; enforces sequential ordering when `exec_mode=seq`
- **Soft delete** via `archived_at`
- **TOON output** for all tool responses (compact, LLM-friendly)
- **modernc.org/sqlite** (pure-Go, no CGO)

## Configuration

- `go.mod` — module `github.com/vijay431/orkestra`; deps: `mark3labs/mcp-go`, `modernc.org/sqlite`
- `Dockerfile` + `docker-compose.yml` — container deploy
- `Makefile` — build/test shortcuts
- `migrations/001_init.sql` — schema (embedded via `go:embed` in `cmd/server/main.go`)
- `.github/workflows/ci.yml` — CI
- `.github/dependabot.yml` — dep updates

### Env vars (cmd/server/main.go)
- `PROJECT_ID` (required) · `DB_PATH` · `TRANSPORT` (stdio|http) · `HTTP_ADDR` · `AUTH_TOKEN` · `BACKUP_DIR` · `BACKUP_INTERVAL` · `BACKUP_KEEP` · `LOG_LEVEL`

## Documentation

- `README.md` — overview, install, usage
- `CLAUDE.md` — project context for Claude Code
- `skill/SKILL.md` — LLM operator guide (tool selection, TOON, workflows)
- `skill/references/api-guide.md` — full tool parameter reference
- `skill/references/examples.md`, `troubleshooting.md`
- `docs/quickstart.md`, `architecture.md`, `workflows.md`, `toon.md`, `data-safety.md`
- `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `ACKNOWLEDGMENTS.md`, `LICENSE`

## Tests

- `internal/ticket/store_test.go` (201) · `service_test.go` (108)
- `internal/mcp/tools_test.go` (375)
- `internal/toon/encoder_test.go` (226) · `diagram_test.go` (152)
- `cmd/server/integration_test.go` (172)
- `test/e2e/mcp_e2e_test.go` + `helpers_test.go`
- `internal/testutil/db.go` — shared in-memory SQLite helper

## Quick Start

```bash
PROJECT_ID=myapp go run ./cmd/server      # local stdio
go test ./...                              # full suite
PROJECT_ID=myapp docker compose up -d      # docker
```

## Key Dependencies

- `github.com/mark3labs/mcp-go` v0.49.0 — MCP server framework
- `modernc.org/sqlite` v1.50.0 — pure-Go SQLite (FTS5 enabled)
