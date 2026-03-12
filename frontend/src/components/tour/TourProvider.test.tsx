import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router-dom";
import { TourProvider, useTourContext } from "./TourProvider";

const mockCompleteTour = vi.fn();
const mockTourState = {
  tourState: { status: "active" as const, current_step: 13, completed_at: undefined },
  loading: false,
  startTour: vi.fn(),
  advanceStep: vi.fn(),
  dismissTour: vi.fn(),
  completeTour: mockCompleteTour,
  restartTour: vi.fn(),
  isActive: true,
  isPending: false,
};

vi.mock("../../hooks/useTour", () => ({
  useTour: () => mockTourState,
}));

// Capture the current location inside the router
let capturedPathname = "";
function LocationSpy() {
  const loc = useLocation();
  capturedPathname = loc.pathname;
  return null;
}

// Consumer component that exposes completeTour for testing
function CompleteTourButton() {
  const ctx = useTourContext();
  return (
    <button onClick={ctx.completeTour}>Finish</button>
  );
}

function renderWithProviders(initialPath: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>
        <TourProvider>
          <CompleteTourButton />
          <LocationSpy />
        </TourProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("TourProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedPathname = "";
  });

  it("navigates to '/' when completeTour is called", async () => {
    const user = userEvent.setup();
    renderWithProviders("/projects/example-project");

    expect(capturedPathname).toBe("/");  // finish step route navigates to /

    await user.click(screen.getByText("Finish"));

    expect(mockCompleteTour).toHaveBeenCalled();
    expect(capturedPathname).toBe("/");
  });
});
