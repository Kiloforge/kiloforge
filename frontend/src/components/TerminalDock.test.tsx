import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TerminalDock } from "./TerminalDock";
import type { WindowEntry } from "../hooks/useWindowManager";

const entry: WindowEntry = {
  agentId: "agent-123",
  name: "Dev Agent",
  role: "developer",
  initialX: 0,
  initialY: 0,
  minimized: true,
  unreadCount: 0,
  notificationType: null,
};

function renderDock(windows: WindowEntry[] = [entry]) {
  const props = {
    windows,
    onRestore: vi.fn(),
    onClose: vi.fn(),
  };
  return { ...render(<TerminalDock {...props} />), props };
}

describe("TerminalDock", () => {
  it("returns null when windows is empty", () => {
    const { container } = renderDock([]);
    expect(container.innerHTML).toBe("");
  });

  it("renders pill for each window", () => {
    renderDock([entry, { ...entry, agentId: "agent-456", name: "Review Agent" }]);
    expect(screen.getByText("Dev Agent")).toBeInTheDocument();
    expect(screen.getByText("Review Agent")).toBeInTheDocument();
  });

  it("shows truncated agentId when no name", () => {
    renderDock([{ ...entry, name: undefined }]);
    expect(screen.getByText("agent-12")).toBeInTheDocument();
  });

  it("calls onRestore when pill clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDock();
    await user.click(screen.getByText("Dev Agent"));
    expect(props.onRestore).toHaveBeenCalledWith("agent-123");
  });

  it("calls onClose when close button clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDock();
    await user.click(screen.getByText("×"));
    expect(props.onClose).toHaveBeenCalledWith("agent-123");
  });

  it("shows unread badge when count > 0", () => {
    renderDock([{ ...entry, unreadCount: 5 }]);
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("shows 99+ for large unread counts", () => {
    renderDock([{ ...entry, unreadCount: 150 }]);
    expect(screen.getByText("99+")).toBeInTheDocument();
  });

  it("does not show badge when unreadCount is 0", () => {
    renderDock([{ ...entry, unreadCount: 0 }]);
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });
});
