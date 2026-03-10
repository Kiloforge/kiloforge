//go:build e2e

package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

// seedBoardData creates a project and populates the board with cards in various columns.
func seedBoardData(t *testing.T, srv *e2eServer) {
	t.Helper()

	_ = srv.projects.Add(domain.Project{
		Slug:     "board-project",
		RepoName: "board-project",
	})

	board := domain.NewBoardState()
	now := time.Now()
	board.Cards = map[string]domain.BoardCard{
		"track-backlog-1": {
			TrackID:   "track-backlog-1",
			Title:     "Backlog Task One",
			Type:      "feature",
			Column:    domain.ColumnBacklog,
			Position:  0,
			MovedAt:   now,
			CreatedAt: now,
		},
		"track-backlog-2": {
			TrackID:   "track-backlog-2",
			Title:     "Backlog Task Two",
			Type:      "chore",
			Column:    domain.ColumnBacklog,
			Position:  1,
			MovedAt:   now,
			CreatedAt: now,
		},
		"track-inprogress-1": {
			TrackID:   "track-inprogress-1",
			Title:     "In Progress Feature",
			Type:      "feature",
			Column:    domain.ColumnInProgress,
			Position:  0,
			AgentID:   "agent-001",
			PRNumber:  42,
			MovedAt:   now,
			CreatedAt: now,
		},
		"track-done-1": {
			TrackID:   "track-done-1",
			Title:     "Completed Work",
			Type:      "feature",
			Column:    domain.ColumnDone,
			Position:  0,
			MovedAt:   now,
			CreatedAt: now,
		},
	}
	if err := srv.boardStore.SaveBoard("board-project", board); err != nil {
		t.Fatalf("save board: %v", err)
	}
}

// --- Phase 1: Board Display Tests ---

func TestE2E_KanbanBoard_GetEmptyBoard(t *testing.T) {
	srv := startE2EServerWithBoard(t)

	_ = srv.projects.Add(domain.Project{
		Slug:     "empty-project",
		RepoName: "empty-project",
	})

	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/empty-project", &board)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	columns, ok := board["columns"].([]any)
	if !ok {
		t.Fatal("expected columns array")
	}
	if len(columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(columns))
	}

	expected := []string{"backlog", "approved", "in_progress", "done"}
	for i, col := range columns {
		if col != expected[i] {
			t.Errorf("column %d: expected %q, got %v", i, expected[i], col)
		}
	}
}

func TestE2E_KanbanBoard_GetBoardWithCards(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cards, ok := board["cards"].(map[string]any)
	if !ok {
		t.Fatal("expected cards map")
	}
	if len(cards) != 4 {
		t.Errorf("expected 4 cards, got %d", len(cards))
	}

	// Verify each card is in the correct column.
	checks := map[string]string{
		"track-backlog-1":    "backlog",
		"track-backlog-2":    "backlog",
		"track-inprogress-1": "in_progress",
		"track-done-1":       "done",
	}
	for trackID, expectedCol := range checks {
		card, exists := cards[trackID].(map[string]any)
		if !exists {
			t.Errorf("card %q not found", trackID)
			continue
		}
		if card["column"] != expectedCol {
			t.Errorf("card %q: expected column %q, got %v", trackID, expectedCol, card["column"])
		}
	}
}

func TestE2E_KanbanBoard_CardHasRequiredFields(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer resp.Body.Close()

	cards := board["cards"].(map[string]any)
	card := cards["track-backlog-1"].(map[string]any)

	requiredFields := []string{"track_id", "title", "column", "moved_at", "created_at"}
	for _, field := range requiredFields {
		if card[field] == nil || card[field] == "" {
			t.Errorf("card missing required field %q", field)
		}
	}
	if card["title"] != "Backlog Task One" {
		t.Errorf("expected title 'Backlog Task One', got %v", card["title"])
	}
}

// --- Phase 2: Card Movement Tests ---

