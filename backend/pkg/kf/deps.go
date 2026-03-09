package kf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// DepsGraph represents the track dependency adjacency list from deps.yaml.
// Keys are track IDs, values are lists of prerequisite track IDs.
type DepsGraph map[string][]string

// ReadDeps parses deps.yaml from a reader.
func ReadDeps(r io.Reader) (DepsGraph, error) {
	graph := make(DepsGraph)
	scanner := bufio.NewScanner(r)
	var currentID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Top-level key: "track-id:" or "track-id: []"
			key := strings.TrimSuffix(strings.TrimSpace(line), ":")
			key = strings.TrimSuffix(key, " []")
			if strings.HasSuffix(line, "[]") {
				graph[key] = nil
				currentID = ""
			} else {
				currentID = strings.TrimSuffix(strings.TrimSpace(line), ":")
				if _, ok := graph[currentID]; !ok {
					graph[currentID] = nil
				}
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "- ") && currentID != "" {
			dep := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			graph[currentID] = append(graph[currentID], dep)
		}
	}
	return graph, scanner.Err()
}

// ReadDepsFile reads deps.yaml from a file path.
func ReadDepsFile(path string) (DepsGraph, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(DepsGraph), nil
		}
		return nil, err
	}
	defer f.Close()
	return ReadDeps(f)
}

// WriteDeps writes deps.yaml in canonical format: header + sorted entries.
func WriteDeps(w io.Writer, graph DepsGraph) error {
	header := `# Track Dependency Graph
#
# PROTOCOL:
#   Canonical source for track dependency ordering (adjacency list).
#   Each key is a track ID; its value is a list of prerequisite track IDs.
#
# RULES:
#   - Only pending/in-progress tracks listed. Completed tracks pruned on cleanup.
#   - Architect appends entries when creating tracks.
#   - Developer checks deps before claiming: all deps must be completed.
#   - Cycles are forbidden.
#
# ORDERING: Entries sorted alphabetically by track ID. Dep lists sorted within each entry.
#
`
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}

	ids := make([]string, 0, len(graph))
	for id := range graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		deps := graph[id]
		if len(deps) == 0 {
			fmt.Fprintf(w, "%s: []\n\n", id)
		} else {
			sorted := make([]string, len(deps))
			copy(sorted, deps)
			sort.Strings(sorted)
			fmt.Fprintf(w, "%s:\n", id)
			for _, d := range sorted {
				fmt.Fprintf(w, "  - %s\n", d)
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}

// WriteDepsFile writes deps.yaml to a file path.
func WriteDepsFile(path string, graph DepsGraph) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteDeps(f, graph)
}

// GetDeps returns the dependency list for a track, or nil if not found.
func (g DepsGraph) GetDeps(trackID string) []string {
	return g[trackID]
}

// AddDep adds a dependency edge: trackID depends on depID.
func (g DepsGraph) AddDep(trackID, depID string) {
	deps := g[trackID]
	for _, d := range deps {
		if d == depID {
			return // already exists
		}
	}
	g[trackID] = append(deps, depID)
}

// RemoveDep removes a dependency edge.
func (g DepsGraph) RemoveDep(trackID, depID string) {
	deps := g[trackID]
	for i, d := range deps {
		if d == depID {
			g[trackID] = append(deps[:i], deps[i+1:]...)
			return
		}
	}
}

// RemoveTrack removes all entries for a track (both as key and as dependency value).
func (g DepsGraph) RemoveTrack(trackID string) {
	delete(g, trackID)
	// Also remove from other tracks' dep lists
	for id, deps := range g {
		for i, d := range deps {
			if d == trackID {
				g[id] = append(deps[:i], deps[i+1:]...)
				break
			}
		}
	}
}

// AllDepsSatisfied checks if all dependencies of trackID are in the completed set.
func (g DepsGraph) AllDepsSatisfied(trackID string, completedIDs map[string]bool) bool {
	deps := g[trackID]
	for _, d := range deps {
		if !completedIDs[d] {
			return false
		}
	}
	return true
}

// HasDeps returns true if the track has any dependencies.
func (g DepsGraph) HasDeps(trackID string) bool {
	return len(g[trackID]) > 0
}

// TopologicalSort returns track IDs in dependency order using Kahn's algorithm.
// Only tracks present in the candidates set are included in the output.
// Returns an error if a cycle is detected among the candidates.
func (g DepsGraph) TopologicalSort(candidates map[string]bool) ([]string, error) {
	// Build in-degree map scoped to candidates.
	inDegree := make(map[string]int, len(candidates))
	for id := range candidates {
		inDegree[id] = 0
	}
	for id := range candidates {
		for _, dep := range g[id] {
			if candidates[dep] {
				inDegree[id]++
			}
		}
	}

	// Seed queue with zero in-degree nodes.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // deterministic order

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// For each candidate that depends on node, decrement in-degree.
		for id := range candidates {
			for _, dep := range g[id] {
				if dep == node {
					inDegree[id]--
					if inDegree[id] == 0 {
						queue = append(queue, id)
						sort.Strings(queue) // maintain deterministic order
					}
					break
				}
			}
		}
	}

	if len(result) != len(candidates) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}
	return result, nil
}
