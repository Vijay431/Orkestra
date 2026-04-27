package ticket

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store handles all SQLite operations scoped to a single project.
type Store struct {
	db        *sql.DB
	projectID string
}

func NewStore(db *sql.DB, projectID string) *Store {
	return &Store{db: db, projectID: projectID}
}

// RunMigrations applies the embedded SQL migration.
func RunMigrations(db *sql.DB, sql string) error {
	_, err := db.Exec(sql)
	return err
}

// nextID atomically increments the per-project sequence and returns the new ID.
func (s *Store) nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	var nextVal int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO ticket_seq(project_id, next_val) VALUES(?, 1)
		ON CONFLICT(project_id) DO UPDATE SET next_val = next_val + 1
		RETURNING next_val
	`, s.projectID).Scan(&nextVal)
	if err != nil {
		return "", fmt.Errorf("nextID: %w", err)
	}
	return fmt.Sprintf("%s-%03d", s.projectID, nextVal), nil
}

// Create inserts a new ticket into the backlog.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Ticket, error) {
	if in.Type == "" {
		in.Type = TypeTask
	}
	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	if in.ExecMode == "" {
		in.ExecMode = ExecModeParallel
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	id, err := s.nextID(ctx, tx)
	if err != nil {
		return nil, err
	}

	labelsJSON := encodeLabels(in.Labels)
	now := time.Now().UTC()

	var parentID *string
	if in.ParentID != "" {
		parentID = &in.ParentID
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO tickets
			(id, project_id, title, status, priority, type, description, assignee,
			 parent_id, labels, exec_mode, exec_order, created_at, updated_at)
		VALUES (?, ?, ?, 'bk', ?, ?, ?, '', ?, ?, ?, ?, ?, ?)
	`, id, s.projectID, in.Title, in.Priority, in.Type,
		in.Description, parentID, labelsJSON, in.ExecMode, in.ExecOrder,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: tickets.parent_id, tickets.exec_order") {
			return nil, fmt.Errorf("%w: exec_order must be unique within parent", ErrInvalid)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.Get(ctx, id)
}

// Get retrieves a ticket by ID (with comments, links, children).
func (s *Store) Get(ctx context.Context, id string) (*Ticket, error) {
	t, err := s.scanTicket(ctx, s.db, `
		SELECT id, project_id, title, status, priority, type, description, assignee,
		       COALESCE(parent_id,''), labels, exec_mode, exec_order, archived_at, created_at, updated_at
		FROM tickets WHERE id=? AND project_id=?
	`, id, s.projectID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}

	if err := s.loadRelations(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update applies field changes with optimistic locking via etag (updated_at).
func (s *Store) Update(ctx context.Context, in UpdateInput) (*Ticket, error) {
	now := time.Now().UTC()
	sets := []string{"updated_at=?"}
	args := []any{now.Format(time.RFC3339Nano)}

	if in.Title != nil {
		sets = append(sets, "title=?")
		args = append(args, *in.Title)
	}
	if in.Status != nil {
		sets = append(sets, "status=?")
		args = append(args, *in.Status)
	}
	if in.Priority != nil {
		sets = append(sets, "priority=?")
		args = append(args, *in.Priority)
	}
	if in.Type != nil {
		sets = append(sets, "type=?")
		args = append(args, *in.Type)
	}
	if in.Description != nil {
		sets = append(sets, "description=?")
		args = append(args, *in.Description)
	}
	if in.Assignee != nil {
		sets = append(sets, "assignee=?")
		args = append(args, *in.Assignee)
	}
	if in.Labels != nil {
		sets = append(sets, "labels=?")
		args = append(args, encodeLabels(in.Labels))
	}
	if in.ExecMode != nil {
		sets = append(sets, "exec_mode=?")
		args = append(args, *in.ExecMode)
	}
	if in.ExecOrder != nil {
		sets = append(sets, "exec_order=?")
		args = append(args, *in.ExecOrder)
	}

	args = append(args, in.ID, s.projectID)
	query := fmt.Sprintf("UPDATE tickets SET %s WHERE id=? AND project_id=? AND archived_at IS NULL",
		strings.Join(sets, ","))

	if in.Etag != "" {
		query += " AND updated_at=?"
		args = append(args, in.Etag)
	}

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: tickets.parent_id, tickets.exec_order") {
			return nil, fmt.Errorf("%w: exec_order must be unique within parent", ErrInvalid)
		}
		return nil, err
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		// Distinguish not-found from etag conflict
		exists := false
		_ = s.db.QueryRowContext(ctx, `SELECT 1 FROM tickets WHERE id=? AND project_id=? AND archived_at IS NULL`,
			in.ID, s.projectID).Scan(&exists)
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, in.ID)
		}
		return nil, fmt.Errorf("%w: etag mismatch for %s", ErrConflict, in.ID)
	}

	return s.Get(ctx, in.ID)
}

