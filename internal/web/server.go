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
	"time"

	"github.com/vijay431/orkestra/internal/ticket"
)

//go:embed static/*
var staticFiles embed.FS

// New builds and returns the HTTP mux for the web UI and JSON API.
func New(svc *ticket.Service, projectID string) http.Handler {
	mux := http.NewServeMux()

	vendorFS, err := fs.Sub(staticFiles, "static/vendor")
	if err != nil {
		panic("web: static/vendor not embedded: " + err.Error())
	}
	mux.Handle("GET /vendor/", http.StripPrefix("/vendor/", http.FileServerFS(vendorFS)))

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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		_, _ = w.Write(data) // error unactionable; response header already sent
	})

	mux.HandleFunc("GET /api/project", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"project_id": projectID})
	})

	mux.HandleFunc("GET /api/tickets", func(w http.ResponseWriter, r *http.Request) {
		tickets, err := svc.List(r.Context(), ticket.ListFilter{
			IncludeArchived: false,
			Limit:           500,
		})
		if err != nil {
			log.Printf("web: handler error: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		// json.Encoder renders nil slices as null; use empty slice to return [] instead
		if tickets == nil {
			tickets = []ticket.Ticket{}
		}
		writeJSON(w, http.StatusOK, tickets)
	})

	mux.HandleFunc("GET /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		t, err := svc.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, ticket.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
				return
			}
			log.Printf("web: handler error: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
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

	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", addr)
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
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("web: shutdown error: %v", err)
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