func TestE2E_KanbanBoard_MoveCardForward(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	body, _ := json.Marshal(map[string]string{
		"track_id":  "track-backlog-1",
		"to_column": "in_progress",
	})
	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if result["track_id"] != "track-backlog-1" {
		t.Errorf("expected track_id track-backlog-1, got %v", result["track_id"])
	}
	if result["from_column"] != "backlog" {
		t.Errorf("expected from_column backlog, got %v", result["from_column"])
	}
	if result["to_column"] != "in_progress" {
		t.Errorf("expected to_column in_progress, got %v", result["to_column"])
	}

	// Verify board state reflects the move.
	var board map[string]any
	getResp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer getResp.Body.Close()

	cards := board["cards"].(map[string]any)
	card := cards["track-backlog-1"].(map[string]any)
	if card["column"] != "in_progress" {
		t.Errorf("card should be in in_progress after move, got %v", card["column"])
	}
}

func TestE2E_KanbanBoard_MoveCardBackward(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	body, _ := json.Marshal(map[string]string{
		"track_id":  "track-inprogress-1",
		"to_column": "backlog",
	})
	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["from_column"] != "in_progress" {
		t.Errorf("expected from_column in_progress, got %v", result["from_column"])
	}
	if result["to_column"] != "backlog" {
		t.Errorf("expected to_column backlog, got %v", result["to_column"])
	}
}

func TestE2E_KanbanBoard_MoveToInvalidColumn(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	body, _ := json.Marshal(map[string]string{
		"track_id":  "track-backlog-1",
		"to_column": "nonexistent",
	})
	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid column, got %d", resp.StatusCode)
	}
}

func TestE2E_KanbanBoard_MoveNonexistentCard(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	body, _ := json.Marshal(map[string]string{
		"track_id":  "nonexistent-track",
		"to_column": "in_progress",
	})
	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for nonexistent card, got %d", resp.StatusCode)
	}
}

func TestE2E_KanbanBoard_MoveSameColumn(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	body, _ := json.Marshal(map[string]string{
		"track_id":  "track-backlog-1",
		"to_column": "backlog",
	})
	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	// Moving to same column should succeed (no-op).
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for same-column move, got %d", resp.StatusCode)
	}
}

func TestE2E_KanbanBoard_ColumnCountsAfterMove(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	// Count backlog cards before move.
	var before map[string]any
	r1 := e2eGetJSON(t, srv.URL+"/api/board/board-project", &before)
	defer r1.Body.Close()

	backlogBefore := countCardsInColumn(before, "backlog")
	inProgressBefore := countCardsInColumn(before, "in_progress")

	// Move a card.
	body, _ := json.Marshal(map[string]string{
		"track_id":  "track-backlog-1",
		"to_column": "in_progress",
	})
	moveResp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	moveResp.Body.Close()

	// Count after move.
	var after map[string]any
	r2 := e2eGetJSON(t, srv.URL+"/api/board/board-project", &after)
	defer r2.Body.Close()

	backlogAfter := countCardsInColumn(after, "backlog")
	inProgressAfter := countCardsInColumn(after, "in_progress")

	if backlogAfter != backlogBefore-1 {
		t.Errorf("backlog count: expected %d, got %d", backlogBefore-1, backlogAfter)
	}
	if inProgressAfter != inProgressBefore+1 {
		t.Errorf("in_progress count: expected %d, got %d", inProgressBefore+1, inProgressAfter)
	}
}

// --- Phase 3: Board Sync Tests ---

func TestE2E_KanbanBoard_SyncBoard_NoProject(t *testing.T) {
	srv := startE2EServerWithBoard(t)

	resp, err := http.Post(srv.URL+"/api/board/nonexistent/sync", "application/json", nil)
	if err != nil {
		t.Fatalf("POST sync: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for nonexistent project, got %d", resp.StatusCode)
	}
}

func TestE2E_KanbanBoard_BoardNotConfigured(t *testing.T) {
	srv := startE2EServer(t) // No board service

	resp, err := http.Get(srv.URL + "/api/board/any-project")
	if err != nil {
		t.Fatalf("GET board: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when board not configured, got %d", resp.StatusCode)
	}
}

