---
layout: default
title: Data Safety
nav_order: 7
permalink: /data-safety
---

# 🛡️ Data Safety
{: .no_toc }

Your tickets, your disk, your rules.
{: .fs-5 .fw-300 }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Where Your Data Lives

```mermaid
flowchart LR
    A[Orkestra Server] -->|writes| B[(SQLite WAL<br/>DB_PATH)]
    A -->|VACUUM INTO| C[(Backups<br/>BACKUP_DIR)]
    B -.->|never leaves| D[Your machine]
    C -.->|never leaves| D
    style D fill:#9BE564
```

No telemetry. No phone-home. No cloud sync. The only network surface is the MCP HTTP listener you point your agents at.

---

## Backups

Orkestra runs a periodic `VACUUM INTO` to a timestamped file in `BACKUP_DIR`, keeping the last `BACKUP_KEEP` snapshots.

| Variable | Default | Purpose |
|----------|---------|---------|
| `BACKUP_DIR` | `/data/backups` | Where snapshots land |
| `BACKUP_KEEP` | `24` | How many to retain (oldest pruned) |
| `BACKUP_INTERVAL` | `1h` | How often to snapshot |

```mermaid
flowchart TD
    A[Tick: every BACKUP_INTERVAL] --> B[VACUUM INTO<br/>BACKUP_DIR/orkestra-TIMESTAMP.db]
    B --> C{Count<br/>> BACKUP_KEEP?}
    C -->|yes| D[Delete oldest]
    C -->|no| E[Done]
    D --> E
```

Restore is a file copy: stop the server, replace `DB_PATH` with the snapshot, start again.

---

## Durability

- **WAL mode** — concurrent reads while one writer commits
- **Single writer** — `SetMaxOpenConns(1)` prevents `SQLITE_BUSY` thrash
- **Soft delete** — `archived_at` instead of row deletion; nothing is truly gone unless you `VACUUM`

---

## 🧪 Testing

```bash
go test ./...                # unit + integration
go test -tags e2e ./test/e2e # end-to-end via real HTTP
go test -race ./...          # race detector
```

CI runs all three on every push to every branch — see [`.github/workflows/ci.yml`](https://github.com/Vijay431/Orkestra/blob/main/.github/workflows/ci.yml).

---

## 🐳 Building

```bash
# Local binary
go build -o orkestra ./cmd/server

# Docker (scratch image, ~20 MB)
docker build -t orkestra .
docker compose up -d
```

The image is `FROM scratch` — no shell, no package manager, nothing to exploit beyond the Go binary itself.
