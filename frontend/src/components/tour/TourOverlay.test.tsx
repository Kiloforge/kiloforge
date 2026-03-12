import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { TourOverlay } from "./TourOverlay";
import { TOUR_STEPS, TOTAL_STEPS } from "./tourSteps";

// Mock TourProvider context so we can render TourOverlay in isolation
const mockContext = {
  tourState: { status: "active" as const, current_step: 0, completed_at: undefined },
  isActive: true,
  isPending: false,
  currentStep: 0,
  totalSteps: TOTAL_STEPS,
  startTour: vi.fn(),
  dismissTour: vi.fn(),
  restartTour: vi.fn(),
  nextStep: vi.fn(),
  completeTour: vi.fn(),
  demoProjectSlug: "demo",
};

vi.mock("./TourProvider", () => ({
  useTourContext: () => mockContext,
}));

describe("TourOverlay", () => {
  it("does not render a 'Skip step' link on the move-card step", () => {
    const moveCardIndex = TOUR_STEPS.findIndex((s) => s.id === "move-card");
    expect(moveCardIndex).toBeGreaterThan(0);

    mockContext.currentStep = moveCardIndex;

    render(<TourOverlay />);

    expect(screen.queryByText("Skip step")).not.toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
  });

  it("renders 'Next' button for regular spotlight steps", () => {
    // Use a non-wait, non-welcome, non-finish step
    const regularIndex = TOUR_STEPS.findIndex((s) => s.id === "add-project");
    expect(regularIndex).toBeGreaterThan(0);

    mockContext.currentStep = regularIndex;

    render(<TourOverlay />);

    expect(screen.getByText("Next")).toBeInTheDocument();
    expect(screen.queryByText("Skip step")).not.toBeInTheDocument();
  });

  describe("spotlight scroll tracking", () => {
    let rafQueue: FrameRequestCallback[];
    let nextRafId: number;

    beforeEach(() => {
      rafQueue = [];
      nextRafId = 1;
      vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
        rafQueue.push(cb);
        return nextRafId++;
      });
      vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => {});
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
      vi.restoreAllMocks();
    });

    function flushRaf() {
      const cbs = rafQueue.splice(0);
      for (const cb of cbs) cb(performance.now());
    }

    it("repositions spotlight when scroll event fires", () => {
      const addProjectIndex = TOUR_STEPS.findIndex((s) => s.id === "add-project");
      mockContext.currentStep = addProjectIndex;

      // Create a target element that querySelector will find
      const target = document.createElement("div");
      target.setAttribute("data-tour", "add-project-form");
      document.body.appendChild(target);

      let rectTop = 100;
      vi.spyOn(target, "getBoundingClientRect").mockImplementation(
        () =>
          ({
            top: rectTop,
            left: 50,
            width: 200,
            height: 40,
            bottom: rectTop + 40,
            right: 250,
            x: 50,
            y: rectTop,
            toJSON: () => ({}),
          }) as DOMRect,
      );

      const { container } = render(<TourOverlay />);

      const getSpotlight = () => container.querySelector("[class*='spotlight']") as HTMLElement | null;
      let spotlight = getSpotlight();
      // Initial render calls updateRect synchronously — spotlight should be positioned
      expect(spotlight).not.toBeNull();
      expect(spotlight!.style.top).toBe("92px"); // 100 - 8 pad

      // Simulate scroll — element moves to new position
      rectTop = 300;
      act(() => {
        window.dispatchEvent(new Event("scroll"));
        flushRaf();
      });

      spotlight = getSpotlight();
      expect(spotlight).not.toBeNull();
      expect(spotlight!.style.top).toBe("292px"); // 300 - 8 pad

      document.body.removeChild(target);
    });
  });
});