// Claim atomically moves a ticket to in_progress if it is not already claimed/done/cancelled.
// Enforces sequential ordering: if the ticket's parent is exec_mode=seq and this ticket's
// exec_order > 1, the previous sibling must be done.
func (s *Store) Claim(ctx context.Context, id string) (*Ticket, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	// Fetch current ticket details for sequential check
	var parentID sql.NullString
	var execMode string
	var execOrder sql.NullInt64
	var currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(parent_id,''), exec_mode, exec_order, status
		FROM tickets WHERE id=? AND project_id=? AND archived_at IS NULL
	`, id, s.projectID).Scan(&parentID, &execMode, &execOrder, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}

	// Sequential enforcement
	if parentID.Valid && parentID.String != "" && execMode == string(ExecModeSequential) && execOrder.Valid && execOrder.Int64 > 1 {
		var prevStatus string
		err := tx.QueryRowContext(ctx, `
			SELECT status FROM tickets
			WHERE parent_id=? AND exec_order=? AND project_id=? AND archived_at IS NULL
		`, parentID.String, execOrder.Int64-1, s.projectID).Scan(&prevStatus)
		if err != nil || prevStatus != string(StatusDone) {
			return nil, fmt.Errorf("%w: ord=%d predecessor not done", ErrSeqBlocked, execOrder.Int64)
		}
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE tickets SET status='ip', updated_at=?
		WHERE id=? AND project_id=? AND archived_at IS NULL
		  AND status NOT IN ('ip','dn','cl')
	`, now.Format(time.RFC3339Nano), id, s.projectID)
	if err != nil {
		return nil, err
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("%w: %s already claimed or terminal", ErrConflict, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Archive soft-deletes a ticket by setting archived_at.
func (s *Store) Archive(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE tickets SET archived_at=?, updated_at=?
		WHERE id=? AND project_id=? AND archived_at IS NULL
	`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, s.projectID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return nil
}

// List returns tickets matching the filter.
func (s *Store) List(ctx context.Context, f ListFilter) ([]Ticket, error) {
	where := []string{"project_id=?"}
	args := []any{s.projectID}

	if !f.IncludeArchived {
		where = append(where, "archived_at IS NULL")
	}
	if f.Status != nil {
		where = append(where, "status=?")
		args = append(args, *f.Status)
	}
	if f.Priority != nil {
		where = append(where, "priority=?")
		args = append(args, *f.Priority)
	}
	if f.Type != nil {
		where = append(where, "type=?")
		args = append(args, *f.Type)
	}

	limit := 50
	if f.Limit > 0 && f.Limit < 200 {
		limit = f.Limit
	}

	q := fmt.Sprintf(`SELECT id, project_id, title, status, priority, type, description, assignee,
		COALESCE(parent_id,''), labels, exec_mode, exec_order, archived_at, created_at, updated_at
		FROM tickets WHERE %s ORDER BY created_at DESC LIMIT %d`,
		strings.Join(where, " AND "), limit)

	return s.scanTickets(ctx, q, args...)
}

// Backlog returns backlog tickets ordered by priority (cr > h > m > l).
func (s *Store) Backlog(ctx context.Context, f ListFilter) ([]Ticket, error) {
	f.Status = ptr(StatusBacklog)
	f.IncludeArchived = false

	where := []string{"project_id=?", "status='bk'", "archived_at IS NULL"}
	args := []any{s.projectID}
	if f.Priority != nil {
		where = append(where, "priority=?")
		args = append(args, *f.Priority)
	}
	if f.Type != nil {
		where = append(where, "type=?")
		args = append(args, *f.Type)
	}

	limit := 50
	if f.Limit > 0 {
		limit = f.Limit
	}

	q := fmt.Sprintf(`SELECT id, project_id, title, status, priority, type, description, assignee,
		COALESCE(parent_id,''), labels, exec_mode, exec_order, archived_at, created_at, updated_at
		FROM tickets WHERE %s
		ORDER BY CASE priority WHEN 'cr' THEN 0 WHEN 'h' THEN 1 WHEN 'm' THEN 2 ELSE 3 END, created_at ASC
		LIMIT %d`, strings.Join(where, " AND "), limit)

	return s.scanTickets(ctx, q, args...)
}

// Board returns tickets grouped by status.
func (s *Store) Board(ctx context.Context, f ListFilter) (map[string][]Ticket, error) {
	where := []string{"project_id=?", "archived_at IS NULL"}
	args := []any{s.projectID}
	if f.Type != nil {
		where = append(where, "type=?")
		args = append(args, *f.Type)
	}

	q := fmt.Sprintf(`SELECT id, project_id, title, status, priority, type, description, assignee,
		COALESCE(parent_id,''), labels, exec_mode, exec_order, archived_at, created_at, updated_at
		FROM tickets WHERE %s ORDER BY status, created_at DESC`, strings.Join(where, " AND "))

	tickets, err := s.scanTickets(ctx, q, args...)
	if err != nil {
		return nil, err
	}

	board := make(map[string][]Ticket)
	for _, t := range tickets {
		board[string(t.Status)] = append(board[string(t.Status)], t)
	}
	return board, nil
}

// Children returns direct children of a ticket, sorted by exec_order.
func (s *Store) Children(ctx context.Context, parentID string) ([]Ticket, error) {
	return s.scanTickets(ctx, `
		SELECT id, project_id, title, status, priority, type, description, assignee,
		       COALESCE(parent_id,''), labels, exec_mode, exec_order, archived_at, created_at, updated_at
		FROM tickets WHERE parent_id=? AND project_id=? AND archived_at IS NULL
		ORDER BY COALESCE(exec_order, 999999), created_at ASC
	`, parentID, s.projectID)
}

// ChildrenDeep recursively collects children up to maxDepth.
func (s *Store) ChildrenDeep(ctx context.Context, parentID string, maxDepth int) ([]Ticket, error) {
	return s.collectChildren(ctx, parentID, 0, maxDepth)
}

func (s *Store) collectChildren(ctx context.Context, parentID string, depth, maxDepth int) ([]Ticket, error) {
	if depth >= maxDepth {
		return nil, nil
	}
	children, err := s.Children(ctx, parentID)
	if err != nil {
		return nil, err
	}
	result := make([]Ticket, 0, len(children))
	for _, c := range children {
		result = append(result, c)
		grandchildren, err := s.collectChildren(ctx, c.ID, depth+1, maxDepth)
		if err != nil {
			return nil, err
		}
		result = append(result, grandchildren...)
	}
	return result, nil
}

// AddComment adds a comment and returns the updated ticket.
func (s *Store) AddComment(ctx context.Context, ticketID, author, body string) (*Ticket, error) {
	// Verify ticket exists and belongs to this project
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM tickets WHERE id=? AND project_id=? AND archived_at IS NULL`,
		ticketID, s.projectID).Scan(&exists)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, ticketID)
	}

	if author == "" {
		author = "llm"
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO comments(ticket_id, author, body) VALUES(?,?,?)
	`, ticketID, author, body)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, ticketID)
}

// AddLink creates a directional link between two tickets.
func (s *Store) AddLink(ctx context.Context, fromID, toID string, lt LinkType) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO links(from_id, to_id, link_type) VALUES(?,?,?)
	`, fromID, toID, lt)
	return err
}

