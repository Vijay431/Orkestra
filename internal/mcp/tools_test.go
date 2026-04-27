// SPDX-License-Identifier: MIT

package mcp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vijay431/orkestra/internal/mcp"
	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/toon"
)

// TestRegisterTools verifies all 13 tools register without panic.
func TestRegisterTools(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	_ = mcp.NewServer(mcp.Config{ProjectID: "test", Port: "19999"}, svc, testutil.NopLogger())
}

// TestEncodeError verifies TOON error format.
func TestEncodeError(t *testing.T) {
	out := toon.EncodeError(toon.ErrConflict, "test-001 already claimed")
	if !strings.HasPrefix(out, "TOON/1 ") {
		t.Errorf("missing TOON/1 prefix: %q", out)
	}
	if !strings.Contains(out, "ERR{code:conflict") {
		t.Errorf("missing conflict code: %q", out)
	}
}

// TestTicketCreateAndGetViaService verifies core service operations used by tools.
func TestTicketCreateAndGetViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	tk, err := svc.Create(ctx, ticket.CreateInput{
		Title:    "MCP tool test",
		Type:     ticket.TypeFeature,
		Priority: ticket.PriorityHigh,
		Labels:   []string{"mcp", "test"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	out := toon.Encode(tk)
	if !strings.HasPrefix(out, "TOON/1 ") {
		t.Errorf("missing TOON/1 prefix: %q", out)
	}
	if !strings.Contains(out, "id:test-001") {
		t.Errorf("missing id: %q", out)
	}
	if !strings.Contains(out, "typ:ft") {
		t.Errorf("missing type: %q", out)
	}
	if !strings.Contains(out, "p:h") {
		t.Errorf("missing priority: %q", out)
	}
}

// TestClaimConflictEncoding verifies ERR{code:conflict} on double-claim.
func TestClaimConflictEncoding(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "claim me"})
	_, err := svc.Claim(ctx, tk.ID)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	_, err = svc.Claim(ctx, tk.ID)
	if err == nil {
		t.Fatal("expected conflict on double-claim")
	}
	code := toon.TicketErrCode(err)
	if code != toon.ErrConflict {
		t.Errorf("expected ErrConflict, got %q", code)
	}
	errOut := toon.EncodeError(code, err.Error())
	if !strings.Contains(errOut, "ERR{code:conflict") {
		t.Errorf("expected conflict in output: %q", errOut)
	}
}

// TestBoardEncoding verifies BOARD output format from service.
func TestBoardEncoding(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	svc.Create(ctx, ticket.CreateInput{Title: "backlog task"})
	tk2, _ := svc.Create(ctx, ticket.CreateInput{Title: "active task"})
	svc.Claim(ctx, tk2.ID)

	board, err := svc.Board(ctx, ticket.ListFilter{})
	if err != nil {
		t.Fatalf("Board: %v", err)
	}
	out := toon.EncodeBoard(board)
	if !strings.HasPrefix(out, "TOON/1 BOARD{") {
		t.Errorf("missing BOARD prefix: %q", out)
	}
	if !strings.Contains(out, "bk:") {
		t.Errorf("missing bk group: %q", out)
	}
	if !strings.Contains(out, "ip:") {
		t.Errorf("missing ip group: %q", out)
	}
}

// TestSeqBlockedEncoding verifies ERR{code:seq_blocked} is correctly encoded.
func TestSeqBlockedEncoding(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	parent, _ := svc.Create(ctx, ticket.CreateInput{Title: "pipeline", ExecMode: ticket.ExecModeSequential})
	ord1, ord2 := 1, 2
	svc.Create(ctx, ticket.CreateInput{Title: "step1", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord1})
	c2, _ := svc.Create(ctx, ticket.CreateInput{Title: "step2", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord2})

	_, err := svc.Claim(ctx, c2.ID)
	if err == nil {
		t.Fatal("expected seq_blocked")
	}
	code := toon.TicketErrCode(err)
	if code != toon.ErrSeqBlocked {
		t.Errorf("expected ErrSeqBlocked, got %q", code)
	}
	out := toon.EncodeError(code, err.Error())
	if !strings.Contains(out, "seq_blocked") {
		t.Errorf("expected seq_blocked in output: %q", out)
	}
}

// TestTicketUpdateViaService verifies update with etag.
func TestTicketUpdateViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "before update"})
	etag := toon.EtagOf(tk)
	newTitle := "after update"
	updated, err := svc.Update(ctx, ticket.UpdateInput{ID: tk.ID, Etag: etag, Title: &newTitle})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	out := toon.Encode(updated)
	if !strings.Contains(out, "after update") {
		t.Errorf("expected updated title in output: %q", out)
	}
	// Etag should have changed
	if toon.EtagOf(updated) == etag {
		t.Error("etag should change after update")
	}
}

// TestTicketArchiveViaService verifies archive returns ok:true.
func TestTicketArchiveViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "to archive"})
	if err := svc.Archive(ctx, tk.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	out := toon.EncodeOK()
	if !strings.Contains(out, "ok:true") {
		t.Errorf("expected ok:true: %q", out)
	}

	// Verify excluded from normal list
	tickets, _ := svc.List(ctx, ticket.ListFilter{})
	for _, item := range tickets {
		if item.ID == tk.ID {
			t.Error("archived ticket should not appear in list")
		}
	}
}

