package domain

import "time"

// BoardColumn defines the valid columns for the native track board.
const (
	ColumnBacklog    = "backlog"
	ColumnApproved   = "approved"
	ColumnInProgress = "in_progress"
	ColumnDone       = "done"
)

// BoardColumns is the ordered list of board columns.
var BoardColumns = []string{
	ColumnBacklog,
	ColumnApproved,
	ColumnInProgress,
	ColumnDone,
}

// ColumnOrder maps column names to their ordinal position.
var ColumnOrder = map[string]int{
	ColumnBacklog:    0,
	ColumnApproved:   1,
	ColumnInProgress: 2,
	ColumnDone:       3,
}

// IsValidColumn returns true if the column name is valid.
func IsValidColumn(col string) bool {
	_, ok := ColumnOrder[col]
	return ok
}

// IsBackwardMove returns true if moving from fromCol to toCol is a demotion.
func IsBackwardMove(fromCol, toCol string) bool {
	from, ok1 := ColumnOrder[fromCol]
	to, ok2 := ColumnOrder[toCol]
	return ok1 && ok2 && to < from
}

// IsForwardMove returns true if moving from fromCol to toCol is a promotion.
func IsForwardMove(fromCol, toCol string) bool {
	from, ok1 := ColumnOrder[fromCol]
	to, ok2 := ColumnOrder[toCol]
	return ok1 && ok2 && to > from
}

// BoardCard represents a track card on the native board.
type BoardCard struct {
	TrackID        string    `json:"track_id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	Column         string    `json:"column"`
	Position       int       `json:"position"`
	AgentID        string    `json:"agent_id,omitempty"`
	AgentStatus    string    `json:"agent_status,omitempty"`
	AssignedWorker string    `json:"assigned_worker,omitempty"`
	PRNumber       int       `json:"pr_number,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	MovedAt        time.Time `json:"moved_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// BoardState is the full board state for a project.
type BoardState struct {
	Columns []string             `json:"columns"`
	Cards   map[string]BoardCard `json:"cards"`
}

// NewBoardState creates an empty board state with standard columns.
func NewBoardState() *BoardState {
	return &BoardState{
		Columns: BoardColumns,
		Cards:   make(map[string]BoardCard),
	}
}

// ClampForwardMove restricts manual forward moves to at most one step, and
// never beyond "approved". If the move is not forward, it returns toCol
// unchanged. This is a pure function with no side effects.
func ClampForwardMove(fromCol, toCol string) string {
	from, ok1 := ColumnOrder[fromCol]
	to, ok2 := ColumnOrder[toCol]
	if !ok1 || !ok2 || to <= from {
		return toCol // not forward or invalid — pass through
	}
	// Forward move: max manual target is "approved".
	maxTarget := ColumnOrder[ColumnApproved]
	if from >= maxTarget {
		// Already at or beyond approved — no manual forward move allowed.
		return fromCol
	}
	// Clamp to at most one step ahead, capped at approved.
	clamped := from + 1
	if clamped > maxTarget {
		clamped = maxTarget
	}
	// Map ordinal back to column name.
	for col, ord := range ColumnOrder {
		if ord == clamped {
			return col
		}
	}
	return fromCol // fallback (shouldn't happen)
}

// CardsByColumn returns cards in the given column, sorted by position.
func (b *BoardState) CardsByColumn(col string) []BoardCard {
	var cards []BoardCard
	for _, c := range b.Cards {
		if c.Column == col {
			cards = append(cards, c)
		}
	}
	return cards
}
