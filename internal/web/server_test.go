// SPDX-License-Identifier: MIT

package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/web"
)

const testProject = "webtest"

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	svc := testutil.NewTestService(t, testProject)
	return web.New(svc, testProject)
}

func newHandlerWithSvc(t *testing.T) (*ticket.Service, http.Handler) {
	t.Helper()
	svc := testutil.NewTestService(t, testProject)
	return svc, web.New(svc, testProject)
}

func TestGetProject(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/project", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["project_id"] != testProject {
		t.Errorf("project_id = %q, want %q", body["project_id"], testProject)
	}
}

func TestGetTicketsEmpty(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/tickets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Must decode as a JSON array (not null)
	var tickets []json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&tickets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected empty array, got %d items", len(tickets))
	}
}

func TestGetTicketsWithData(t *testing.T) {
	svc, h := newHandlerWithSvc(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, ticket.CreateInput{Title: "ticket one", Type: ticket.TypeTask})
	if err != nil {
		t.Fatalf("create ticket 1: %v", err)
	}
	_, err = svc.Create(ctx, ticket.CreateInput{Title: "ticket two", Type: ticket.TypeBug})
	if err != nil {
		t.Fatalf("create ticket 2: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/tickets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var tickets []ticket.Ticket
	if err := json.NewDecoder(rec.Body).Decode(&tickets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}
}

func TestGetTicketByID(t *testing.T) {
	svc, h := newHandlerWithSvc(t)
	ctx := context.Background()

	tk, err := svc.Create(ctx, ticket.CreateInput{Title: "single ticket", Type: ticket.TypeFeature})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/tickets/"+tk.ID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got ticket.Ticket
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != tk.ID {
		t.Errorf("ID = %q, want %q", got.ID, tk.ID)
	}
	if got.Title != "single ticket" {
		t.Errorf("Title = %q, want %q", got.Title, "single ticket")
	}
}

func TestGetTicketNotFound(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/tickets/nonexistent-999", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "not_found" {
		t.Errorf("error = %q, want not_found", body["error"])
	}
}

func TestGetIndex(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestGetIndexHTMLContent(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()

	// Task 1 a11y markers
	for _, want := range []string{
		`role="dialog"`,
		`--color-focus-ring`,
		`aria-live="polite"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response body missing %q", want)
		}
	}

	// P1 skip-link target
	if !strings.Contains(body, `id="board"`) {
		t.Errorf("response body missing id=\"board\"")
	}

	// P2 toast role
	if !strings.Contains(body, `role="alert"`) {
		t.Errorf("response body missing role=\"alert\"")
	}

	// P4 security headers
	securityHeaders := map[string]string{
		"Content-Security-Policy": "'unsafe-eval'",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	}
	for header, wantContains := range securityHeaders {
		got := rec.Header().Get(header)
		if !strings.Contains(got, wantContains) {
			t.Errorf("header %q = %q, want it to contain %q", header, got, wantContains)
		}
	}
}
