// SPDX-License-Identifier: MIT

package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/vijay431/orkestra/internal/ticket"
)

//go:embed static/*
var staticFiles embed.FS

// New builds and returns the HTTP mux for the web UI and JSON API.
func New(svc *ticket.Service, projectID string) http.Handler {
	mux := http.NewServeMux()

	// Serve index.html at /
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(staticFiles, "static/index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data) //nolint:errcheck
	})

	// GET /api/project — project metadata
	mux.HandleFunc("GET /api/project", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"project_id": projectID})
	})

	// GET /api/tickets — list all non-archived tickets
	mux.HandleFunc("GET /api/tickets", func(w http.ResponseWriter, r *http.Request) {
		tickets, err := svc.List(r.Context(), ticket.ListFilter{
			IncludeArchived: false,
			Limit:           500,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// Return [] not null for empty slice
		if tickets == nil {
			tickets = []ticket.Ticket{}
		}
		writeJSON(w, http.StatusOK, tickets)
	})

	// GET /api/tickets/{id} — single ticket with relations
	mux.HandleFunc("GET /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		t, err := svc.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, ticket.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, t)
	})

	return mux
}

// Start starts the HTTP server on addr and blocks until ctx is cancelled,
// then gracefully shuts down.
func Start(ctx context.Context, addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: h,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("web server shutdown error: %v", err)
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}
