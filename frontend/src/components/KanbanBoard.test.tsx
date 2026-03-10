import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { KanbanBoard } from "./KanbanBoard";
import type { BoardState, TrackDependency } from "../types/api";

vi.mock("./tour/TourProvider", () => ({
  useTourContextSafe: () => null,
}));

const mockBoard: BoardState = {
  columns: ["backlog", "approved", "in_progress", "done"],
  cards: {
    "track-1": {
      track_id: "track-1",
      title: "Feature Alpha",
      type: "feature",
      column: "backlog",
      position: 0,
      moved_at: "2026-03-10T00:00:00Z",
      created_at: "2026-03-10T00:00:00Z",
    },
    "track-2": {
      track_id: "track-2",
      title: "Bug Fix Beta",
      type: "fix",
      column: "in_progress",
      position: 0,
      moved_at: "2026-03-10T00:00:00Z",
      created_at: "2026-03-10T00:00:00Z",
    },
    "track-3": {
      track_id: "track-3",
      title: "Chore Gamma",
      type: "chore",
      column: "done",
      position: 0,
      agent_status: "completed",
      moved_at: "2026-03-10T00:00:00Z",
      created_at: "2026-03-10T00:00:00Z",
    },
  },
};

function renderBoard(props?: Partial<Parameters<typeof KanbanBoard>[0]>) {
  const defaultProps = {
    board: mockBoard,
    projectSlug: "test-proj",
    onMoveCard: vi.fn(),
    onDeleteTrack: vi.fn(),
    ...props,
  };

  return render(
    <MemoryRouter>
      <KanbanBoard {...defaultProps} />
    </MemoryRouter>,
  );
}

