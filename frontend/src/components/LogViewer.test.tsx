import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LogViewer } from "./LogViewer";

beforeEach(() => {
  vi.restoreAllMocks();
});

function renderLogViewer(agentId = "agent-123") {
  const onClose = vi.fn();
  return { ...render(<LogViewer agentId={agentId} onClose={onClose} />), onClose };
}

describe("LogViewer", () => {
  it("shows Loading... initially", () => {
    vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    renderLogViewer();
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders log lines after fetch", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: ["line 1", "line 2", "line 3"] }),
    } as Response);

    renderLogViewer();
    await waitFor(() => {
      expect(screen.getByText(/line 1/)).toBeInTheDocument();
      expect(screen.getByText(/line 3/)).toBeInTheDocument();
    });
  });

  it("shows error message on fetch failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("network error"));
    renderLogViewer();
    await waitFor(() => {
      expect(screen.getByText("Failed to load log.")).toBeInTheDocument();
    });
  });

  it("shows agent id in header", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: [] }),
    } as Response);
    renderLogViewer("agent-xyz");
    expect(screen.getByText("agent-xyz")).toBeInTheDocument();
  });

  it("shows 'No log data available.' when lines are empty", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: [] }),
    } as Response);
    renderLogViewer();
    await waitFor(() => {
      expect(screen.getByText("No log data available.")).toBeInTheDocument();
    });
  });

  it("calls onClose when close button clicked", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: [] }),
    } as Response);
    const { onClose } = renderLogViewer();
    await user.click(screen.getByText("×"));
    expect(onClose).toHaveBeenCalled();
  });

  it("calls onClose when backdrop clicked", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: [] }),
    } as Response);
    const { onClose, container } = renderLogViewer();
    // Click the overlay
    await user.click(container.firstElementChild!);
    expect(onClose).toHaveBeenCalled();
  });

  it("renders follow toggle checkbox", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      json: () => Promise.resolve({ lines: [] }),
    } as Response);
    renderLogViewer();
    expect(screen.getByLabelText("Follow")).toBeInTheDocument();
  });
});
