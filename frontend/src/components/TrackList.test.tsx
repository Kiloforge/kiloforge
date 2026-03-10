import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { TrackList } from "./TrackList";
import type { Track } from "../types/api";

function makeTrack(overrides: Partial<Track> = {}): Track {
  return { id: "t-1", title: "Test Track", status: "pending", ...overrides };
}

function renderList(tracks: Track[], projectSlug = "proj") {
  return render(
    <MemoryRouter>
      <TrackList tracks={tracks} projectSlug={projectSlug} />
    </MemoryRouter>,
  );
}

describe("TrackList", () => {
  it("renders tracks", () => {
    renderList([makeTrack()]);
    expect(screen.getByText("Test Track")).toBeInTheDocument();
  });

  it("shows empty state when no tracks", () => {
    renderList([]);
    expect(screen.getByText("No tracks found")).toBeInTheDocument();
  });

  it("shows ready dot for pending track with no deps", () => {
    const { container } = renderList([makeTrack({ id: "t-ready" })]);
    const dot = container.querySelector("[data-testid='ready-dot']");
    expect(dot).toBeInTheDocument();
  });

  it("shows ready dot for pending track with all deps met", () => {
    const { container } = renderList([
      makeTrack({ id: "t-met", deps_count: 2, deps_met: 2 }),
    ]);
    const dot = container.querySelector("[data-testid='ready-dot']");
    expect(dot).toBeInTheDocument();
  });

  it("does not show ready dot for pending track with unmet deps", () => {
    const { container } = renderList([
      makeTrack({ id: "t-blocked", deps_count: 3, deps_met: 1 }),
    ]);
    const dot = container.querySelector("[data-testid='ready-dot']");
    expect(dot).not.toBeInTheDocument();
  });

  it("does not show ready dot for in-progress track", () => {
    const { container } = renderList([
      makeTrack({ id: "t-ip", status: "in-progress" }),
    ]);
    const dot = container.querySelector("[data-testid='ready-dot']");
    expect(dot).not.toBeInTheDocument();
  });

  it("does not show ready dot for complete track", () => {
    const { container } = renderList([
      makeTrack({ id: "t-done", status: "complete" }),
    ]);
    const dot = container.querySelector("[data-testid='ready-dot']");
    expect(dot).not.toBeInTheDocument();
  });
});
