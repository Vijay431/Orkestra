package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vijay431/orkestra/internal/ticket"
	"github.com/vijay431/orkestra/internal/toon"
)

// RegisterTools attaches all 13 Orkestra tools to the MCP server.
func RegisterTools(s *server.MCPServer, svc *ticket.Service) {
	s.AddTool(toolTicketCreate(), handleTicketCreate(svc))
	s.AddTool(toolTicketGet(), handleTicketGet(svc))
	s.AddTool(toolTicketClaim(), handleTicketClaim(svc))
	s.AddTool(toolTicketUpdate(), handleTicketUpdate(svc))
	s.AddTool(toolTicketArchive(), handleTicketArchive(svc))
	s.AddTool(toolTicketList(), handleTicketList(svc))
	s.AddTool(toolTicketComment(), handleTicketComment(svc))
	s.AddTool(toolTicketLink(), handleTicketLink(svc))
	s.AddTool(toolTicketSearch(), handleTicketSearch(svc))
	s.AddTool(toolTicketChildren(), handleTicketChildren(svc))
	s.AddTool(toolTicketBacklog(), handleTicketBacklog(svc))
	s.AddTool(toolTicketBoard(), handleTicketBoard(svc))
	s.AddTool(toolTicketDiagram(), handleTicketDiagram(svc))
}

// --- tool definitions ---