describe("KanbanBoard", () => {
  it("renders 3 visible columns (backlog, approved, in_progress)", () => {
    renderBoard();
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.queryByText("Done")).not.toBeInTheDocument();
  });

  it("hides cards in the done column from the board", () => {
    renderBoard();
    expect(screen.getByText("Feature Alpha")).toBeInTheDocument();
    expect(screen.getByText("Bug Fix Beta")).toBeInTheDocument();
    // Chore Gamma is in "done" — should not be visible
    expect(screen.queryByText("Chore Gamma")).not.toBeInTheDocument();
  });

  it("renders cards in correct columns", () => {
    renderBoard();
    expect(screen.getByText("Feature Alpha")).toBeInTheDocument();
    expect(screen.getByText("Bug Fix Beta")).toBeInTheDocument();
  });

  it("shows column counts", () => {
    renderBoard();
    // Backlog has 1 card, approved has 0, in_progress has 1
    const ones = screen.getAllByText("1");
    expect(ones.length).toBeGreaterThanOrEqual(2);
    const zeros = screen.getAllByText("0");
    expect(zeros.length).toBeGreaterThanOrEqual(1);
  });

  it("shows approve/reject buttons for backlog cards", () => {
    renderBoard();
    expect(screen.getByTitle("Approve")).toBeInTheDocument();
    expect(screen.getByTitle("Reject")).toBeInTheDocument();
  });

  it("shows confirmation dialog when reject is clicked", async () => {
    const user = userEvent.setup();
    renderBoard();
    await user.click(screen.getByTitle("Reject"));
    expect(screen.getByText("Delete track?")).toBeInTheDocument();
    expect(screen.getByText("Yes")).toBeInTheDocument();
    expect(screen.getByText("No")).toBeInTheDocument();
  });

  it("calls onDeleteTrack when reject is confirmed", async () => {
    const user = userEvent.setup();
    const onDeleteTrack = vi.fn();
    renderBoard({ onDeleteTrack });
    await user.click(screen.getByTitle("Reject"));
    await user.click(screen.getByText("Yes"));
    expect(onDeleteTrack).toHaveBeenCalledWith("track-1");
  });

  it("cancels reject when No is clicked", async () => {
    const user = userEvent.setup();
    renderBoard();
    await user.click(screen.getByTitle("Reject"));
    await user.click(screen.getByText("No"));
    expect(screen.queryByText("Delete track?")).not.toBeInTheDocument();
    expect(screen.getByTitle("Approve")).toBeInTheDocument();
  });

  it("calls onMoveCard when approve is clicked", async () => {
    const user = userEvent.setup();
    const onMoveCard = vi.fn();
    renderBoard({ onMoveCard });
    await user.click(screen.getByTitle("Approve"));
    expect(onMoveCard).toHaveBeenCalledWith("track-1", "approved");
  });

  it("renders empty board with no cards", () => {
    const emptyBoard: BoardState = {
      columns: ["backlog", "approved", "in_progress", "done"],
      cards: {},
    };
    renderBoard({ board: emptyBoard });
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    // 3 visible columns, all with 0 count
    const zeros = screen.getAllByText("0");
    expect(zeros.length).toBe(3);
  });

  it("renders card type badges", () => {
    renderBoard();
    expect(screen.getByText("feature")).toBeInTheDocument();
    expect(screen.getByText("fix")).toBeInTheDocument();
    // "chore" is in done column — not visible
    expect(screen.queryByText("chore")).not.toBeInTheDocument();
  });

  it("renders track links when projectSlug is provided", () => {
    renderBoard();
    const link = screen.getByText("Feature Alpha");
    expect(link.closest("a")).toHaveAttribute("href", "/projects/test-proj/tracks/track-1");
  });

  it("shows ready badge on backlog card with no dependencies", () => {
    renderBoard({ dependencies: new Map() });
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("shows ready badge on backlog card with all deps completed", () => {
    const deps = new Map<string, TrackDependency[]>([
      ["track-1", [{ id: "dep-1", title: "Dep", status: "completed" }]],
    ]);
    renderBoard({ dependencies: deps });
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("does not show ready badge on backlog card with unmet deps", () => {
    const deps = new Map<string, TrackDependency[]>([
      ["track-1", [{ id: "dep-1", title: "Dep", status: "pending" }]],
    ]);
    renderBoard({ dependencies: deps });
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
  });

  it("does not show ready badge on non-backlog cards", () => {
    renderBoard({ dependencies: new Map() });
    // track-2 is in_progress, track-3 is done — neither should have Ready badge
    const track2Card = screen.getByText("Bug Fix Beta").closest("[data-track-id]");
    expect(track2Card?.textContent).not.toContain("Ready");
  });

  it("applies entering animation class only to newly added cards", () => {
    const { rerender } = render(
      <MemoryRouter>
        <KanbanBoard
          board={mockBoard}
          projectSlug="test-proj"
          onMoveCard={vi.fn()}
          onDeleteTrack={vi.fn()}
        />
      </MemoryRouter>,
    );

    // Add a new card to the board
    const updatedBoard: BoardState = {
      ...mockBoard,
      cards: {
        ...mockBoard.cards,
        "track-new": {
          track_id: "track-new",
          title: "New Card",
          type: "feature",
          column: "approved",
          position: 0,
          moved_at: "2026-03-11T00:00:00Z",
          created_at: "2026-03-11T00:00:00Z",
        },
      },
    };

    rerender(
      <MemoryRouter>
        <KanbanBoard
          board={updatedBoard}
          projectSlug="test-proj"
          onMoveCard={vi.fn()}
          onDeleteTrack={vi.fn()}
        />
      </MemoryRouter>,
    );

    // The new card should have the entering animation class
    const newCard = screen.getByText("New Card").closest("[data-track-id]");
    expect(newCard?.className).toMatch(/cardEntering/);

    // Existing cards should NOT have the entering animation class
    const existingCard = screen.getByText("Feature Alpha").closest("[data-track-id]");
    expect(existingCard?.className).not.toMatch(/cardEntering/);
  });

  it("shows completion animation when card moves to done", () => {
    const initialBoard: BoardState = {
      columns: ["backlog", "approved", "in_progress", "done"],
      cards: {
        "track-completing": {
          track_id: "track-completing",
          title: "Completing Card",
          type: "feature",
          column: "in_progress",
          position: 0,
          moved_at: "2026-03-10T00:00:00Z",
          created_at: "2026-03-10T00:00:00Z",
        },
      },
    };

    const { rerender } = render(
      <MemoryRouter>
        <KanbanBoard
          board={initialBoard}
          projectSlug="test-proj"
          onMoveCard={vi.fn()}
          onDeleteTrack={vi.fn()}
        />
      </MemoryRouter>,
    );

    // Card should be visible initially
    expect(screen.getByText("Completing Card")).toBeInTheDocument();

    // Move card to done
    const doneBoard: BoardState = {
      ...initialBoard,
      cards: {
        "track-completing": {
          ...initialBoard.cards["track-completing"],
          column: "done",
        },
      },
    };

    rerender(
      <MemoryRouter>
        <KanbanBoard
          board={doneBoard}
          projectSlug="test-proj"
          onMoveCard={vi.fn()}
          onDeleteTrack={vi.fn()}
        />
      </MemoryRouter>,
    );

    // Card should still be visible temporarily with completion animation
    const completingCard = screen.getByText("Completing Card").closest("[data-track-id]");
    expect(completingCard?.className).toMatch(/cardCompleting/);
  });
});
