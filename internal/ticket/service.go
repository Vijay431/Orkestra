// SPDX-License-Identifier: MIT

package ticket

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Service wraps Store with backup lifecycle and convenience methods.
type Service struct {
	store     *Store
	projectID string
	log       *slog.Logger
}

func NewService(db *sql.DB, projectID string, log *slog.Logger) *Service {
	return &Service{
		store:     NewStore(db, projectID),
		projectID: projectID,
		log:       log,
	}
}

// Store exposes the underlying store for direct access from tests.
func (s *Service) Store() *Store { return s.store }

// Create creates a new ticket.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Ticket, error) {
	return s.store.Create(ctx, in)
}

// Get retrieves a ticket with all relations.
func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	return s.store.Get(ctx, id)
}

// Update applies fields with etag-based optimistic locking.
func (s *Service) Update(ctx context.Context, in UpdateInput) (*Ticket, error) {
	return s.store.Update(ctx, in)
}

// Claim atomically moves a ticket to in_progress.
func (s *Service) Claim(ctx context.Context, id string) (*Ticket, error) {
	return s.store.Claim(ctx, id)
}

// Archive soft-deletes a ticket.
func (s *Service) Archive(ctx context.Context, id string) error {
	return s.store.Archive(ctx, id)
}

// List returns tickets matching the filter.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Ticket, error) {
	return s.store.List(ctx, f)
}

// Backlog returns backlog tickets ordered by priority.
func (s *Service) Backlog(ctx context.Context, f ListFilter) ([]Ticket, error) {
	return s.store.Backlog(ctx, f)
}

// Board returns tickets grouped by status.
func (s *Service) Board(ctx context.Context, f ListFilter) (map[string][]Ticket, error) {
	return s.store.Board(ctx, f)
}

// Children returns direct children of a ticket.
func (s *Service) Children(ctx context.Context, id string) ([]Ticket, error) {
	return s.store.Children(ctx, id)
}

// ChildrenDeep returns the full subtree up to maxDepth levels.
func (s *Service) ChildrenDeep(ctx context.Context, id string, maxDepth int) ([]Ticket, error) {
	return s.store.ChildrenDeep(ctx, id, maxDepth)
}

// AddComment adds a comment to a ticket.
func (s *Service) AddComment(ctx context.Context, ticketID, author, body string) (*Ticket, error) {
	return s.store.AddComment(ctx, ticketID, author, body)
}

// AddLink creates a directional link.
func (s *Service) AddLink(ctx context.Context, fromID, toID string, lt LinkType) error {
	return s.store.AddLink(ctx, fromID, toID, lt)
}

// Search performs full-text search.
func (s *Service) Search(ctx context.Context, query string, includeArchived bool) ([]Ticket, error) {
	return s.store.Search(ctx, query, includeArchived)
}

// Ping checks DB health.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

// RunBackupLoop starts the periodic backup goroutine. It stops when ctx is cancelled.
func (s *Service) RunBackupLoop(ctx context.Context, dbPath, backupDir string, interval time.Duration, keepCount int) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		s.log.Error("failed to create backup dir", "err", err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doBackup(ctx, backupDir, keepCount)
		}
	}
}

func (s *Service) doBackup(ctx context.Context, backupDir string, keepCount int) {
	name := fmt.Sprintf("orkestra-%s.db", time.Now().UTC().Format("20060102T150405"))
	dest := filepath.Join(backupDir, name)

	if err := s.store.Backup(ctx, dest); err != nil {
		s.log.Error("backup failed", "dest", dest, "err", err)
		return
	}
	s.log.Info("backup created", "path", dest)
	s.pruneBackups(backupDir, keepCount)
}

func (s *Service) pruneBackups(backupDir string, keepCount int) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "orkestra-") && strings.HasSuffix(e.Name(), ".db") {
			files = append(files, filepath.Join(backupDir, e.Name()))
		}
	}
	sort.Strings(files) // chronological order (name includes timestamp)
	for len(files) > keepCount {
		if err := os.Remove(files[0]); err != nil {
			s.log.Warn("failed to remove old backup", "path", files[0], "err", err)
		}
		files = files[1:]
	}
}

// LastBackup returns the path and time of the most recent backup file, if any.
func (s *Service) LastBackup(backupDir string) (string, time.Time) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return "", time.Time{}
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "orkestra-") && strings.HasSuffix(e.Name(), ".db") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return "", time.Time{}
	}
	sort.Strings(files)
	last := files[len(files)-1]
	info, _ := os.Stat(filepath.Join(backupDir, last))
	if info == nil {
		return last, time.Time{}
	}
	return last, info.ModTime()
}
