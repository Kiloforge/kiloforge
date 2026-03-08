package domain

import "time"

// BoardColumn defines the valid columns for the native track board.
const (
	ColumnBacklog    = "backlog"
	ColumnApproved   = "approved"
	ColumnInProgress = "in_progress"
	ColumnInReview   = "in_review"
	ColumnDone       = "done"
)

// BoardColumns is the ordered list of board columns.
var BoardColumns = []string{
	ColumnBacklog,
	ColumnApproved,
	ColumnInProgress,
	ColumnInReview,
	ColumnDone,
}

// ColumnOrder maps column names to their ordinal position.
var ColumnOrder = map[string]int{
	ColumnBacklog:    0,
	ColumnApproved:   1,
	ColumnInProgress: 2,
	ColumnInReview:   3,
	ColumnDone:       4,
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
