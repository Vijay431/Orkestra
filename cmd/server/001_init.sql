PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
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
