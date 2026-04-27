// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	orkmcp "github.com/vijay431/orkestra/internal/mcp"
	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
)

func startTestServer(t *testing.T, projectID string) string {
	return startTestServerWithCfg(t, orkmcp.Config{
		ProjectID: projectID,
		BindAddr:  "127.0.0.1",
	})
}

func startTestServerWithCfg(t *testing.T, cfg orkmcp.Config) string {
	t.Helper()
	db := testutil.NewTestDB(t)
	log := testutil.NopLogger()
	svc := ticket.NewService(db, cfg.ProjectID, log)

	cfg.Port = fmt.Sprintf("%d", testutil.FreePort(t))
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}

	srv := orkmcp.NewServer(cfg, svc, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = srv.Start(ctx) }()

	base := fmt.Sprintf("http://127.0.0.1:%s", cfg.Port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server did not become healthy within 5s")
	return ""
}

func TestHealth(t *testing.T) {
	base := startTestServer(t, "inttest")

	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
	if body["project"] != "inttest" {
		t.Errorf("project = %v, want inttest", body["project"])
	}
	if body["db_ok"] != true {
		t.Errorf("db_ok = %v, want true", body["db_ok"])
	}
}

func TestSkillEndpoint(t *testing.T) {
	base := startTestServer(t, "inttest")

	resp, err := http.Get(base + "/skill")
	if err != nil {
		t.Fatalf("GET /skill: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Orkestra") {
		t.Errorf("skill endpoint should contain 'Orkestra', got: %q", string(body)[:min(200, len(body))])
	}
}

func TestSSERequiresAuth(t *testing.T) {
	base := startTestServerWithCfg(t, orkmcp.Config{
		ProjectID: "authtest",
		BindAddr:  "127.0.0.1",
		MCPToken:  "secret123",
	})

	// SSE without token → 401
	resp, err := http.Get(base + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	// /health should still work without token
	resp2, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("health without token: expected 200, got %d", resp2.StatusCode)
	}
}

func TestMessageRequiresAuth(t *testing.T) {
	base := startTestServerWithCfg(t, orkmcp.Config{
		ProjectID: "authtest2",
		BindAddr:  "127.0.0.1",
		MCPToken:  "tok456",
	})

	// POST /message without token → 401
	resp, err := http.Post(base+"/message", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /message: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 on /message without token, got %d", resp.StatusCode)
	}

	// POST /message with correct token → not 401 (may be 400/other due to missing session, but not 401)
	req, _ := http.NewRequest("POST", base+"/message", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok456")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /message with token: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode == http.StatusUnauthorized {
		t.Errorf("expected non-401 with valid token, got 401")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
