import { describe, it, expect } from "vitest";
import { clampForwardMove } from "./board";

const COLUMNS = ["backlog", "approved", "in_progress", "done"];

describe("clampForwardMove", () => {
  // Forward moves that should be clamped
  it("clamps backlog→approved (1 step, allowed)", () => {
    expect(clampForwardMove("backlog", "approved", COLUMNS)).toBe("approved");
  });

  it("clamps backlog→in_progress (2 steps, clamp to approved)", () => {
    expect(clampForwardMove("backlog", "in_progress", COLUMNS)).toBe(
      "approved",
    );
  });

  it("clamps backlog→done (3 steps, clamp to approved)", () => {
    expect(clampForwardMove("backlog", "done", COLUMNS)).toBe("approved");
  });

  it("clamps approved→in_progress (beyond approved, stays)", () => {
    expect(clampForwardMove("approved", "in_progress", COLUMNS)).toBe(
      "approved",
    );
  });

  it("clamps approved→done (beyond approved, stays)", () => {
    expect(clampForwardMove("approved", "done", COLUMNS)).toBe("approved");
  });

  it("clamps in_progress→done (beyond approved, stays)", () => {
    expect(clampForwardMove("in_progress", "done", COLUMNS)).toBe(
      "in_progress",
    );
  });

  // Backward moves — pass through unchanged
  it("allows approved→backlog (backward)", () => {
    expect(clampForwardMove("approved", "backlog", COLUMNS)).toBe("backlog");
  });

  it("allows done→backlog (backward)", () => {
    expect(clampForwardMove("done", "backlog", COLUMNS)).toBe("backlog");
  });

  it("allows in_progress→approved (backward)", () => {
    expect(clampForwardMove("in_progress", "approved", COLUMNS)).toBe(
      "approved",
    );
  });

  // Same column — pass through
  it("returns same column when from === to", () => {
    expect(clampForwardMove("backlog", "backlog", COLUMNS)).toBe("backlog");
  });

  // Invalid column — pass through
  it("returns toCol for unknown columns", () => {
    expect(clampForwardMove("unknown", "done", COLUMNS)).toBe("done");
  });
});
