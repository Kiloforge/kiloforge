package kf

import (
	"testing"
)

func TestTopologicalSort_Empty(t *testing.T) {
	t.Parallel()
	graph := DepsGraph{}
	candidates := map[string]bool{}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestTopologicalSort_NoDeps(t *testing.T) {
	t.Parallel()
	graph := DepsGraph{
		"a": nil,
		"b": nil,
		"c": nil,
	}
	candidates := map[string]bool{"a": true, "b": true, "c": true}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	// Should be alphabetically sorted since no deps
	expected := []string{"a", "b", "c"}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("result[%d] = %q, want %q", i, result[i], v)
		}
	}
}

func TestTopologicalSort_LinearChain(t *testing.T) {
	t.Parallel()
	// c depends on b, b depends on a
	graph := DepsGraph{
		"a": nil,
		"b": {"a"},
		"c": {"b"},
	}
	candidates := map[string]bool{"a": true, "b": true, "c": true}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// a must come before b, b before c
	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}
	if indexOf("a") >= indexOf("b") {
		t.Error("a should come before b")
	}
	if indexOf("b") >= indexOf("c") {
		t.Error("b should come before c")
	}
}

func TestTopologicalSort_DiamondDeps(t *testing.T) {
	t.Parallel()
	// d depends on b and c, both depend on a
	graph := DepsGraph{
		"a": nil,
		"b": {"a"},
		"c": {"a"},
		"d": {"b", "c"},
	}
	candidates := map[string]bool{"a": true, "b": true, "c": true, "d": true}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	indexOf := func(s string) int {
		for i, v := range result {
			if v == s {
				return i
			}
		}
		return -1
	}
	if indexOf("a") >= indexOf("b") || indexOf("a") >= indexOf("c") {
		t.Error("a should come before b and c")
	}
	if indexOf("b") >= indexOf("d") || indexOf("c") >= indexOf("d") {
		t.Error("b and c should come before d")
	}
}

func TestTopologicalSort_CycleDetection(t *testing.T) {
	t.Parallel()
	graph := DepsGraph{
		"a": {"b"},
		"b": {"a"},
	}
	candidates := map[string]bool{"a": true, "b": true}

	_, err := graph.TopologicalSort(candidates)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestTopologicalSort_SubsetOfCandidates(t *testing.T) {
	t.Parallel()
	// Only include b and c as candidates; a is not a candidate
	// b depends on a (not a candidate), c depends on b
	graph := DepsGraph{
		"a": nil,
		"b": {"a"},
		"c": {"b"},
	}
	candidates := map[string]bool{"b": true, "c": true}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	// b should come before c (a is external, not counted as in-degree)
	if result[0] != "b" || result[1] != "c" {
		t.Errorf("expected [b, c], got %v", result)
	}
}

func TestTopologicalSort_IndependentTracks(t *testing.T) {
	t.Parallel()
	// Tracks with no deps among candidates
	graph := DepsGraph{
		"x": nil,
		"y": nil,
		"z": nil,
	}
	candidates := map[string]bool{"x": true, "y": true, "z": true}

	result, err := graph.TopologicalSort(candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	// Should be deterministic (alphabetical)
	if result[0] != "x" || result[1] != "y" || result[2] != "z" {
		t.Errorf("expected [x, y, z], got %v", result)
	}
}
