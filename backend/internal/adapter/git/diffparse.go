package git

import (
	"fmt"
	"strconv"
	"strings"

	"kiloforge/internal/core/domain"
)

// ParseUnifiedDiff parses unified diff output from git into structured FileDiff objects.
func ParseUnifiedDiff(raw string) ([]domain.FileDiff, error) {
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")
	// Remove trailing empty line from final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var files []domain.FileDiff
	var current *domain.FileDiff
	var currentHunk *domain.Hunk
	var oldLineNo, newLineNo int

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Start of a new file diff.
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				files = append(files, *current)
			}
			current = &domain.FileDiff{Status: domain.FileStatusModified}
			currentHunk = nil
			parseDiffHeader(line, current)
			continue
		}

		if current == nil {
			continue
		}

		// Detect file metadata lines.
		switch {
		case strings.HasPrefix(line, "new file mode"):
			current.Status = domain.FileStatusAdded
		case strings.HasPrefix(line, "deleted file mode"):
			current.Status = domain.FileStatusDeleted
		case strings.HasPrefix(line, "rename from "):
			current.OldPath = strings.TrimPrefix(line, "rename from ")
			current.Status = domain.FileStatusRenamed
		case strings.HasPrefix(line, "rename to "):
			current.Path = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "similarity index"):
			// no-op, already handled via rename from/to
		case strings.HasPrefix(line, "Binary files"):
			current.IsBinary = true
		case strings.HasPrefix(line, "--- "):
			path := strings.TrimPrefix(line, "--- ")
			if path == "/dev/null" {
				current.Status = domain.FileStatusAdded
			} else if strings.HasPrefix(path, "a/") {
				oldPath := path[2:]
				if current.Status != domain.FileStatusRenamed {
					// Don't overwrite rename old_path
				} else if current.OldPath == "" {
					current.OldPath = oldPath
				}
			}
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimPrefix(line, "+++ ")
			if path == "/dev/null" {
				current.Status = domain.FileStatusDeleted
			} else if strings.HasPrefix(path, "b/") {
				if current.Status != domain.FileStatusRenamed || current.Path == "" {
					current.Path = path[2:]
				}
			}
		case strings.HasPrefix(line, "@@ "):
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("parse hunk header %q: %w", line, err)
			}
			current.Hunks = append(current.Hunks, hunk)
			currentHunk = &current.Hunks[len(current.Hunks)-1]
			oldLineNo = hunk.OldStart
			newLineNo = hunk.NewStart
		case strings.HasPrefix(line, `\ No newline at end of file`):
			// Skip this marker.
		case currentHunk != nil:
			dl := domain.DiffLine{}
			if len(line) == 0 {
				// Empty line in a hunk is a context line (space prefix stripped by git
				// for truly empty lines in the source).
				dl.Type = domain.DiffLineContext
				dl.Content = ""
				dl.OldNo = intP(oldLineNo)
				dl.NewNo = intP(newLineNo)
				oldLineNo++
				newLineNo++
				currentHunk.Lines = append(currentHunk.Lines, dl)
				continue
			}
			switch line[0] {
			case '+':
				dl.Type = domain.DiffLineAdd
				dl.Content = line[1:]
				dl.NewNo = intP(newLineNo)
				newLineNo++
				current.Insertions++
			case '-':
				dl.Type = domain.DiffLineDelete
				dl.Content = line[1:]
				dl.OldNo = intP(oldLineNo)
				oldLineNo++
				current.Deletions++
			case ' ':
				dl.Type = domain.DiffLineContext
				dl.Content = line[1:]
				dl.OldNo = intP(oldLineNo)
				dl.NewNo = intP(newLineNo)
				oldLineNo++
				newLineNo++
			default:
				continue
			}
			currentHunk.Lines = append(currentHunk.Lines, dl)
		}
	}

	if current != nil {
		files = append(files, *current)
	}

	return files, nil
}

// parseDiffHeader extracts file paths from the "diff --git a/X b/Y" line.
func parseDiffHeader(line string, fd *domain.FileDiff) {
	// Format: "diff --git a/path b/path"
	rest := strings.TrimPrefix(line, "diff --git ")
	// Split on " b/" — but we need to handle paths with spaces.
	// Simple approach: find " b/" by scanning from the right.
	idx := strings.LastIndex(rest, " b/")
	if idx < 0 {
		return
	}
	bPath := strings.TrimPrefix(rest[idx+1:], "b/")
	fd.Path = bPath
}

// parseHunkHeader parses "@@ -old,count +new,count @@ optional header text".
func parseHunkHeader(line string) (domain.Hunk, error) {
	h := domain.Hunk{Header: line}

	// Strip the leading "@@ " and find the closing " @@".
	inner := strings.TrimPrefix(line, "@@ ")
	end := strings.Index(inner, " @@")
	if end < 0 {
		return h, fmt.Errorf("missing closing @@")
	}
	inner = inner[:end]

	parts := strings.SplitN(inner, " ", 2)
	if len(parts) != 2 {
		return h, fmt.Errorf("expected two range specs, got %d", len(parts))
	}

	old := strings.TrimPrefix(parts[0], "-")
	new := strings.TrimPrefix(parts[1], "+")

	var err error
	h.OldStart, h.OldLines, err = parseRange(old)
	if err != nil {
		return h, fmt.Errorf("parse old range: %w", err)
	}
	h.NewStart, h.NewLines, err = parseRange(new)
	if err != nil {
		return h, fmt.Errorf("parse new range: %w", err)
	}

	return h, nil
}

// parseRange parses "start,count" or "start" (count defaults to 1).
func parseRange(s string) (start, count int, err error) {
	parts := strings.SplitN(s, ",", 2)
	start, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	} else {
		count = 1
	}
	return start, count, nil
}

func intP(v int) *int {
	return &v
}
