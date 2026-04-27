//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestE2EFullWorkflow exercises the core ticket lifecycle end-to-end:
// create → claim → seq_blocked → update → comment → link → board
func TestE2EFullWorkflow(t *testing.T) {
	base := startTestServer(t, "e2e")
	c := newMCPClient(t, base, "")

	// Create a parent epic
	epicOut := c.CallTool(t, "ticket_create", map[string]any{
		"title": "E2E Epic",
		"type":  "ep",
	})
	if !strings.Contains(epicOut, "TOON/1") {
		t.Fatalf("expected TOON/1 in ticket_create response: %q", epicOut)
	}
	epicID := extractTOONField(epicOut, "id:")
	if epicID == "" {
		t.Fatalf("could not extract epic ID from: %q", epicOut)
	}

	// Create two sequential child tasks
	ord1Out := c.CallTool(t, "ticket_create", map[string]any{
		"title":      "Step 1",
		"parent_id":  epicID,
		"exec_mode":  "seq",
		"exec_order": 1,
	})
	step1ID := extractTOONField(ord1Out, "id:")
	if step1ID == "" {
		t.Fatalf("could not extract step1 ID from: %q", ord1Out)
	}

	ord2Out := c.CallTool(t, "ticket_create", map[string]any{
		"title":      "Step 2",
		"parent_id":  epicID,
		"exec_mode":  "seq",
		"exec_order": 2,
	})
	step2ID := extractTOONField(ord2Out, "id:")
	if step2ID == "" {
		t.Fatalf("could not extract step2 ID from: %q", ord2Out)
	}

	// Claim step1 should succeed
	claimOut := c.CallTool(t, "ticket_claim", map[string]any{"id": step1ID})
	if !strings.Contains(claimOut, "ip") {
		t.Errorf("expected status ip after claim: %q", claimOut)
	}

	// Claim step2 before step1 is done → seq_blocked
	blockedOut := c.CallTool(t, "ticket_claim", map[string]any{"id": step2ID})
	if !strings.Contains(blockedOut, "seq_blocked") {
		t.Errorf("expected seq_blocked error: %q", blockedOut)
	}

	// Get step1 etag so we can update it
	getOut := c.CallTool(t, "ticket_get", map[string]any{"id": step1ID})
	etag := extractTOONField(getOut, "ua:")
	if etag == "" {
		t.Fatalf("could not extract etag from: %q", getOut)
	}

	// Update step1 to done
	updateOut := c.CallTool(t, "ticket_update", map[string]any{
		"id":     step1ID,
		"etag":   etag,
		"status": "dn",
	})
	if !strings.Contains(updateOut, "dn") {
		t.Errorf("expected status dn after update: %q", updateOut)
	}

	// Add a comment to step1
	commentOut := c.CallTool(t, "ticket_comment", map[string]any{
		"id":     step1ID,
		"author": "e2e-agent",
		"body":   "completed in e2e test",
	})
	if !strings.Contains(commentOut, "completed in e2e test") {
		t.Errorf("expected comment body in output: %q", commentOut)
	}

	// Link epic blocks step2
	linkOut := c.CallTool(t, "ticket_link", map[string]any{
		"from_id":   epicID,
		"to_id":     step2ID,
		"link_type": "blk",
	})
	if !strings.Contains(linkOut, "ok:true") {
		t.Errorf("expected ok:true from ticket_link: %q", linkOut)
	}

	// Board should show in-progress and backlog columns
	boardOut := c.CallTool(t, "ticket_board", map[string]any{})
	if !strings.Contains(boardOut, "BOARD{") {
		t.Errorf("expected BOARD in output: %q", boardOut)
	}
}

// TestE2EAuthFlow verifies SSE returns 401 without a token, 200 with.
func TestE2EAuthFlow(t *testing.T) {
	base := startTestServerWithToken(t, "authtest", "secret-e2e")

	// No token → 401
	resp, err := http.Get(base + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Health remains open
	resp2, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on /health without token, got %d", resp2.StatusCode)
	}

	// With correct token, full workflow works
	c := newMCPClient(t, base, "secret-e2e")
	out := c.CallTool(t, "ticket_create", map[string]any{"title": "auth test ticket"})
	if !strings.Contains(out, "TOON/1") {
		t.Errorf("expected TOON/1 with valid token: %q", out)
	}
}

// TestE2ESearch verifies FTS5 search returns a matching ticket.
func TestE2ESearch(t *testing.T) {
	base := startTestServer(t, "searchtest")
	c := newMCPClient(t, base, "")

	c.CallTool(t, "ticket_create", map[string]any{"title": "XyZuniqueSearchWord OAuth flow"})
	c.CallTool(t, "ticket_create", map[string]any{"title": "Unrelated task about caching"})

	out := c.CallTool(t, "ticket_search", map[string]any{"query": "XyZuniqueSearchWord"})
	if !strings.Contains(out, "XyZuniqueSearchWord") {
		t.Errorf("expected unique search term in results: %q", out)
	}
	if strings.Contains(out, "Unrelated") {
		t.Errorf("unexpected unrelated ticket in search results: %q", out)
	}
}

// TestE2EHealthEndpoint verifies the health endpoint reports db_ok.
func TestE2EHealthEndpoint(t *testing.T) {
	base := startTestServer(t, "healthtest")

	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if body["db_ok"] != true {
		t.Errorf("expected db_ok: true, got %v", body["db_ok"])
	}
	if body["status"] != "ok" {
		t.Errorf("expected status: ok, got %v", body["status"])
	}
}

// TestE2EBacklog verifies backlog returns tickets sorted by priority.
func TestE2EBacklog(t *testing.T) {
	base := startTestServer(t, "backlogtest")
	c := newMCPClient(t, base, "")

	c.CallTool(t, "ticket_create", map[string]any{"title": "low pri task", "priority": "l"})
	c.CallTool(t, "ticket_create", map[string]any{"title": "critical task", "priority": "cr"})
	c.CallTool(t, "ticket_create", map[string]any{"title": "high pri task", "priority": "h"})

	out := c.CallTool(t, "ticket_backlog", map[string]any{})
	if !strings.Contains(out, "TOON/1") {
		t.Errorf("expected TOON/1 in backlog response: %q", out)
	}
	// Critical should appear before low in the output
	critIdx := strings.Index(out, "critical task")
	lowIdx := strings.Index(out, "low pri task")
	if critIdx < 0 || lowIdx < 0 {
		t.Errorf("expected both tickets in backlog output: %q", out)
	} else if critIdx > lowIdx {
		t.Errorf("expected critical before low in sorted backlog: %q", out)
	}
}
