import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SwarmPanel } from "./SwarmPanel";
import type { SwarmStatus } from "../types/api";

const stoppedSwarm: SwarmStatus = {
  running: false,
  max_workers: 3,
  active_workers: 0,
  items: [],
};

const runningSwarm: SwarmStatus = {
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

describe("SwarmPanel", () => {
  const defaultProps = {
    swarm: null as SwarmStatus | null,
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

  it("shows loading spinner", () => {
    render(<SwarmPanel {...defaultProps} loading={true} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("shows empty state when swarm is null", () => {
    render(<SwarmPanel {...defaultProps} />);
    expect(screen.getByText(/Swarm not configured/)).toBeInTheDocument();
  });

  it("renders Start button when swarm is stopped", () => {
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} />);
    expect(screen.getByText("Start")).toBeInTheDocument();
  });

  it("renders Stop button when swarm is running", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("Stop")).toBeInTheDocument();
  });

  it("calls onStart when Start is clicked", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} onStart={onStart} />);
    await user.click(screen.getByText("Start"));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("calls onStop when Stop is clicked", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} onStop={onStop} />);
    await user.click(screen.getByText("Stop"));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("disables Start button while starting", () => {
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} starting={true} />);
    expect(screen.getByText("Starting...")).toBeDisabled();
  });

  it("disables Stop button while stopping", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} stopping={true} />);
    expect(screen.getByText("Stopping...")).toBeDisabled();
  });

  it("shows agent count and stats", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("2 / 3")).toBeInTheDocument(); // active / max
    expect(screen.getByText("2 queued")).toBeInTheDocument();
  });

  it("shows active agents list", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("developer-1")).toBeInTheDocument();
    expect(screen.getByText("developer-2")).toBeInTheDocument();
    expect(screen.getByText("track-abc")).toBeInTheDocument();
    expect(screen.getByText("track-def")).toBeInTheDocument();
  });

  it("shows queued tracks list", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("track-ghi")).toBeInTheDocument();
    expect(screen.getByText("track-jkl")).toBeInTheDocument();
  });

  it("shows max swarm size input with current value", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    const input = screen.getByDisplayValue("3");
    expect(input).toBeInTheDocument();
  });

  it("uses 'Max Swarm Size' label", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("Max Swarm Size:")).toBeInTheDocument();
  });

  it("uses 'Agents' label instead of 'Workers'", () => {
    render(<SwarmPanel {...defaultProps} swarm={runningSwarm} />);
    expect(screen.getByText("Agents:")).toBeInTheDocument();
  });

  it("calls onUpdateSettings when max swarm size is changed", async () => {
    const user = userEvent.setup();
    const onUpdateSettings = vi.fn();
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} onUpdateSettings={onUpdateSettings} />);
    const input = screen.getByDisplayValue("3");
    await user.clear(input);
    await user.type(input, "5");
    await user.click(screen.getByText("Save"));
    expect(onUpdateSettings).toHaveBeenCalledWith({ max_workers: 5 });
  });

  it("applies error styling to size input when value is out of range", async () => {
    const user = userEvent.setup();
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} />);
    const input = screen.getByDisplayValue("3");
    await user.clear(input);
    await user.type(input, "0");
    expect(input.className).toMatch(/sizeInputError/);
  });

  it("does not apply error styling for valid size value", async () => {
    const user = userEvent.setup();
    render(<SwarmPanel {...defaultProps} swarm={stoppedSwarm} />);
    const input = screen.getByDisplayValue("3");
    await user.clear(input);
    await user.type(input, "5");
    expect(input.className).not.toMatch(/sizeInputError/);
  });
});