// Search performs full-text search using FTS5.
func (s *Store) Search(ctx context.Context, query string, includeArchived bool) ([]Ticket, error) {
	archiveFilter := "AND t.archived_at IS NULL"
	if includeArchived {
		archiveFilter = ""
	}
	q := fmt.Sprintf(`
		SELECT t.id, t.project_id, t.title, t.status, t.priority, t.type, t.description, t.assignee,
		       COALESCE(t.parent_id,''), t.labels, t.exec_mode, t.exec_order, t.archived_at, t.created_at, t.updated_at
		FROM tickets t
		JOIN tickets_fts f ON t.rowid = f.rowid
		WHERE f.tickets_fts MATCH ? AND t.project_id=? %s
		ORDER BY rank LIMIT 50`, archiveFilter)

	return s.scanTickets(ctx, q, query, s.projectID)
}

// Ping verifies the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Backup creates a copy of the database file using VACUUM INTO.
func (s *Store) Backup(ctx context.Context, destPath string) error {
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", destPath)
	return err
}

// --- helpers ---

func (s *Store) scanTicket(ctx context.Context, q interface{ QueryRowContext(context.Context, string, ...any) *sql.Row }, query string, args ...any) (*Ticket, error) {
	row := q.QueryRowContext(ctx, query, args...)
	return scanRow(row)
}