// --- Phase 4: Card Content Tests ---

func TestE2E_KanbanBoard_CardWithOptionalFields(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer resp.Body.Close()

	cards := board["cards"].(map[string]any)

	// Card with agent_id set.
	ipCard := cards["track-inprogress-1"].(map[string]any)
	if ipCard["agent_id"] != "agent-001" {
		t.Errorf("expected agent_id agent-001, got %v", ipCard["agent_id"])
	}

	// Card with pr_number set.
	prNum, ok := ipCard["pr_number"].(float64) // JSON numbers are float64
	if !ok || int(prNum) != 42 {
		t.Errorf("expected pr_number 42, got %v", ipCard["pr_number"])
	}

	// Card type is present.
	if ipCard["type"] != "feature" {
		t.Errorf("expected type feature, got %v", ipCard["type"])
	}
}

func TestE2E_KanbanBoard_CardTitleAndID(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer resp.Body.Close()

	cards := board["cards"].(map[string]any)

	for trackID, raw := range cards {
		card := raw.(map[string]any)
		if card["track_id"] != trackID {
			t.Errorf("card key %q doesn't match track_id %v", trackID, card["track_id"])
		}
		title, ok := card["title"].(string)
		if !ok || title == "" {
			t.Errorf("card %q has empty or missing title", trackID)
		}
	}
}

// --- Phase 5: Edge and Failure Cases ---

func TestE2E_KanbanBoard_RapidMoves(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	// Move the same card through multiple columns rapidly.
	moves := []string{"approved", "in_progress", "done"}
	for _, col := range moves {
		body, _ := json.Marshal(map[string]string{
			"track_id":  "track-backlog-1",
			"to_column": col,
		})
		resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST move to %s: %v", col, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("move to %s failed with %d", col, resp.StatusCode)
		}
	}

	// Verify final state.
	var board map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/board-project", &board)
	defer resp.Body.Close()

	cards := board["cards"].(map[string]any)
	card := cards["track-backlog-1"].(map[string]any)
	if card["column"] != "done" {
		t.Errorf("after rapid moves, expected done, got %v", card["column"])
	}
}

func TestE2E_KanbanBoard_MoveWithEmptyBody(t *testing.T) {
	srv := startE2EServerWithBoard(t)
	seedBoardData(t, srv)

	resp, err := http.Post(srv.URL+"/api/board/board-project/move", "application/json", nil)
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", resp.StatusCode)
	}
}

func TestE2E_KanbanBoard_AllCardsInOneColumn(t *testing.T) {
	srv := startE2EServerWithBoard(t)

	_ = srv.projects.Add(domain.Project{
		Slug:     "all-done-project",
		RepoName: "all-done-project",
	})

	board := domain.NewBoardState()
	now := time.Now()
	for i := 0; i < 5; i++ {
		id := domain.BoardCard{
			TrackID:   fmt.Sprintf("track-done-%d", i),
			Title:     fmt.Sprintf("Done Task %d", i),
			Column:    domain.ColumnDone,
			Position:  i,
			MovedAt:   now,
			CreatedAt: now,
		}
		board.Cards[id.TrackID] = id
	}
	if err := srv.boardStore.SaveBoard("all-done-project", board); err != nil {
		t.Fatalf("save board: %v", err)
	}

	var result map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/board/all-done-project", &result)
	defer resp.Body.Close()

	cards := result["cards"].(map[string]any)
	if len(cards) != 5 {
		t.Errorf("expected 5 cards, got %d", len(cards))
	}
	for _, raw := range cards {
		card := raw.(map[string]any)
		if card["column"] != "done" {
			t.Errorf("expected all cards in done, got %v", card["column"])
		}
	}
}

// countCardsInColumn counts cards in a specific column from a board API response.
func countCardsInColumn(board map[string]any, column string) int {
	cards, ok := board["cards"].(map[string]any)
	if !ok {
		return 0
	}
	count := 0
	for _, raw := range cards {
		card, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if card["column"] == column {
			count++
		}
	}
	return count
}
