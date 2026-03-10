import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { KanbanBoard } from "./KanbanBoard";
import type { BoardState } from "../types/api";

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
  it("renders all columns with labels", () => {
    renderBoard();
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
  });

  it("renders cards in correct columns", () => {
    renderBoard();
    expect(screen.getByText("Feature Alpha")).toBeInTheDocument();
    expect(screen.getByText("Bug Fix Beta")).toBeInTheDocument();
    expect(screen.getByText("Chore Gamma")).toBeInTheDocument();
  });

  it("shows column counts", () => {
    renderBoard();
    // Backlog has 1 card, in_progress has 1, done has 1
    const counts = screen.getAllByText("1");
    expect(counts.length).toBeGreaterThanOrEqual(3);
  });

  it("shows approve/reject buttons for backlog cards", () => {
    renderBoard();
    // Backlog cards get approve (checkmark) and reject (x) buttons
    // Use title attributes
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
    // Confirmation disappears, approve/reject buttons return
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
    // All counts should be 0
    const zeros = screen.getAllByText("0");
    expect(zeros.length).toBe(4);
  });

  it("renders card type badges", () => {
    renderBoard();
    expect(screen.getByText("feature")).toBeInTheDocument();
    expect(screen.getByText("fix")).toBeInTheDocument();
    expect(screen.getByText("chore")).toBeInTheDocument();
  });

  it("renders track links when projectSlug is provided", () => {
    renderBoard();
    const link = screen.getByText("Feature Alpha");
    expect(link.closest("a")).toHaveAttribute("href", "/projects/test-proj/tracks/track-1");
  });
});
