import { createContext, useContext, useCallback, useState, type ReactNode } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useTour, useDemoBoard } from "../../hooks/useTour";
import type { TourState } from "../../hooks/useTour";
import type { BoardState } from "../../types/api";
import { TOUR_STEPS, TOTAL_STEPS } from "./tourSteps";

interface TourContextValue {
  tourState: TourState;
  isActive: boolean;
  isPending: boolean;
  currentStep: number;
  totalSteps: number;
  startTour: () => void;
  dismissTour: () => void;
  restartTour: () => void;
  nextStep: () => void;
  completeTour: () => void;
  demoBoard: BoardState | undefined;
  fetchDemoBoard: () => void;
  /** The slug of the demo project created during tour */
  demoProjectSlug: string | null;
  setDemoProjectSlug: (slug: string | null) => void;
}

const TourContext = createContext<TourContextValue | null>(null);

export function useTourContext() {
  const ctx = useContext(TourContext);
  if (!ctx) throw new Error("useTourContext must be used within TourProvider");
  return ctx;
}

// Optional hook that doesn't throw — for components outside TourProvider
export function useTourContextSafe(): TourContextValue | null {
  return useContext(TourContext);
}

interface TourProviderProps {
  children: ReactNode;
}

export function TourProvider({ children }: TourProviderProps) {
  const tour = useTour();
  const { data: demoBoard, refetch: fetchDemoBoard } = useDemoBoard();
  const navigate = useNavigate();
  const location = useLocation();
  const [demoProjectSlug, setDemoProjectSlug] = useState<string | null>(null);

  const currentStep = tour.tourState.current_step;

  const nextStep = useCallback(() => {
    const next = currentStep + 1;
    if (next >= TOTAL_STEPS) {
      tour.completeTour();
      return;
    }
    const nextDef = TOUR_STEPS[next];
    // Navigate if the step requires a different page
    if (nextDef?.page === "project" && demoProjectSlug && !location.pathname.includes("/projects/")) {
      navigate(`/projects/${demoProjectSlug}`);
    }
    tour.advanceStep(next);
  }, [currentStep, tour, navigate, location, demoProjectSlug]);

  return (
    <TourContext.Provider
      value={{
        tourState: tour.tourState,
        isActive: tour.isActive,
        isPending: tour.isPending,
        currentStep,
        totalSteps: TOTAL_STEPS,
        startTour: tour.startTour,
        dismissTour: tour.dismissTour,
        restartTour: tour.restartTour,
        nextStep,
        completeTour: tour.completeTour,
        demoBoard,
        fetchDemoBoard: () => { fetchDemoBoard(); },
        demoProjectSlug,
        setDemoProjectSlug,
      }}
    >
      {children}
    </TourContext.Provider>
  );
}
