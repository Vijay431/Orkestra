package toon

// ErrCode represents a TOON error code.
type ErrCode string

const (
	ErrNotFound   ErrCode = "not_found"
	ErrConflict   ErrCode = "conflict"
	ErrInvalid    ErrCode = "invalid"
	ErrSeqBlocked ErrCode = "seq_blocked"
	ErrInternal   ErrCode = "internal"
)

// Priority order for BOARD display
var statusOrder = []string{"bk", "td", "ip", "bl", "cl", "dn"}

// Status colors for Mermaid diagrams
var statusColor = map[string]string{
	"bk": "#aaaaaa",
	"td": "#f0c040",
	"ip": "#40a0f0",
	"dn": "#40c040",
	"bl": "#f04040",
	"cl": "#cccccc",
}
