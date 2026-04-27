package ticket_test

import (
	"context"
	"os"
	"testing"

	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
)

func TestCreateAndGet(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	in := ticket.CreateInput{
		Title:    "Fix login bug",
		Type:     ticket.TypeBug,
		Priority: ticket.PriorityHigh,
		Labels:   []string{"auth", "security"},
	}
	created, err := s.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "test-001" {
		t.Errorf("ID = %q, want test-001", created.ID)
	}
	if created.Status != ticket.StatusBacklog {
		t.Errorf("Status = %q, want bk", created.Status)
	}
	if len(created.Labels) != 2 {
		t.Errorf("Labels = %v, want 2 items", created.Labels)
	}

	got, err := s.Get(ctx, "test-001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Fix login bug" {
		t.Errorf("Title = %q", got.Title)
	}
}

func TestIDSequence(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		in := ticket.CreateInput{Title: "ticket"}
		got, err := s.Create(ctx, in)
		if err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
		want := "test-00" + string(rune('0'+i))
		if got.ID != want {
			t.Errorf("ticket %d: ID = %q, want %q", i, got.ID, want)
		}
	}
}

func TestClaim(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	t1, _ := s.Create(ctx, ticket.CreateInput{Title: "task"})

	claimed, err := s.Claim(ctx, t1.ID)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed.Status != ticket.StatusInProgress {
		t.Errorf("Status = %q, want ip", claimed.Status)
	}

	_, err = s.Claim(ctx, t1.ID)
	if err == nil {
		t.Error("expected conflict on double-claim")
	}
}

func TestSeqBlocked(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	parent, _ := s.Create(ctx, ticket.CreateInput{Title: "pipeline", ExecMode: ticket.ExecModeSequential})
	ord1, ord2 := 1, 2
	s.Create(ctx, ticket.CreateInput{Title: "step1", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord1})
	c2, _ := s.Create(ctx, ticket.CreateInput{Title: "step2", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord2})

	_, err := s.Claim(ctx, c2.ID)
	if err == nil {
		t.Error("expected seq_blocked error")
	}
}

func TestEtagConflict(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	t1, _ := s.Create(ctx, ticket.CreateInput{Title: "original"})

	staleEtag := "2000-01-01T00:00:00Z"
	newTitle := "updated"
	_, err := s.Update(ctx, ticket.UpdateInput{
		ID:    t1.ID,
		Etag:  staleEtag,
		Title: &newTitle,
	})
	if err == nil {
		t.Error("expected conflict with stale etag")
	}
}

func TestArchive(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	t1, _ := s.Create(ctx, ticket.CreateInput{Title: "to archive"})
	if err := s.Archive(ctx, t1.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	tickets, _ := s.List(ctx, ticket.ListFilter{})
	for _, tk := range tickets {
		if tk.ID == t1.ID {
			t.Error("archived ticket appeared in list without include_archived")
		}
	}

	tickets, _ = s.List(ctx, ticket.ListFilter{IncludeArchived: true})
	found := false
	for _, tk := range tickets {
		if tk.ID == t1.ID {
			found = true
		}
	}
	if !found {
		t.Error("archived ticket not found with include_archived=true")
	}
}

func TestChildrenOrdering(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	parent, _ := s.Create(ctx, ticket.CreateInput{Title: "parent", ExecMode: ticket.ExecModeSequential})
	ord3, ord1, ord2 := 3, 1, 2
	s.Create(ctx, ticket.CreateInput{Title: "c3", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord3})
	s.Create(ctx, ticket.CreateInput{Title: "c1", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord1})
	s.Create(ctx, ticket.CreateInput{Title: "c2", ParentID: parent.ID, ExecMode: ticket.ExecModeSequential, ExecOrder: &ord2})

	children, err := s.Children(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	for i, c := range children {
		want := i + 1
		if c.ExecOrder == nil || *c.ExecOrder != want {
			t.Errorf("children[%d].ExecOrder = %v, want %d", i, c.ExecOrder, want)
		}
	}
}

func TestFTSSearch(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("FTS5 content tables may behave differently in CI")
	}
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	s.Create(ctx, ticket.CreateInput{Title: "JWT authentication bug", Description: "Token refresh fails"})
	s.Create(ctx, ticket.CreateInput{Title: "Rate limiter feature", Description: "Add throttling"})

	results, err := s.Search(ctx, "JWT", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one FTS result for 'JWT'")
	}
}

func TestDuplicateExecOrder(t *testing.T) {
	s := testutil.NewTestStore(t, "test")
	ctx := context.Background()

	parent, _ := s.Create(ctx, ticket.CreateInput{Title: "parent"})
	ord := 1
	s.Create(ctx, ticket.CreateInput{Title: "first", ParentID: parent.ID, ExecOrder: &ord})

	_, err := s.Create(ctx, ticket.CreateInput{Title: "duplicate", ParentID: parent.ID, ExecOrder: &ord})
	if err == nil {
		t.Error("expected error for duplicate exec_order within same parent")
	}
}