func toolTicketCreate() mcp.Tool {
	return mcp.NewTool("ticket_create",
		mcp.WithDescription("Create a new ticket in the project backlog. Returns TOON/1 encoded ticket."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Ticket title")),
		mcp.WithString("type", mcp.Description("Type: bug|ft|tsk|ep|chr (default: tsk)")),
		mcp.WithString("priority", mcp.Description("Priority: cr|h|m|l (default: m)")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithArray("labels", mcp.Description("Label strings"), mcp.WithStringItems()),
		mcp.WithString("parent_id", mcp.Description("Parent ticket ID for sub-tasks")),
		mcp.WithString("exec_mode", mcp.Description("Execution mode: par|seq (default: par)")),
		mcp.WithNumber("exec_order", mcp.Description("Execution order for sequential children (integer)")),
	)
}

func toolTicketGet() mcp.Tool {
	return mcp.NewTool("ticket_get",
		mcp.WithDescription("Get a ticket by ID with comments, links, and child IDs."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID (e.g. myapp-001)")),
	)
}

func toolTicketClaim() mcp.Tool {
	return mcp.NewTool("ticket_claim",
		mcp.WithDescription("Atomically claim a ticket by moving it to in_progress. Returns ERR{code:conflict} if already claimed. Enforces sequential ordering."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID to claim")),
	)
}

func toolTicketUpdate() mcp.Tool {
	return mcp.NewTool("ticket_update",
		mcp.WithDescription("Update ticket fields. Supply etag (updated_at from last read) for optimistic locking. Returns ERR{code:conflict} if etag is stale."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
		mcp.WithString("etag", mcp.Description("updated_at from last read (for optimistic locking)")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("status", mcp.Description("New status: bk|td|ip|dn|bl|cl")),
		mcp.WithString("priority", mcp.Description("New priority: cr|h|m|l")),
		mcp.WithString("type", mcp.Description("New type: bug|ft|tsk|ep|chr")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("assignee", mcp.Description("Assignee name")),
		mcp.WithArray("labels", mcp.Description("Replace labels"), mcp.WithStringItems()),
		mcp.WithString("exec_mode", mcp.Description("New exec_mode: par|seq")),
		mcp.WithNumber("exec_order", mcp.Description("New exec_order integer")),
	)
}

func toolTicketArchive() mcp.Tool {
	return mcp.NewTool("ticket_archive",
		mcp.WithDescription("Soft-delete a ticket (sets archived_at). Archived tickets are excluded from normal queries unless include_archived=true."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID to archive")),
	)
}

func toolTicketList() mcp.Tool {
	return mcp.NewTool("ticket_list",
		mcp.WithDescription("List tickets with optional filters. Returns TOON/1 array."),
		mcp.WithString("status", mcp.Description("Filter by status: bk|td|ip|dn|bl|cl")),
		mcp.WithString("priority", mcp.Description("Filter by priority: cr|h|m|l")),
		mcp.WithString("type", mcp.Description("Filter by type: bug|ft|tsk|ep|chr")),
		mcp.WithArray("labels", mcp.Description("Filter by labels"), mcp.WithStringItems()),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
		mcp.WithBoolean("include_archived", mcp.Description("Include archived tickets (default false)")),
	)
}

func toolTicketComment() mcp.Tool {
	return mcp.NewTool("ticket_comment",
		mcp.WithDescription("Add a comment to a ticket. Returns updated TOON/1 ticket."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Comment text")),
		mcp.WithString("author", mcp.Description("Author name (default: llm)")),
	)
}

func toolTicketLink() mcp.Tool {
	return mcp.NewTool("ticket_link",
		mcp.WithDescription("Create a directional link between two tickets."),
		mcp.WithString("from_id", mcp.Required(), mcp.Description("Source ticket ID")),
		mcp.WithString("to_id", mcp.Required(), mcp.Description("Target ticket ID")),
		mcp.WithString("link_type", mcp.Required(), mcp.Description("Link type: blk|rel|dup")),
	)
}

func toolTicketSearch() mcp.Tool {
	return mcp.NewTool("ticket_search",
		mcp.WithDescription("Full-text search across ticket titles, descriptions, and labels. Returns TOON/1 array ranked by relevance."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithBoolean("include_archived", mcp.Description("Include archived tickets (default false)")),
	)
}

func toolTicketChildren() mcp.Tool {
	return mcp.NewTool("ticket_children",
		mcp.WithDescription("List children of a ticket. Sequential children are sorted by exec_order. Set recursive=true for full subtree."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Parent ticket ID")),
		mcp.WithBoolean("recursive", mcp.Description("Return full subtree (default false)")),
		mcp.WithNumber("depth", mcp.Description("Max depth for recursive mode (default 3)")),
	)
}

func toolTicketBacklog() mcp.Tool {
	return mcp.NewTool("ticket_backlog",
		mcp.WithDescription("List backlog tickets ordered by priority (cr→h→m→l). Use this to decide what to work on next."),
		mcp.WithString("priority", mcp.Description("Filter by priority")),
		mcp.WithString("type", mcp.Description("Filter by type")),
		mcp.WithArray("labels", mcp.Description("Filter by labels"), mcp.WithStringItems()),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
	)
}

func toolTicketBoard() mcp.Tool {
	return mcp.NewTool("ticket_board",
		mcp.WithDescription("Show kanban board grouped by status: BOARD{bk:[...],td:[...],ip:[...],dn:[...]}"),
		mcp.WithString("type", mcp.Description("Filter by ticket type")),
		mcp.WithArray("labels", mcp.Description("Filter by labels"), mcp.WithStringItems()),
	)
}

func toolTicketDiagram() mcp.Tool {
	return mcp.NewTool("ticket_diagram",
		mcp.WithDescription("Generate a Mermaid flowchart of a ticket's hierarchy. Parallel children in ⚡ subgraph, sequential in 🔗 subgraph with ordering arrows. Nodes colored by status."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Root ticket ID")),
		mcp.WithBoolean("recursive", mcp.Description("Recurse into grandchildren (default true)")),
		mcp.WithNumber("depth", mcp.Description("Max depth (default 3)")),
	)
}

// --- handlers ---

func handleTicketCreate(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := req.RequireString("title")
		if err != nil {
			return errResult(toon.ErrInvalid, "title is required"), nil
		}

		in := ticket.CreateInput{
			Title:       title,
			Type:        ticket.TicketType(req.GetString("type", string(ticket.TypeTask))),
			Priority:    ticket.Priority(req.GetString("priority", string(ticket.PriorityMedium))),
			Description: req.GetString("description", ""),
			Labels:      req.GetStringSlice("labels", nil),
			ParentID:    req.GetString("parent_id", ""),
			ExecMode:    ticket.ExecMode(req.GetString("exec_mode", string(ticket.ExecModeParallel))),
		}
		if ord := req.GetInt("exec_order", -1); ord >= 0 {
			in.ExecOrder = &ord
		}

		t, err := svc.Create(ctx, in)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.Encode(t)), nil
	}
}

func handleTicketGet(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		t, err := svc.Get(ctx, id)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.Encode(t)), nil
	}
}

func handleTicketClaim(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		t, err := svc.Claim(ctx, id)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.Encode(t)), nil
	}
}

func handleTicketUpdate(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}

		in := ticket.UpdateInput{
			ID:   id,
			Etag: req.GetString("etag", ""),
		}

		if v := req.GetString("title", ""); v != "" {
			in.Title = &v
		}
		if v := req.GetString("status", ""); v != "" {
			s := ticket.Status(v)
			in.Status = &s
		}
		if v := req.GetString("priority", ""); v != "" {
			p := ticket.Priority(v)
			in.Priority = &p
		}
		if v := req.GetString("type", ""); v != "" {
			tp := ticket.TicketType(v)
			in.Type = &tp
		}
		if v := req.GetString("description", "\x00"); v != "\x00" {
			in.Description = &v
		}
		if v := req.GetString("assignee", ""); v != "" {
			in.Assignee = &v
		}
		if labels := req.GetStringSlice("labels", nil); labels != nil {
			in.Labels = labels
		}
		if v := req.GetString("exec_mode", ""); v != "" {
			em := ticket.ExecMode(v)
			in.ExecMode = &em
		}
		if ord := req.GetInt("exec_order", -1); ord >= 0 {
			in.ExecOrder = &ord
		}

		t, err := svc.Update(ctx, in)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.Encode(t)), nil
	}
}

func handleTicketArchive(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		if err := svc.Archive(ctx, id); err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.EncodeOK()), nil
	}
}

func handleTicketList(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := ticket.ListFilter{
			Labels:          req.GetStringSlice("labels", nil),
			Limit:           req.GetInt("limit", 50),
			IncludeArchived: req.GetBool("include_archived", false),
		}
		if v := req.GetString("status", ""); v != "" {
			s := ticket.Status(v)
			f.Status = &s
		}
		if v := req.GetString("priority", ""); v != "" {
			p := ticket.Priority(v)
			f.Priority = &p
		}
		if v := req.GetString("type", ""); v != "" {
			tp := ticket.TicketType(v)
			f.Type = &tp
		}
		tickets, err := svc.List(ctx, f)
		if err != nil {
			return errResult(toon.ErrInternal, err.Error()), nil
		}
		return mcp.NewToolResultText(encodeList(tickets)), nil
	}
}

