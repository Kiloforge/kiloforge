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

// ReadConflicts parses conflicts.yaml from a reader.
// Format: each non-comment, non-blank line is "<idA>/<idB>: {json}".
func ReadConflicts(r io.Reader) ([]ConflictPair, error) {
	var pairs []ConflictPair
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pair, err := parseConflictLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse conflict line %q: %w", line, err)
		}
		pairs = append(pairs, pair)
	}
	return pairs, scanner.Err()
}

// ReadConflictsFile reads conflicts.yaml from a file path.
func ReadConflictsFile(path string) ([]ConflictPair, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return ReadConflicts(f)
}

// WriteConflicts writes conflicts.yaml in canonical format.
func WriteConflicts(w io.Writer, pairs []ConflictPair) error {
	header := `# Track Conflict Risk Pairs
#
# PROTOCOL:
#   Each line: <id-a>/<id-b>: {"risk":"high|medium|low","note":"...","added":"..."}
#   Pair key is strictly ordered: lower ID / higher ID (only one record per pair).
#
# RULES:
#   - Architect adds pairs when generating tracks that may conflict.
#   - Pairs auto-cleaned when either track completes or is archived.
#   - Only active (pending/in-progress) tracks should have pairs.
#
# TOOL: Use ` + "`kf-track conflicts`" + ` to manage entries. Do not edit by hand.
#
`
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}

	sorted := make([]ConflictPair, len(pairs))
	copy(sorted, pairs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PairKey() < sorted[j].PairKey()
	})

	for _, p := range sorted {
		jsonBytes, err := json.Marshal(struct {
			Risk  string `json:"risk"`
			Note  string `json:"note"`
			Added string `json:"added"`
		}{p.Risk, p.Note, p.Added})
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s: %s\n", p.PairKey(), jsonBytes)
	}
	return nil
}

// WriteConflictsFile writes conflicts.yaml to a file path.
func WriteConflictsFile(path string, pairs []ConflictPair) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteConflicts(f, pairs)
}

// NewConflictPair creates a pair with IDs in canonical order (lower first).
func NewConflictPair(idA, idB, risk, note string) ConflictPair {
	if idA > idB {
		idA, idB = idB, idA
	}
	return ConflictPair{
		TrackA: idA,
		TrackB: idB,
		Risk:   risk,
		Note:   note,
		Added:  TodayISO(),
	}
}

// FindConflicts returns all pairs involving the given track ID.
func FindConflicts(pairs []ConflictPair, trackID string) []ConflictPair {
	var result []ConflictPair
	for _, p := range pairs {
		if p.Involves(trackID) {
			result = append(result, p)
		}
	}
	return result
}

// RemoveConflictsForTrack removes all pairs involving the given track ID.
func RemoveConflictsForTrack(pairs []ConflictPair, trackID string) []ConflictPair {
	var result []ConflictPair
	for _, p := range pairs {
		if !p.Involves(trackID) {
			result = append(result, p)
		}
	}
	return result
}

// AddOrUpdateConflict adds a new pair or updates an existing one.
func AddOrUpdateConflict(pairs []ConflictPair, pair ConflictPair) []ConflictPair {
	key := pair.PairKey()
	for i, p := range pairs {
		if p.PairKey() == key {
			pairs[i] = pair
			return pairs
		}
	}
	return append(pairs, pair)
}

func parseConflictLine(line string) (ConflictPair, error) {
	idx := strings.Index(line, ": ")
	if idx < 0 {
		return ConflictPair{}, fmt.Errorf("no ': ' separator")
	}
	pairKey := line[:idx]
	jsonStr := line[idx+2:]

	slash := strings.Index(pairKey, "/")
	if slash < 0 {
		return ConflictPair{}, fmt.Errorf("no '/' in pair key %q", pairKey)
	}

	var pair ConflictPair
	pair.TrackA = pairKey[:slash]
	pair.TrackB = pairKey[slash+1:]

	if err := json.Unmarshal([]byte(jsonStr), &pair); err != nil {
		return ConflictPair{}, err
	}
	return pair, nil
}
