# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
### Changed
### Fixed

## [0.3.0] - 2026-04-29

### Added

- Read-only Kanban board web UI on port 7777 (`internal/web/`), togglable via `WEB_ENABLED` / `WEB_ADDR`
- Published docs site via GitHub Pages (Astro-based, migrated from Jekyll)
- CodeRabbit automated review integration

### Fixed

- `BACKUP_DIR` default changed from `/data/backups` to `backups/` so local (non-Docker) runs work out of the box
- `DB_PATH` default changed from `/data/orkestra.db` to `orkestra.db` for the same reason
- Web UI HTTP server now binds to `0.0.0.0` inside the Docker container so port mapping works correctly
- HTTP server shutdown timeout, error leakage, and comment style in web package
- Security issues identified by CodeRabbit in CI

### Changed

- CI pipeline updated with security hardening

## [0.2.0] - 2026-04-10

### Added

- HTTP server package (`internal/web/`) with JSON API backing the Kanban UI
- Web UI documentation and Docker configuration updates

## [0.1.0] - 2026-04-01

### Added

- `.gitignore` entry to exclude `.worktrees/` directory

## [1.0.0] - 2026-04-27

### Added

- 13 MCP tools covering the full ticket lifecycle: `ticket_create`, `ticket_get`, `ticket_claim`, `ticket_update`, `ticket_list`, `ticket_backlog`, `ticket_board`, `ticket_search`, `ticket_comment`, `ticket_link`, `ticket_children`, `ticket_diagram`, `ticket_archive`
- TOON/1 compact notation — ~60–70% fewer tokens than JSON responses
- SQLite backend with WAL mode, FTS5 full-text search, and soft delete via `archived_at`
- Periodic backup via `VACUUM INTO` with configurable retention (`BACKUP_KEEP`)
- Optimistic locking via `etag = updated_at` on all updates
- Atomic `ticket_claim` with CAS semantics; enforces sequential ordering when `exec_mode=seq`
- `install.sh` with auto-detection for Claude Code, Cursor, GitHub Copilot, Windsurf, Zed, and Continue.dev
- Docker production image (~20 MB, `scratch` base) and `docker-compose.yml` for local setup
- Full unit, integration, and E2E test suite
