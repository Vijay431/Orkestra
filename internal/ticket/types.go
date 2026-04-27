package ticket

import (
	"errors"
	"time"
)

type Status string

const (
	StatusBacklog    Status = "bk"
	StatusTodo       Status = "td"
	StatusInProgress Status = "ip"
	StatusDone       Status = "dn"
	StatusBlocked    Status = "bl"
	StatusCancelled  Status = "cl"
)

type Priority string

const (
	PriorityCritical Priority = "cr"
	PriorityHigh     Priority = "h"
	PriorityMedium   Priority = "m"
	PriorityLow      Priority = "l"
)

type TicketType string

const (
	TypeBug     TicketType = "bug"
	TypeFeature TicketType = "ft"
	TypeTask    TicketType = "tsk"
	TypeEpic    TicketType = "ep"
	TypeChore   TicketType = "chr"
)

type ExecMode string

const (
	ExecModeParallel   ExecMode = "par"
	ExecModeSequential ExecMode = "seq"
)

type LinkType string

const (
	LinkBlocks     LinkType = "blk"
	LinkRelates    LinkType = "rel"
	LinkDuplicates LinkType = "dup"
)

var (
	ErrNotFound   = errors.New("not_found")
	ErrConflict   = errors.New("conflict")
	ErrSeqBlocked = errors.New("seq_blocked")
	ErrInvalid    = errors.New("invalid")
)

type Ticket struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	Type        TicketType `json:"type"`
	Description string     `json:"description,omitempty"`
	Assignee    string     `json:"assignee,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	Labels      []string   `json:"labels,omitempty"`
	ExecMode    ExecMode   `json:"exec_mode"`
	ExecOrder   *int       `json:"exec_order,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Populated on demand by GetTicket / GetChildren
	Children []string  `json:"children,omitempty"`
	Comments []Comment `json:"comments,omitempty"`
	Links    []Link    `json:"links,omitempty"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TicketID  string    `json:"ticket_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Link struct {
	FromID   string   `json:"from_id"`
	ToID     string   `json:"to_id"`
	LinkType LinkType `json:"link_type"`
}

type CreateInput struct {
	Title       string
	Type        TicketType
	Priority    Priority
	Description string
	Labels      []string
	ParentID    string
	ExecMode    ExecMode
	ExecOrder   *int
}

type UpdateInput struct {
	ID          string
	Etag        string // updated_at as RFC3339Nano — used for optimistic locking
	Title       *string
	Status      *Status
	Priority    *Priority
	Type        *TicketType
	Description *string
	Assignee    *string
	Labels      []string
	ExecMode    *ExecMode
	ExecOrder   *int
}

type ListFilter struct {
	Status          *Status
	Priority        *Priority
	Type            *TicketType
	Labels          []string
	Limit           int
	IncludeArchived bool
}