func handleTicketComment(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		body, err := req.RequireString("body")
		if err != nil {
			return errResult(toon.ErrInvalid, "body is required"), nil
		}
		author := req.GetString("author", "llm")
		t, err := svc.AddComment(ctx, id, author, body)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.Encode(t)), nil
	}
}

func handleTicketLink(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fromID, err := req.RequireString("from_id")
		if err != nil {
			return errResult(toon.ErrInvalid, "from_id is required"), nil
		}
		toID, err := req.RequireString("to_id")
		if err != nil {
			return errResult(toon.ErrInvalid, "to_id is required"), nil
		}
		lt, err := req.RequireString("link_type")
		if err != nil {
			return errResult(toon.ErrInvalid, "link_type is required (blk|rel|dup)"), nil
		}
		if err := svc.AddLink(ctx, fromID, toID, ticket.LinkType(lt)); err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}
		return mcp.NewToolResultText(toon.EncodeOK()), nil
	}
}

func handleTicketSearch(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return errResult(toon.ErrInvalid, "query is required"), nil
		}
		includeArchived := req.GetBool("include_archived", false)
		tickets, err := svc.Search(ctx, query, includeArchived)
		if err != nil {
			return errResult(toon.ErrInternal, err.Error()), nil
		}
		return mcp.NewToolResultText(encodeList(tickets)), nil
	}
}

func handleTicketChildren(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		recursive := req.GetBool("recursive", false)
		depth := req.GetInt("depth", 3)
		if depth <= 0 || depth > 10 {
			depth = 3
		}

		var tickets []ticket.Ticket
		if recursive {
			tickets, err = svc.ChildrenDeep(ctx, id, depth)
		} else {
			tickets, err = svc.Children(ctx, id)
		}
		if err != nil {
			return errResult(toon.ErrInternal, err.Error()), nil
		}
		return mcp.NewToolResultText(encodeList(tickets)), nil
	}
}

func handleTicketBacklog(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := ticket.ListFilter{
			Labels: req.GetStringSlice("labels", nil),
			Limit:  req.GetInt("limit", 50),
		}
		if v := req.GetString("priority", ""); v != "" {
			p := ticket.Priority(v)
			f.Priority = &p
		}
		if v := req.GetString("type", ""); v != "" {
			tp := ticket.TicketType(v)
			f.Type = &tp
		}
		tickets, err := svc.Backlog(ctx, f)
		if err != nil {
			return errResult(toon.ErrInternal, err.Error()), nil
		}
		return mcp.NewToolResultText(encodeList(tickets)), nil
	}
}

func handleTicketBoard(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := ticket.ListFilter{}
		if v := req.GetString("type", ""); v != "" {
			tp := ticket.TicketType(v)
			f.Type = &tp
		}
		board, err := svc.Board(ctx, f)
		if err != nil {
			return errResult(toon.ErrInternal, err.Error()), nil
		}
		return mcp.NewToolResultText(toon.EncodeBoard(board)), nil
	}
}

func handleTicketDiagram(svc *ticket.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult(toon.ErrInvalid, "id is required"), nil
		}
		depth := req.GetInt("depth", 3)
		if depth <= 0 || depth > 10 {
			depth = 3
		}

		root, err := svc.Get(ctx, id)
		if err != nil {
			return errResult(toon.TicketErrCode(err), err.Error()), nil
		}

		loader := func(ctx context.Context, parentID string) ([]ticket.Ticket, error) {
			return svc.Children(ctx, parentID)
		}
		diagram := toon.GenerateDiagram(ctx, root, loader, depth)
		return mcp.NewToolResultText(fmt.Sprintf("```mermaid\n%s```", diagram)), nil
	}
}

// --- helpers ---

func encodeList(tickets []ticket.Ticket) string {
	if len(tickets) == 0 {
		return "TOON/1 []"
	}
	parts := make([]string, len(tickets))
	for i, t := range tickets {
		t := t
		parts[i] = toon.EncodeSummary(&t)
	}
	return "TOON/1 [" + strings.Join(parts, ",") + "]"
}

func errResult(code toon.ErrCode, msg string) *mcp.CallToolResult {
	return mcp.NewToolResultText(toon.EncodeError(code, msg))
}
