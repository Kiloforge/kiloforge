import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueuePanel } from "./QueuePanel";
import type { QueueStatus } from "../types/api";

const stoppedQueue: QueueStatus = {
  running: false,
  max_workers: 3,
  active_workers: 0,
  items: [],
};

const runningQueue: QueueStatus = {
  running: true,
  max_workers: 3,
  active_workers: 2,
  items: [
    {
      track_id: "track-abc",
      project_slug: "proj",
      status: "assigned",
      agent_id: "developer-1",
      enqueued_at: "2026-03-10T00:00:00Z",
      assigned_at: "2026-03-10T00:01:00Z",
      completed_at: null,
    },
    {
      track_id: "track-def",
      project_slug: "proj",
      status: "assigned",
      agent_id: "developer-2",
      enqueued_at: "2026-03-10T00:00:00Z",
      assigned_at: "2026-03-10T00:02:00Z",
      completed_at: null,
    },
    {
      track_id: "track-ghi",
      project_slug: "proj",
      status: "queued",
      enqueued_at: "2026-03-10T00:03:00Z",
      assigned_at: null,
      completed_at: null,
    },
    {
      track_id: "track-jkl",
      project_slug: "proj",
      status: "queued",
      enqueued_at: "2026-03-10T00:04:00Z",
      assigned_at: null,
      completed_at: null,
    },
  ],
};

describe("QueuePanel", () => {
  const defaultProps = {
    queue: null as QueueStatus | null,
    loading: false,
    starting: false,
    stopping: false,
    updatingSettings: false,
    onStart: vi.fn(),
    onStop: vi.fn(),
    onUpdateSettings: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    render(<QueuePanel {...defaultProps} loading={true} />);
    expect(screen.getByText("Loading queue...")).toBeInTheDocument();
  });

  it("shows empty state when queue is null", () => {
    render(<QueuePanel {...defaultProps} />);
    expect(screen.getByText(/Queue not configured/)).toBeInTheDocument();
  });

  it("renders Start button when queue is stopped", () => {
    render(<QueuePanel {...defaultProps} queue={stoppedQueue} />);
    expect(screen.getByText("Start")).toBeInTheDocument();
  });

  it("renders Stop button when queue is running", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} />);
    expect(screen.getByText("Stop")).toBeInTheDocument();
  });

  it("calls onStart when Start is clicked", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    render(<QueuePanel {...defaultProps} queue={stoppedQueue} onStart={onStart} />);
    await user.click(screen.getByText("Start"));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("calls onStop when Stop is clicked", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    render(<QueuePanel {...defaultProps} queue={runningQueue} onStop={onStop} />);
    await user.click(screen.getByText("Stop"));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("disables Start button while starting", () => {
    render(<QueuePanel {...defaultProps} queue={stoppedQueue} starting={true} />);
    expect(screen.getByText("Starting...")).toBeDisabled();
  });

  it("disables Stop button while stopping", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} stopping={true} />);
    expect(screen.getByText("Stopping...")).toBeDisabled();
  });

  it("shows worker count and stats", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} />);
    expect(screen.getByText("2 / 3")).toBeInTheDocument(); // active / max
    expect(screen.getByText("2 queued")).toBeInTheDocument();
  });

  it("shows active workers list", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} />);
    expect(screen.getByText("developer-1")).toBeInTheDocument();
    expect(screen.getByText("developer-2")).toBeInTheDocument();
    expect(screen.getByText("track-abc")).toBeInTheDocument();
    expect(screen.getByText("track-def")).toBeInTheDocument();
  });

  it("shows queued tracks list", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} />);
    expect(screen.getByText("track-ghi")).toBeInTheDocument();
    expect(screen.getByText("track-jkl")).toBeInTheDocument();
  });

  it("shows max workers input with current value", () => {
    render(<QueuePanel {...defaultProps} queue={runningQueue} />);
    const input = screen.getByDisplayValue("3");
    expect(input).toBeInTheDocument();
  });

  it("calls onUpdateSettings when max workers is changed", async () => {
    const user = userEvent.setup();
    const onUpdateSettings = vi.fn();
    render(<QueuePanel {...defaultProps} queue={stoppedQueue} onUpdateSettings={onUpdateSettings} />);
    const input = screen.getByDisplayValue("3");
    await user.clear(input);
    await user.type(input, "5");
    await user.click(screen.getByText("Save"));
    expect(onUpdateSettings).toHaveBeenCalledWith({ max_workers: 5 });
  });
});
