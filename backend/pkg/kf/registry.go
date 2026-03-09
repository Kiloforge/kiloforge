package kf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ReadRegistry parses tracks.yaml from the given reader.
// Format: each non-comment, non-blank line is "<track-id>: {json}".
func ReadRegistry(r io.Reader) ([]TrackEntry, error) {
	var entries []TrackEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry, err := parseRegistryLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse line %q: %w", line, err)
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

// ReadRegistryFile reads tracks.yaml from a file path.
func ReadRegistryFile(path string) ([]TrackEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadRegistry(f)
}

// WriteRegistry writes tracks.yaml in canonical format: header + sorted entries.
func WriteRegistry(w io.Writer, entries []TrackEntry) error {
	// Write header
	header := `# Kiloforge Track Registry
#
# FORMAT: <track-id>: {"title":"...","status":"...","type":"...","created":"...","updated":"..."}
# STATUS: pending | in-progress | completed | archived
# ORDER:  Lines sorted alphabetically by track ID. JSON fields in canonical order:
#         title, status, type, created, updated [, archived_at, archive_reason]
# TOOL:   Use ` + "`kf-track`" + ` to manage entries. Do not edit by hand.
#
`
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}

	// Sort by ID
	sorted := make([]TrackEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	for _, e := range sorted {
		line, err := formatRegistryLine(e)
		if err != nil {
			return fmt.Errorf("format entry %q: %w", e.ID, err)
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// WriteRegistryFile writes tracks.yaml to a file path.
func WriteRegistryFile(path string, entries []TrackEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteRegistry(f, entries)
}

// FindEntry returns the entry with the given ID, or nil if not found.
func FindEntry(entries []TrackEntry, id string) *TrackEntry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}

// ActiveEntries returns only pending and in-progress entries.
func ActiveEntries(entries []TrackEntry) []TrackEntry {
	var active []TrackEntry
	for _, e := range entries {
		if e.IsActive() {
			active = append(active, e)
		}
	}
	return active
}

// FilterByStatus returns entries matching the given status.
func FilterByStatus(entries []TrackEntry, status string) []TrackEntry {
	var result []TrackEntry
	for _, e := range entries {
		if e.Status == status {
			result = append(result, e)
		}
	}
	return result
}

func parseRegistryLine(line string) (TrackEntry, error) {
	idx := strings.Index(line, ": ")
	if idx < 0 {
		return TrackEntry{}, fmt.Errorf("no ': ' separator")
	}
	id := line[:idx]
	jsonStr := line[idx+2:]

	var entry TrackEntry
	if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
		return TrackEntry{}, err
	}
	entry.ID = id
	return entry, nil
}

func formatRegistryLine(e TrackEntry) (string, error) {
	// Build JSON with canonical field order
	fields := []jsonField{
		{"title", e.Title},
		{"status", e.Status},
		{"type", e.Type},
		{"created", e.Created},
		{"updated", e.Updated},
	}
	if e.ArchivedAt != "" {
		fields = append(fields, jsonField{"archived_at", e.ArchivedAt})
	}
	if e.ArchiveReason != "" {
		fields = append(fields, jsonField{"archive_reason", e.ArchiveReason})
	}
	jsonStr := canonicalJSON(fields)
	return e.ID + ": " + jsonStr, nil
}

type jsonField struct {
	Key   string
	Value string
}

func canonicalJSON(fields []jsonField) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(',')
		}
		val, _ := json.Marshal(f.Value)
		fmt.Fprintf(&b, "%q:%s", f.Key, val)
	}
	b.WriteByte('}')
	return b.String()
}
