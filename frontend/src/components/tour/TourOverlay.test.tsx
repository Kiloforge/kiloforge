import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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
});
