// SPDX-License-Identifier: MIT

package toon

import (
	"fmt"
	"strings"

	"github.com/vijay431/orkestra/internal/ticket"
)

const version = "TOON/1 "

// Encode converts a Ticket to its TOON string representation.
func Encode(t *ticket.Ticket) string {
	return version + encodeTicket(t, false)
}

// EncodePlain encodes without the TOON/1 version prefix (used inside arrays/boards).
func EncodePlain(t *ticket.Ticket) string {
	return encodeTicket(t, false)
}

// EncodeSummary encodes a compact ticket (no comments, no links) for board/list views.
func EncodeSummary(t *ticket.Ticket) string {
	return encodeTicket(t, true)
}

// EncodeError returns a TOON error envelope.
func EncodeError(code ErrCode, msg string) string {
	return version + fmt.Sprintf("ERR{code:%s,msg:%s}", code, escapeString(msg))
}

// EncodeOK returns a TOON success envelope.
func EncodeOK() string {
	return version + "{ok:true}"
}

// EncodeBoard returns a TOON BOARD{} grouped by status.
func EncodeBoard(board map[string][]ticket.Ticket) string {
	var sb strings.Builder
	sb.WriteString(version)
	sb.WriteString("BOARD{")

	first := true
	for _, status := range statusOrder {
		tickets, ok := board[status]
		if !ok || len(tickets) == 0 {
			continue
		}
		if !first {
			sb.WriteString(",")
		}
		first = false
		sb.WriteString(status)
		sb.WriteString(":[")
		for i, t := range tickets {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(encodeTicket(&t, true))
		}
		sb.WriteString("]")
	}
	sb.WriteString("}")
	return sb.String()
}

func encodeTicket(t *ticket.Ticket, summary bool) string {
	var sb strings.Builder
	sb.WriteString("T{")
	sb.WriteString("id:")
	sb.WriteString(t.ID)
	sb.WriteString(",t:")
	sb.WriteString(escapeString(t.Title))
	sb.WriteString(",s:")
	sb.WriteString(string(t.Status))
	sb.WriteString(",p:")
	sb.WriteString(string(t.Priority))
	sb.WriteString(",typ:")
	sb.WriteString(string(t.Type))

	if t.ExecMode != "" && t.ExecMode != ticket.ExecModeParallel {
		sb.WriteString(",em:")
		sb.WriteString(string(t.ExecMode))
	}
	if t.ExecOrder != nil {
		sb.WriteString(",ord:")
		sb.WriteString(fmt.Sprintf("%d", *t.ExecOrder))
	}
	if len(t.Labels) > 0 {
		sb.WriteString(",lbl:[")
		for i, l := range t.Labels {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(escapeString(l))
		}
		sb.WriteString("]")
	}
	if t.ParentID != "" {
		sb.WriteString(",par:")
		sb.WriteString(t.ParentID)
	}
	if len(t.Children) > 0 {
		sb.WriteString(",ch:[")
		sb.WriteString(strings.Join(t.Children, ","))
		sb.WriteString("]")
	}

	if !summary {
		if t.Description != "" {
			sb.WriteString(",d:")
			sb.WriteString(escapeString(t.Description))
		}
		if t.Assignee != "" {
			sb.WriteString(",as:")
			sb.WriteString(escapeString(t.Assignee))
		}
		if len(t.Comments) > 0 {
			sb.WriteString(",cmt:[")
			for i, c := range t.Comments {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(fmt.Sprintf("C{a:%s,t:%s,ts:%s}",
					escapeString(c.Author),
					escapeString(c.Body),
					c.CreatedAt.UTC().Format("2006-01-02T15:04")))
			}
			sb.WriteString("]")
		}
		if len(t.Links) > 0 {
			sb.WriteString(",lnk:[")
			for i, l := range t.Links {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(fmt.Sprintf("L{f:%s,t:%s,k:%s}", l.FromID, l.ToID, l.LinkType))
			}
			sb.WriteString("]")
		}
	}

	sb.WriteString(",ca:")
	sb.WriteString(t.CreatedAt.UTC().Format("2006-01-02"))
	sb.WriteString(",ua:")
	sb.WriteString(EtagOf(t))
	sb.WriteString("}")
	return sb.String()
}

// EtagOf returns the etag string for a ticket (clients pass this back on updates).
func EtagOf(t *ticket.Ticket) string {
	return t.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z")
}

// escapeString wraps a string in double quotes, escaping internal quotes and backslashes.
func escapeString(s string) string {
	if s == "" {
		return `""`
	}
	// Check if it needs quoting (contains spaces, colons, braces, commas, or quotes)
	needsQuote := false
	for _, c := range s {
		if c == ' ' || c == ':' || c == '{' || c == '}' || c == ',' || c == '"' || c == '\\' || c == '\n' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	var sb strings.Builder
	sb.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		default:
			sb.WriteRune(c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// TicketErrCode maps domain errors to TOON error codes.
func TicketErrCode(err error) ErrCode {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not_found"):
		return ErrNotFound
	case strings.Contains(msg, "conflict"):
		return ErrConflict
	case strings.Contains(msg, "seq_blocked"):
		return ErrSeqBlocked
	case strings.Contains(msg, "invalid"):
		return ErrInvalid
	default:
		return ErrInternal
	}
}
