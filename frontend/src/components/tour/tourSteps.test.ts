import { describe, it, expect } from "vitest";
import { TOUR_STEPS, TOTAL_STEPS } from "./tourSteps";

describe("tourSteps", () => {
  it("has 14 steps total", () => {
    expect(TOUR_STEPS).toHaveLength(14);
    expect(TOTAL_STEPS).toBe(14);
  });

  it("all step IDs are unique", () => {
    const ids = TOUR_STEPS.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all steps have required fields", () => {
    for (const step of TOUR_STEPS) {
      expect(step.id).toBeTruthy();
      expect(step.target).toBeTruthy();
      expect(step.title).toBeTruthy();
      expect(step.content).toBeTruthy();
    }
  });

  it("contains expected step IDs in order", () => {
    const ids = TOUR_STEPS.map((s) => s.id);
    expect(ids).toEqual([
      "welcome",
      "add-project",
      "open-project",
      "setup-notice",
      "swarm-capacity",
      "generate-tracks",
      "board-explanation",
      "track-states",
      "move-card",
      "deps-conflicts",
      "agent-types",
      "notification-center",
      "traces",
      "finish",
    ]);
  });

  it("welcome and finish steps target body", () => {
    expect(TOUR_STEPS[0].target).toBe("body");
    expect(TOUR_STEPS[13].target).toBe("body");
  });

  it("move-card step has wait-for-drag action", () => {
    const moveCard = TOUR_STEPS.find((s) => s.id === "move-card");
    expect(moveCard?.action).toBe("wait-for-drag");
  });

  it("all steps have demoState defined", () => {
    for (const step of TOUR_STEPS) {
      expect(step.demoState).toBeDefined();
    }
  });
});
