package toon_test

import (
	"strings"
	"testing"
	"time"

	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/toon"
)

func baseTicket() *ticket.Ticket {
	return &ticket.Ticket{
		ID:        "test-001",
		Title:     "simple",
		Status:    ticket.StatusBacklog,
		Priority:  ticket.PriorityMedium,
		Type:      ticket.TypeTask,
		ExecMode:  ticket.ExecModeParallel,
		CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestEncodeVersionPrefix(t *testing.T) {
	tk := baseTicket()
	out := toon.Encode(tk)
	if !strings.HasPrefix(out, "TOON/1 ") {
		t.Errorf("missing TOON/1 prefix: %q", out)
	}
}

func TestEncodeSpecialCharsInTitle(t *testing.T) {
	cases := []struct {
		title   string
		wantSub string
	}{
		{`say "hello"`, `t:"say \"hello\""`},
		{"line1\nline2", `t:"line1\nline2"`},
		{`back\slash`, `t:"back\\slash"`},
		{"has:colon", `t:"has:colon"`},
		{"has,comma", `t:"has,comma"`},
		{"has{brace}", `t:"has{brace}"`},
		{"nospace", `t:nospace`},
	}
	for _, tc := range cases {
		tk := baseTicket()
		tk.Title = tc.title
		out := toon.Encode(tk)
		if !strings.Contains(out, tc.wantSub) {
			t.Errorf("title=%q: got %q, want substring %q", tc.title, out, tc.wantSub)
		}
	}
}

func TestEncodeEmptyChildren(t *testing.T) {
	tk := baseTicket()
	tk.Children = []string{}
	out := toon.Encode(tk)
	if strings.Contains(out, "ch:") {
		t.Errorf("empty children should not appear in output: %q", out)
	}
}

func TestEncodeZeroExecOrder(t *testing.T) {
	tk := baseTicket()
	zero := 0
	tk.ExecOrder = &zero
	out := toon.Encode(tk)
	if !strings.Contains(out, "ord:0") {
		t.Errorf("zero exec_order should appear as ord:0: %q", out)
	}
}

func TestEncode50Labels(t *testing.T) {
	tk := baseTicket()
	labels := make([]string, 50)
	for i := range labels {
		labels[i] = "lbl"
	}
	tk.Labels = labels
	out := toon.Encode(tk)
	count := strings.Count(out, "lbl")
	if count < 50 {
		t.Errorf("expected ≥50 label occurrences, got %d", count)
	}
	if !strings.Contains(out, "lbl:[") {
		t.Errorf("labels array missing: %q", out)
	}
}

func TestEncodeParModeOmitted(t *testing.T) {
	tk := baseTicket()
	tk.ExecMode = ticket.ExecModeParallel
	out := toon.Encode(tk)
	if strings.Contains(out, "em:") {
		t.Errorf("parallel exec_mode should be omitted: %q", out)
	}
}

func TestEncodeSeqModeIncluded(t *testing.T) {
	tk := baseTicket()
	tk.ExecMode = ticket.ExecModeSequential
	out := toon.Encode(tk)
	if !strings.Contains(out, "em:seq") {
		t.Errorf("sequential exec_mode should appear: %q", out)
	}
}

func TestEncodeError(t *testing.T) {
	out := toon.EncodeError(toon.ErrNotFound, "test-001 does not exist")
	if !strings.HasPrefix(out, "TOON/1 ") {
		t.Errorf("missing prefix: %q", out)
	}
	if !strings.Contains(out, "ERR{") {
		t.Errorf("missing ERR envelope: %q", out)
	}
	if !strings.Contains(out, "code:not_found") {
		t.Errorf("missing error code: %q", out)
	}
}

func TestEncodeOK(t *testing.T) {
	out := toon.EncodeOK()
	if out != "TOON/1 {ok:true}" {
		t.Errorf("got %q", out)
	}
}

func TestEncodeBoard(t *testing.T) {
	bk := baseTicket()
	bk.Title = "backlog item"
	ip := baseTicket()
	ip.ID = "test-002"
	ip.Title = "active item"
	ip.Status = ticket.StatusInProgress

	board := map[string][]ticket.Ticket{
		"bk": {*bk},
		"ip": {*ip},
	}
	out := toon.EncodeBoard(board)
	if !strings.HasPrefix(out, "TOON/1 BOARD{") {
		t.Errorf("missing BOARD prefix: %q", out)
	}
	// bk should come before ip in statusOrder
	bkPos := strings.Index(out, "bk:")
	ipPos := strings.Index(out, "ip:")
	if bkPos > ipPos {
		t.Errorf("bk should appear before ip in board output")
	}
}

func TestEncodeComments(t *testing.T) {
	tk := baseTicket()
	tk.Comments = []ticket.Comment{
		{Author: "llm", Body: "started work", CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
	}
	out := toon.Encode(tk)
	if !strings.Contains(out, "cmt:[") {
		t.Errorf("missing comment: %q", out)
	}
	if !strings.Contains(out, "a:llm") {
		t.Errorf("missing author: %q", out)
	}
}

func TestEncodeLinks(t *testing.T) {
	tk := baseTicket()
	tk.Links = []ticket.Link{
		{FromID: "test-001", ToID: "test-002", LinkType: ticket.LinkBlocks},
	}
	out := toon.Encode(tk)
	if !strings.Contains(out, "lnk:[") {
		t.Errorf("missing links: %q", out)
	}
	if !strings.Contains(out, "k:blk") {
		t.Errorf("missing link type: %q", out)
	}
}

func TestEncodeSummaryOmitsDescriptionAndComments(t *testing.T) {
	tk := baseTicket()
	tk.Description = "detailed description"
	tk.Comments = []ticket.Comment{
		{Author: "llm", Body: "comment", CreatedAt: time.Now()},
	}
	out := toon.EncodeSummary(tk)
	if strings.Contains(out, "detailed description") {
		t.Errorf("summary should not include description: %q", out)
	}
	if strings.Contains(out, "cmt:") {
		t.Errorf("summary should not include comments: %q", out)
	}
}

func TestTicketErrCode(t *testing.T) {
	tests := []struct {
		err  error
		want toon.ErrCode
	}{
		{ticket.ErrNotFound, toon.ErrNotFound},
		{ticket.ErrConflict, toon.ErrConflict},
		{ticket.ErrSeqBlocked, toon.ErrSeqBlocked},
		{ticket.ErrInvalid, toon.ErrInvalid},
	}
	for _, tc := range tests {
		got := toon.TicketErrCode(tc.err)
		if got != tc.want {
			t.Errorf("TicketErrCode(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestEtagOf(t *testing.T) {
	tk := baseTicket()
	etag := toon.EtagOf(tk)
	if etag == "" {
		t.Error("EtagOf should not be empty")
	}
	if !strings.Contains(etag, "2024-01-15") {
		t.Errorf("EtagOf = %q, expected date 2024-01-15", etag)
	}
}
