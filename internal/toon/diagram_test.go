// SPDX-License-Identifier: MIT

package toon_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/toon"
)

func noChildren(_ context.Context, _ string) ([]ticket.Ticket, error) {
	return nil, nil
}

func makeTk(id, title string, status ticket.Status, mode ticket.ExecMode, order *int) *ticket.Ticket {
	return &ticket.Ticket{
		ID:        id,
		Title:     title,
		Status:    status,
		Priority:  ticket.PriorityMedium,
		Type:      ticket.TypeTask,
		ExecMode:  mode,
		ExecOrder: order,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestDiagramSingleNode(t *testing.T) {
	root := makeTk("proj-001", "Root task", ticket.StatusBacklog, ticket.ExecModeParallel, nil)
	out := toon.GenerateDiagram(context.Background(), root, noChildren, 3)

	if !strings.HasPrefix(out, "flowchart TD") {
		t.Errorf("expected flowchart TD prefix: %q", out[:min(50, len(out))])
	}
	if !strings.Contains(out, "proj_001") {
		t.Errorf("expected sanitized node ID proj_001: %q", out)
	}
	if !strings.Contains(out, "Root task") {
		t.Errorf("expected title in node label: %q", out)
	}
}

func TestDiagramParallelChildren(t *testing.T) {
	root := makeTk("proj-001", "Parent", ticket.StatusTodo, ticket.ExecModeParallel, nil)
	c1 := makeTk("proj-002", "Child A", ticket.StatusBacklog, ticket.ExecModeParallel, nil)
	c2 := makeTk("proj-003", "Child B", ticket.StatusBacklog, ticket.ExecModeParallel, nil)

	loader := func(_ context.Context, id string) ([]ticket.Ticket, error) {
		if id == root.ID {
			return []ticket.Ticket{*c1, *c2}, nil
		}
		return nil, nil
	}

	out := toon.GenerateDiagram(context.Background(), root, loader, 3)
	if !strings.Contains(out, "⚡ Parallel") {
		t.Errorf("expected parallel subgraph: %q", out)
	}
	if strings.Contains(out, "🔗 Sequential") {
		t.Errorf("unexpected sequential subgraph: %q", out)
	}
	if !strings.Contains(out, "Child A") {
		t.Errorf("expected child A in diagram: %q", out)
	}
}

func TestDiagramSequentialChildren(t *testing.T) {
	root := makeTk("proj-001", "Pipeline", ticket.StatusBacklog, ticket.ExecModeSequential, nil)
	ord1, ord2 := 1, 2
	c1 := makeTk("proj-002", "Step 1", ticket.StatusBacklog, ticket.ExecModeSequential, &ord1)
	c2 := makeTk("proj-003", "Step 2", ticket.StatusBacklog, ticket.ExecModeSequential, &ord2)

	loader := func(_ context.Context, id string) ([]ticket.Ticket, error) {
		if id == root.ID {
			return []ticket.Ticket{*c1, *c2}, nil
		}
		return nil, nil
	}

	out := toon.GenerateDiagram(context.Background(), root, loader, 3)
	if !strings.Contains(out, "🔗 Sequential") {
		t.Errorf("expected sequential subgraph: %q", out)
	}
	// Sequential children should have ordering arrows
	if !strings.Contains(out, "1→2") {
		t.Errorf("expected ordering arrow 1→2: %q", out)
	}
}

func TestDiagramStatusStyling(t *testing.T) {
	cases := []struct {
		status ticket.Status
		color  string
	}{
		{ticket.StatusBacklog, "#aaaaaa"},
		{ticket.StatusTodo, "#f0c040"},
		{ticket.StatusInProgress, "#40a0f0"},
		{ticket.StatusDone, "#40c040"},
		{ticket.StatusBlocked, "#f04040"},
		{ticket.StatusCancelled, "#cccccc"},
	}
	for _, tc := range cases {
		root := makeTk("proj-001", "task", tc.status, ticket.ExecModeParallel, nil)
		out := toon.GenerateDiagram(context.Background(), root, noChildren, 3)
		if !strings.Contains(out, tc.color) {
			t.Errorf("status %q: expected color %q in output: %q", tc.status, tc.color, out)
		}
	}
}

func TestDiagramDeepTree(t *testing.T) {
	root := makeTk("p-001", "Root", ticket.StatusBacklog, ticket.ExecModeParallel, nil)
	l1 := makeTk("p-002", "Level1", ticket.StatusBacklog, ticket.ExecModeParallel, nil)
	l2 := makeTk("p-003", "Level2", ticket.StatusBacklog, ticket.ExecModeParallel, nil)
	l3 := makeTk("p-004", "Level3", ticket.StatusBacklog, ticket.ExecModeParallel, nil)

	loader := func(_ context.Context, id string) ([]ticket.Ticket, error) {
		switch id {
		case root.ID:
			return []ticket.Ticket{*l1}, nil
		case l1.ID:
			return []ticket.Ticket{*l2}, nil
		case l2.ID:
			return []ticket.Ticket{*l3}, nil
		}
		return nil, nil
	}

	// maxDepth=2 should stop before Level3
	out := toon.GenerateDiagram(context.Background(), root, loader, 2)
	if !strings.Contains(out, "Level1") {
		t.Errorf("expected Level1: %q", out)
	}
	if !strings.Contains(out, "Level2") {
		t.Errorf("expected Level2: %q", out)
	}
	if strings.Contains(out, "Level3") {
		t.Errorf("Level3 should be excluded at maxDepth=2: %q", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
