// SPDX-License-Identifier: MIT

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/vijay431/orkestra/internal/ticket"
)

// Config holds all server configuration derived from env vars.
type Config struct {
	ProjectID      string
	Port           string
	BindAddr       string
	MCPToken       string
	BackupDir      string
	BackupInterval time.Duration
	BackupKeep     int
	LogLevel       slog.Level
}

// Server wraps the MCP SSE server with health, skill, and auth.
type Server struct {
	cfg     Config
	svc     *ticket.Service
	mcp     *server.MCPServer
	sse     *server.SSEServer
	log     *slog.Logger
	skillMD string
}

// NewServer constructs the Orkestra MCP server.
func NewServer(cfg Config, svc *ticket.Service, log *slog.Logger) *Server {
	mcpSrv := server.NewMCPServer(
		"Orkestra",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Orkestra is a local LLM-friendly ticket management server using TOON notation. Call ticket_backlog to see what to work on next. Use ticket_create to log new work. See TOON/1 encoded responses for all tool outputs."),
	)

	RegisterTools(mcpSrv, svc)

	baseURL := fmt.Sprintf("http://%s:%s", cfg.BindAddr, cfg.Port)
	sseSrv := server.NewSSEServer(mcpSrv,
		server.WithBaseURL(baseURL),
		server.WithKeepAlive(true),
		server.WithKeepAliveInterval(15*time.Second),
	)

	skillMD := loadSkillMD()

	return &Server{
		cfg:     cfg,
		svc:     svc,
		mcp:     mcpSrv,
		sse:     sseSrv,
		log:     log,
		skillMD: skillMD,
	}
}

// Start begins serving. It blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Public endpoints (no auth)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/skill", s.handleSkill)

	// MCP endpoints (auth-protected)
	mux.Handle("/sse", s.authMiddleware(s.sse.SSEHandler()))
	mux.Handle("/message", s.authMiddleware(s.sse.MessageHandler()))

	addr := fmt.Sprintf("%s:%s", s.cfg.BindAddr, s.cfg.Port)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE needs no write timeout
		IdleTimeout:  120 * time.Second,
	}

	s.log.Info("Orkestra started",
		"addr", addr,
		"project", s.cfg.ProjectID,
		"auth", s.cfg.MCPToken != "",
	)

	errCh := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutting down server")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutCtx)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := s.svc.Ping(r.Context()) == nil
	lastBackup, lastBackupTime := s.svc.LastBackup(s.cfg.BackupDir)

	resp := map[string]any{
		"status":           "ok",
		"project":          s.cfg.ProjectID,
		"db_ok":            dbOK,
		"last_backup":      lastBackup,
		"last_backup_time": "",
	}
	if !lastBackupTime.IsZero() {
		resp["last_backup_time"] = lastBackupTime.UTC().Format(time.RFC3339)
	}
	if !dbOK {
		resp["status"] = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

func (s *Server) handleSkill(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	fmt.Fprint(w, s.skillMD)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.MCPToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + s.cfg.MCPToken
		if auth != expected {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loadSkillMD reads ORKESTRA_SKILL.md from the same directory as the binary
// or from the working directory, falling back to an embedded minimal string.
func loadSkillMD() string {
	candidates := []string{
		"ORKESTRA_SKILL.md",
		filepath.Join("skill", "SKILL.md"),
		filepath.Join(filepath.Dir(os.Args[0]), "ORKESTRA_SKILL.md"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return minimalSkill
}

const minimalSkill = `# Orkestra — LLM Operator Guide

## TOON Quick Reference
T{id,t,s,p,typ,lbl,em,ord,par,ch,d,cmt,lnk,ca,ua}

Status:   bk=backlog td=todo ip=in_progress dn=done bl=blocked cl=cancelled
Priority: cr=critical h=high m=medium l=low
Type:     bug ft=feature tsk=task ep=epic chr=chore
ExecMode: par=parallel(default) seq=sequential

## Core Workflow
1. ticket_backlog  → see what needs doing
2. ticket_claim    → atomically claim a ticket (moves to ip)
3. (do the work)
4. ticket_update   → mark done (s=dn, supply etag)

## Epic Decomposition
ticket_create typ=ep t="Feature"
ticket_create parent_id=<epic> t="Sub-task" em=par
ticket_diagram id=<epic>  → visual Mermaid flowchart

## Error Codes
ERR{code:not_found}    ticket does not exist
ERR{code:conflict}     etag stale or ticket already claimed
ERR{code:seq_blocked}  sequential predecessor not done
ERR{code:invalid}      bad input (e.g. duplicate exec_order)
`
