package domain

// EscalatedItem represents a PR that has been escalated.
type EscalatedItem struct {
	Slug    string
	PR      int
	TrackID string
	Cycles  int
}
