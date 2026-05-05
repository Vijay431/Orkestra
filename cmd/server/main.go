// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	stdlog "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	orkmcp "github.com/vijay431/orkestra/internal/mcp"
	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/web"
)

//go:embed 001_init.sql
var initSQL string

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck(os.Args[2:]))
	}

	log := newLogger(getenv("LOG_LEVEL", "info"))

	cfg := orkmcp.Config{
		ProjectID:  mustEnv("PROJECT_ID", log),
		Port:       getenv("PORT", "8080"),
		BindAddr:   getenv("BIND_ADDR", "0.0.0.0"),
		MCPToken:   os.Getenv("MCP_TOKEN"),
		BackupDir:  getenv("BACKUP_DIR", "backups"),
		BackupKeep: getenvInt("BACKUP_KEEP", 24),
	}
	cfg.BackupInterval = getenvDuration("BACKUP_INTERVAL", time.Hour)

	dbPath := getenv("DB_PATH", "orkestra.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		log.Error("failed to open database", "path", dbPath, "err", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1) // SQLite: single writer

	if err := ticket.RunMigrations(db, initSQL); err != nil {
		log.Error("migration failed", "err", err)
		os.Exit(1)
	}
	log.Info("database ready", "path", dbPath)

	svc := ticket.NewService(db, cfg.ProjectID, log)

	// Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Backup goroutine
	go svc.RunBackupLoop(ctx, dbPath, cfg.BackupDir, cfg.BackupInterval, cfg.BackupKeep)

	// Web UI server
	if os.Getenv("WEB_ENABLED") != "false" {
		webAddr := getenv("WEB_ADDR", "127.0.0.1:7777")
		h := web.New(svc, cfg.ProjectID)
		go func() {
			if err := web.Start(ctx, webAddr, h); err != nil {
				stdlog.Printf("web: server stopped: %v", err)
			}
		}()
		stdlog.Printf("web: listening on http://%s", webAddr)
	}

	// MCP server
	srv := orkmcp.NewServer(cfg, svc, log)
	if err := srv.Start(ctx); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}

	// Drain in-flight work then close DB
	log.Info("closing database")
	if err := db.Close(); err != nil {
		log.Warn("db close error", "err", err)
	}
	log.Info("shutdown complete")
}

func mustEnv(key string, log *slog.Logger) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error("required environment variable not set", "var", key)
		fmt.Fprintf(os.Stderr, "Error: %s is required\n", key)
		os.Exit(1)
	}
	return v
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func newLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func runHealthcheck(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: orkestra healthcheck <url>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, args[0], nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 2
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // best-effort drain

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: unhealthy (status %d)\n", resp.StatusCode)
		return 1
	}
	return 0
}
