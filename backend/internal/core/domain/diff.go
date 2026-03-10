package domain

// DiffLineType classifies a line in a unified diff hunk.
type DiffLineType string

const (
	DiffLineContext DiffLineType = "context"
	DiffLineAdd     DiffLineType = "add"
	DiffLineDelete  DiffLineType = "delete"
)

// DiffLine represents a single line within a diff hunk.
type DiffLine struct {
	Type    DiffLineType `json:"type"`
	Content string       `json:"content"`
	OldNo   *int         `json:"old_no"`
	NewNo   *int         `json:"new_no"`
}

// Hunk represents a contiguous block of changes within a file diff.
type Hunk struct {
	OldStart int        `json:"old_start"`
	OldLines int        `json:"old_lines"`
	NewStart int        `json:"new_start"`
	NewLines int        `json:"new_lines"`
	Header   string     `json:"header"`
	Lines    []DiffLine `json:"lines"`
}

// FileStatus describes the type of change to a file.
type FileStatus string

const (
	FileStatusAdded    FileStatus = "added"
	FileStatusModified FileStatus = "modified"
	FileStatusDeleted  FileStatus = "deleted"
	FileStatusRenamed  FileStatus = "renamed"
)

// FileDiff represents the diff for a single file.
type FileDiff struct {
	Path       string     `json:"path"`
	OldPath    string     `json:"old_path,omitempty"`
	Status     FileStatus `json:"status"`
	Insertions int        `json:"insertions"`
	Deletions  int        `json:"deletions"`
	IsBinary   bool       `json:"is_binary"`
	Hunks      []Hunk     `json:"hunks"`
}

// DiffStats contains summary statistics for a diff.
type DiffStats struct {
	FilesChanged int `json:"files_changed"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// DiffResult contains the complete structured diff between two branches.
type DiffResult struct {
	Branch    string     `json:"branch"`
	Base      string     `json:"base"`
	Stats     DiffStats  `json:"stats"`
	Files     []FileDiff `json:"files"`
	Truncated bool       `json:"truncated,omitempty"`
}

// BranchInfo describes an active worktree branch with its agent context.
type BranchInfo struct {
	Branch  string `json:"branch"`
	AgentID string `json:"agent_id,omitempty"`
	TrackID string `json:"track_id,omitempty"`
	Status  string `json:"status"`
}
