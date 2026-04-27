// SPDX-License-Identifier: MIT

package toon

import (
	"context"
	"fmt"
	"strings"

	"github.com/vijay431/orkestra/internal/ticket"
)

const maxDiagramDepth = 3

// ChildLoader fetches direct children for a given ticket ID.
type ChildLoader func(ctx context.Context, id string) ([]ticket.Ticket, error)

// GenerateDiagram produces a Mermaid flowchart for a ticket and its subtree.
// maxDepth caps the recursion (default: maxDiagramDepth).
func GenerateDiagram(ctx context.Context, root *ticket.Ticket, load ChildLoader, maxDepth int) string {
	if maxDepth <= 0 {
		maxDepth = maxDiagramDepth
	}

	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	// Root node
	rootLabel := nodeLabel(root)
	sb.WriteString(fmt.Sprintf("  %s[%q]\n", nodeID(root.ID), rootLabel))
	sb.WriteString(applyStyle(root))

	// Collect style directives separately so they go at the end
	var styles []string
	styles = append(styles, styleFor(root))

	renderChildren(ctx, &sb, &styles, root, load, 0, maxDepth)

	// Emit styles
	for _, s := range styles {
		if s != "" {
			sb.WriteString(s)
		}
	}

	return sb.String()
}

func renderChildren(ctx context.Context, sb *strings.Builder, styles *[]string, parent *ticket.Ticket, load ChildLoader, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}

	children, err := load(ctx, parent.ID)
	if err != nil || len(children) == 0 {
		return
	}

	// Separate into parallel and sequential groups
	var parChildren []ticket.Ticket
	var seqChildren []ticket.Ticket
	for _, c := range children {
		if c.ExecMode == ticket.ExecModeSequential {
			seqChildren = append(seqChildren, c)
		} else {
			parChildren = append(parChildren, c)
		}
	}

	parentNodeID := nodeID(parent.ID)

	// Parallel subgraph
	if len(parChildren) > 0 {
		subID := fmt.Sprintf("par_%s", sanitize(parent.ID))
		sb.WriteString(fmt.Sprintf("  subgraph %s[\"⚡ Parallel\"]\n", subID))
		for _, c := range parChildren {
			c := c
			label := nodeLabel(&c)
			sb.WriteString(fmt.Sprintf("    %s[%q]\n", nodeID(c.ID), label))
			*styles = append(*styles, styleFor(&c))
		}
		sb.WriteString("  end\n")
		sb.WriteString(fmt.Sprintf("  %s --> %s\n", parentNodeID, subID))

		// Recurse into parallel children
		for _, c := range parChildren {
			c := c
			renderChildren(ctx, sb, styles, &c, load, depth+1, maxDepth)
		}
	}

	// Sequential subgraph
	if len(seqChildren) > 0 {
		subID := fmt.Sprintf("seq_%s", sanitize(parent.ID))
		sb.WriteString(fmt.Sprintf("  subgraph %s[\"🔗 Sequential\"]\n", subID))
		for _, c := range seqChildren {
			c := c
			label := nodeLabel(&c)
			sb.WriteString(fmt.Sprintf("    %s[%q]\n", nodeID(c.ID), label))
			*styles = append(*styles, styleFor(&c))
		}
		// Chain arrows for sequential ordering
		for i := 1; i < len(seqChildren); i++ {
			prev := seqChildren[i-1]
			curr := seqChildren[i]
			sb.WriteString(fmt.Sprintf("  %s -->|\"%d→%d\"| %s\n",
				nodeID(prev.ID), i, i+1, nodeID(curr.ID)))
		}
		sb.WriteString("  end\n")
		sb.WriteString(fmt.Sprintf("  %s --> %s\n", parentNodeID, subID))
	}
}

func nodeID(id string) string {
	return sanitize(id)
}

func sanitize(s string) string {
	return strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(s)
}

func nodeLabel(t *ticket.Ticket) string {
	status := string(t.Status)
	priority := string(t.Priority)
	typ := string(t.Type)
	title := t.Title
	if len(title) > 30 {
		title = title[:27] + "..."
	}
	return fmt.Sprintf("%s [%s|%s|%s]", title, typ, status, priority)
}

func applyStyle(t *ticket.Ticket) string {
	return styleFor(t)
}

func styleFor(t *ticket.Ticket) string {
	color, ok := statusColor[string(t.Status)]
	if !ok {
		return ""
	}
	textColor := ""
	if string(t.Status) == "ip" {
		textColor = ",color:#fff"
	}
	return fmt.Sprintf("  style %s fill:%s%s\n", nodeID(t.ID), color, textColor)
}