func (s *Store) scanTickets(ctx context.Context, query string, args ...any) ([]Ticket, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		t, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, *t)
	}
	return tickets, rows.Err()
}

type rowScanner interface {
	Scan(...any) error
}

func scanRow(row rowScanner) (*Ticket, error) {
	var t Ticket
	var labelsJSON sql.NullString
	var execOrder sql.NullInt64
	var archivedAt sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&t.ID, &t.ProjectID, &t.Title, &t.Status, &t.Priority, &t.Type,
		&t.Description, &t.Assignee, &t.ParentID, &labelsJSON,
		&t.ExecMode, &execOrder, &archivedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if labelsJSON.Valid && labelsJSON.String != "" {
		_ = json.Unmarshal([]byte(labelsJSON.String), &t.Labels)
	}
	if execOrder.Valid {
		v := int(execOrder.Int64)
		t.ExecOrder = &v
	}
	if archivedAt.Valid && archivedAt.String != "" {
		ts, _ := time.Parse(time.RFC3339Nano, archivedAt.String)
		t.ArchivedAt = &ts
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return &t, nil
}

func (s *Store) loadRelations(ctx context.Context, t *Ticket) error {
	// Children IDs
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM tickets WHERE parent_id=? AND project_id=? AND archived_at IS NULL
		ORDER BY COALESCE(exec_order, 999999), created_at ASC
	`, t.ID, s.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		t.Children = append(t.Children, id)
	}
	rows.Close()

	// Comments
	crows, err := s.db.QueryContext(ctx, `
		SELECT id, ticket_id, author, body, created_at
		FROM comments WHERE ticket_id=? ORDER BY created_at ASC
	`, t.ID)
	if err != nil {
		return err
	}
	defer crows.Close()
	for crows.Next() {
		var c Comment
		var ts string
		if err := crows.Scan(&c.ID, &c.TicketID, &c.Author, &c.Body, &ts); err != nil {
			return err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		t.Comments = append(t.Comments, c)
	}
	crows.Close()

	// Links
	lrows, err := s.db.QueryContext(ctx, `
		SELECT from_id, to_id, link_type FROM links WHERE from_id=? OR to_id=?
	`, t.ID, t.ID)
	if err != nil {
		return err
	}
	defer lrows.Close()
	for lrows.Next() {
		var l Link
		if err := lrows.Scan(&l.FromID, &l.ToID, &l.LinkType); err != nil {
			return err
		}
		t.Links = append(t.Links, l)
	}
	return lrows.Err()
}

func encodeLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	b, _ := json.Marshal(labels)
	return string(b)
}

func ptr[T any](v T) *T { return &v }
