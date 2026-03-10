import { describe, it, expect } from "vitest";
import { isTrackReady } from "./track";
import type { Track } from "../types/api";

function makeTrack(overrides: Partial<Track> = {}): Track {
  return { id: "t-1", title: "Test", status: "pending", ...overrides };
}

describe("isTrackReady", () => {
  it("returns true for pending track with no deps", () => {
    expect(isTrackReady(makeTrack())).toBe(true);
  });

  it("returns true for pending track with deps_count=0", () => {
    expect(isTrackReady(makeTrack({ deps_count: 0 }))).toBe(true);
  });

  it("returns true for pending track with all deps met", () => {
    expect(isTrackReady(makeTrack({ deps_count: 3, deps_met: 3 }))).toBe(true);
  });

  it("returns false for pending track with unmet deps", () => {
    expect(isTrackReady(makeTrack({ deps_count: 3, deps_met: 1 }))).toBe(false);
  });

  it("returns false for in-progress track even with no deps", () => {
    expect(isTrackReady(makeTrack({ status: "in-progress" }))).toBe(false);
  });

  it("returns false for complete track", () => {
    expect(isTrackReady(makeTrack({ status: "complete" }))).toBe(false);
  });
});
