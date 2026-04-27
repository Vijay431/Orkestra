package ticket_test

import (
	"context"
	"testing"

	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
)

func TestServicePing(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	if err := svc.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestServiceLastBackupEmpty(t *testing.T) {
	svc := testutil.NewTestService(t, "test")
	name, ts := svc.LastBackup(t.TempDir())
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time, got %v", ts)
	}
}

func TestServiceCreate(t *testing.T) {
	svc := testutil.NewTestService(t, "svc")
	ctx := context.Background()

	tk, err := svc.Create(ctx, ticket.CreateInput{Title: "via service", Type: ticket.TypeFeature})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.ID != "svc-001" {
		t.Errorf("ID = %q, want svc-001", tk.ID)
	}

	got, err := svc.Get(ctx, tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "via service" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Type != ticket.TypeFeature {
		t.Errorf("Type = %q, want ft", got.Type)
	}
}

func TestServiceUpdate(t *testing.T) {
	svc := testutil.NewTestService(t, "svc")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "original"})
	etag := tk.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z")
	newTitle := "updated"
	got, err := svc.Update(ctx, ticket.UpdateInput{ID: tk.ID, Etag: etag, Title: &newTitle})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != "updated" {
		t.Errorf("Title = %q, want updated", got.Title)
	}
}

func TestServiceAddComment(t *testing.T) {
	svc := testutil.NewTestService(t, "svc")
	ctx := context.Background()

	tk, _ := svc.Create(ctx, ticket.CreateInput{Title: "commented"})
	got, err := svc.AddComment(ctx, tk.ID, "bot", "first comment")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(got.Comments))
	}
	if got.Comments[0].Body != "first comment" {
		t.Errorf("Body = %q", got.Comments[0].Body)
	}
}

func TestServiceAddLink(t *testing.T) {
	svc := testutil.NewTestService(t, "svc")
	ctx := context.Background()

	a, _ := svc.Create(ctx, ticket.CreateInput{Title: "A"})
	b, _ := svc.Create(ctx, ticket.CreateInput{Title: "B"})
	if err := svc.AddLink(ctx, a.ID, b.ID, ticket.LinkBlocks); err != nil {
		t.Fatalf("AddLink: %v", err)
	}

	got, err := svc.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(got.Links))
	}
	if got.Links[0].LinkType != ticket.LinkBlocks {
		t.Errorf("LinkType = %q, want blk", got.Links[0].LinkType)
	}
}
