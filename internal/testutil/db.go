// SPDX-License-Identifier: MIT

package testutil

import (
	"database/sql"
	"io"
	"log/slog"
	"net"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/vijay431/orkestra/internal/ticket"
)

// Schema is the full migration SQL, kept in sync with migrations/001_init.sql.
const Schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS ticket_seq (
    project_id TEXT PRIMARY KEY,
    next_val   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tickets (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    title       TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'bk',
    priority    TEXT NOT NULL DEFAULT 'm',
    type        TEXT NOT NULL DEFAULT 'tsk',
    description TEXT,
    assignee    TEXT,
    parent_id   TEXT REFERENCES tickets(id),
    labels      TEXT,
    exec_mode   TEXT NOT NULL DEFAULT 'par',
    exec_order  INTEGER,
    archived_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(parent_id, exec_order)
);

CREATE TABLE IF NOT EXISTS comments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id  TEXT    NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author     TEXT    NOT NULL DEFAULT 'llm',
    body       TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS links (
    from_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    to_id     TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    link_type TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id, link_type)
);

CREATE INDEX IF NOT EXISTS idx_tickets_project  ON tickets(project_id);
CREATE INDEX IF NOT EXISTS idx_tickets_archived ON tickets(archived_at);
CREATE INDEX IF NOT EXISTS idx_tickets_parent   ON tickets(parent_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status   ON tickets(project_id, status, archived_at);

CREATE VIRTUAL TABLE IF NOT EXISTS tickets_fts USING fts5(
    id, title, description, labels,
    content=tickets, content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS tickets_ai AFTER INSERT ON tickets BEGIN
    INSERT INTO tickets_fts(rowid, id, title, description, labels)
    VALUES (new.rowid, new.id, new.title, new.description, new.labels);
END;

CREATE TRIGGER IF NOT EXISTS tickets_au AFTER UPDATE ON tickets BEGIN
    INSERT INTO tickets_fts(tickets_fts, rowid, id, title, description, labels)
    VALUES ('delete', old.rowid, old.id, old.title, old.description, old.labels);
    INSERT INTO tickets_fts(rowid, id, title, description, labels)
    VALUES (new.rowid, new.id, new.title, new.description, new.labels);
END;

CREATE TRIGGER IF NOT EXISTS tickets_ad AFTER DELETE ON tickets BEGIN
    INSERT INTO tickets_fts(tickets_fts, rowid, id, title, description, labels)
    VALUES ('delete', old.rowid, old.id, old.title, old.description, old.labels);
END;
`

// NewTestDB opens an in-memory SQLite DB with migrations applied.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// NewTestStore returns a Store backed by a fresh in-memory DB.
func NewTestStore(t *testing.T, projectID string) *ticket.Store {
	t.Helper()
	return ticket.NewStore(NewTestDB(t), projectID)
}

// NewTestService returns a Service backed by a fresh in-memory DB.
func NewTestService(t *testing.T, projectID string) *ticket.Service {
	t.Helper()
	return ticket.NewService(NewTestDB(t), projectID, NopLogger())
}

// NopLogger returns a logger that discards all output.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// FreePort returns a free TCP port on localhost.
func FreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