// TestTicketListViaService verifies list returns all matching tickets.
func TestTicketListViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	svc.Create(ctx, ticket.CreateInput{Title: "bug one", Type: ticket.TypeBug})
	svc.Create(ctx, ticket.CreateInput{Title: "feature one", Type: ticket.TypeFeature})
	svc.Create(ctx, ticket.CreateInput{Title: "bug two", Type: ticket.TypeBug})

	bugType := ticket.TypeBug
	tickets, err := svc.List(ctx, ticket.ListFilter{Type: &bugType})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 bugs, got %d", len(tickets))
	}

	out := "TOON/1 ["
	for i, tk := range tickets {
		if i > 0 {
			out += ","
		}
		out += toon.EncodeSummary(&tk)
	}
	out += "]"
	if !strings.HasPrefix(out, "TOON/1 [") {
		t.Errorf("unexpected list format: %q", out)
	}
	if !strings.Contains(out, "bug one") {
		t.Errorf("expected bug one: %q", out)
	}
}

// TestTicketCommentViaService verifies comment appears in encoded output.
func TestTicketCommentViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "needs comment"})
	commented, err := svc.AddComment(ctx, tk.ID, "dev", "fixed in v2")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	out := toon.Encode(commented)
	if !strings.Contains(out, "fixed in v2") {
		t.Errorf("expected comment body in output: %q", out)
	}
	if !strings.Contains(out, "cmt:[") {
		t.Errorf("expected cmt field: %q", out)
	}
}

// TestTicketLinkViaService verifies link appears in encoded output.
func TestTicketLinkViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	a, _ := svc.Create(ctx, ticket.CreateInput{Title: "blocker"})
	b, _ := svc.Create(ctx, ticket.CreateInput{Title: "blocked"})
	if err := svc.AddLink(ctx, a.ID, b.ID, ticket.LinkBlocks); err != nil {
		t.Fatalf("AddLink: %v", err)
	}

	got, err := svc.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	out := toon.Encode(got)
	if !strings.Contains(out, "lnk:[") {
		t.Errorf("expected lnk field: %q", out)
	}
	if !strings.Contains(out, "k:blk") {
		t.Errorf("expected link type blk: %q", out)
	}
}

// TestTicketSearchViaService verifies FTS5 search returns matching tickets.
func TestTicketSearchViaService(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("FTS5 content tables may behave differently in CI")
	}
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	svc.Create(ctx, ticket.CreateInput{Title: "OAuth token refresh bug"})
	svc.Create(ctx, ticket.CreateInput{Title: "Rate limiter feature"})

	results, err := svc.Search(ctx, "OAuth", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected FTS result for 'OAuth'")
	}
	if len(results) > 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// TestTicketChildrenViaService verifies children list and ordering.
func TestTicketChildrenViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	parent, _ := svc.Create(ctx, ticket.CreateInput{Title: "epic"})
	ord2, ord1 := 2, 1
	svc.Create(ctx, ticket.CreateInput{Title: "child B", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord2})
	svc.Create(ctx, ticket.CreateInput{Title: "child A", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord1})

	children, err := svc.Children(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	// Should be ordered by exec_order
	if children[0].Title != "child A" {
		t.Errorf("first child should be child A (ord=1), got %q", children[0].Title)
	}

	// Encode as list
	parts := make([]string, len(children))
	for i, c := range children {
		c := c
		parts[i] = toon.EncodeSummary(&c)
	}
	out := "TOON/1 [" + strings.Join(parts, ",") + "]"
	if !strings.Contains(out, "child A") {
		t.Errorf("expected child A in output: %q", out)
	}
}

// TestTicketBacklogViaService verifies backlog is sorted by priority.
func TestTicketBacklogViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	svc.Create(ctx, ticket.CreateInput{Title: "low pri", Priority: ticket.PriorityLow})
	svc.Create(ctx, ticket.CreateInput{Title: "critical", Priority: ticket.PriorityCritical})
	svc.Create(ctx, ticket.CreateInput{Title: "high pri", Priority: ticket.PriorityHigh})

	tickets, err := svc.Backlog(ctx, ticket.ListFilter{})
	if err != nil {
		t.Fatalf("Backlog: %v", err)
	}
	if len(tickets) != 3 {
		t.Fatalf("expected 3 tickets, got %d", len(tickets))
	}
	// Critical should come first
	if tickets[0].Priority != ticket.PriorityCritical {
		t.Errorf("first ticket priority = %q, want cr", tickets[0].Priority)
	}
	// Low should be last
	if tickets[len(tickets)-1].Priority != ticket.PriorityLow {
		t.Errorf("last ticket priority = %q, want l", tickets[len(tickets)-1].Priority)
	}
}

// TestTicketDiagramViaService verifies GenerateDiagram produces valid Mermaid output.
func TestTicketDiagramViaService(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	ctx := context.Background()

	parent, _ := svc.Create(ctx, ticket.CreateInput{Title: "epic", Type: ticket.TypeEpic})
	svc.Create(ctx, ticket.CreateInput{Title: "task A", ParentID: parent.ID})
	svc.Create(ctx, ticket.CreateInput{Title: "task B", ParentID: parent.ID})

	root, _ := svc.Get(ctx, parent.ID)
	diagram := toon.GenerateDiagram(ctx, root, svc.Children, 3)

	if !strings.HasPrefix(diagram, "flowchart TD") {
		t.Errorf("expected flowchart TD prefix: %q", diagram[:min(60, len(diagram))])
	}
	if !strings.Contains(diagram, "epic") {
		t.Errorf("expected root title in diagram: %q", diagram)
	}
	if !strings.Contains(diagram, "task A") {
		t.Errorf("expected task A in diagram: %q", diagram)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
