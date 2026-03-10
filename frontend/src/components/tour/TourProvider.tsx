import { createContext, useContext, useCallback, useEffect, useRef, type ReactNode } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTour } from "../../hooks/useTour";
import type { TourState } from "../../hooks/useTour";
import { TOUR_STEPS, TOTAL_STEPS } from "./tourSteps";
import { DEMO_STATES, DEMO_PROJECT_SLUG } from "./tourDemoData";
import type { DemoInjection } from "./tourDemoData";

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
  /** The slug of the demo project used during tour (constant). */
  demoProjectSlug: string;
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

/** Collect all unique query keys referenced by any demo state. */
function allDemoQueryKeys(): (readonly unknown[])[] {
  const seen = new Set<string>();
  const keys: (readonly unknown[])[] = [];
  for (const state of Object.values(DEMO_STATES)) {
    for (const inj of state.inject) {
      const k = JSON.stringify(inj.queryKey);
      if (!seen.has(k)) {
        seen.add(k);
        keys.push(inj.queryKey);
      }
    }
  }
  return keys;
}

const DEMO_QUERY_KEYS = allDemoQueryKeys();

/** Apply a set of demo injections to the query cache. */
function injectDemoData(queryClient: ReturnType<typeof useQueryClient>, injections: DemoInjection[]) {
  for (const { queryKey, data } of injections) {
    queryClient.setQueryData(queryKey, data);
  }
}

/** Remove all demo query data and invalidate to restore live state. */
function clearDemoData(queryClient: ReturnType<typeof useQueryClient>) {
  for (const key of DEMO_QUERY_KEYS) {
    queryClient.removeQueries({ queryKey: key as unknown[] });
  }
  // Invalidate so hooks refetch real data
  queryClient.invalidateQueries();
}

/** Set staleTime to Infinity for all demo-controlled query keys. */
function setDemoQueryDefaults(queryClient: ReturnType<typeof useQueryClient>) {
  for (const key of DEMO_QUERY_KEYS) {
    queryClient.setQueryDefaults(key as unknown[], { staleTime: Infinity, refetchOnWindowFocus: false });
  }
}

/** Restore normal query defaults (remove Infinity overrides). */
function clearDemoQueryDefaults(queryClient: ReturnType<typeof useQueryClient>) {
  for (const key of DEMO_QUERY_KEYS) {
    queryClient.setQueryDefaults(key as unknown[], { staleTime: undefined, refetchOnWindowFocus: undefined });
  }
}

export function TourProvider({ children }: TourProviderProps) {
  const tour = useTour();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const prevActiveRef = useRef(false);

  const currentStep = tour.tourState.current_step;
  const isActive = tour.isActive;

  // --- Demo data injection on tour start / step change ---
  useEffect(() => {
    if (!isActive) {
      // Tour was just deactivated — clean up
      if (prevActiveRef.current) {
        clearDemoQueryDefaults(queryClient);
        clearDemoData(queryClient);
      }
      prevActiveRef.current = false;
      return;
    }

    // Tour is active
    if (!prevActiveRef.current) {
      // Just activated — set query isolation
      setDemoQueryDefaults(queryClient);
    }
    prevActiveRef.current = true;

    // Inject demo data for current step
    const stepDef = TOUR_STEPS[currentStep];
    const demoState = stepDef?.demoState;
    if (demoState) {
      injectDemoData(queryClient, demoState.inject);
    }
  }, [isActive, currentStep, queryClient]);

  // --- Route navigation per step ---
  useEffect(() => {
    if (!isActive) return;
    const stepDef = TOUR_STEPS[currentStep];
    const demoState = stepDef?.demoState;
    if (demoState && location.pathname !== demoState.route) {
      navigate(demoState.route);
    }
  }, [isActive, currentStep, navigate, location.pathname]);

  const nextStep = useCallback(() => {
    const next = currentStep + 1;
    if (next >= TOTAL_STEPS) {
      tour.completeTour();
      return;
    }
    tour.advanceStep(next);
  }, [currentStep, tour]);

  const handleDismiss = useCallback(() => {
    tour.dismissTour();
  }, [tour]);

  const handleComplete = useCallback(() => {
    tour.completeTour();
  }, [tour]);

  const handleRestart = useCallback(() => {
    tour.restartTour();
  }, [tour]);

  const handleStart = useCallback(() => {
    tour.startTour();
  }, [tour]);

  return (
    <TourContext.Provider
      value={{
        tourState: tour.tourState,
        isActive,
        isPending: tour.isPending,
        currentStep,
        totalSteps: TOTAL_STEPS,
        startTour: handleStart,
        dismissTour: handleDismiss,
        restartTour: handleRestart,
        nextStep,
        completeTour: handleComplete,
        demoProjectSlug: DEMO_PROJECT_SLUG,
      }}
    >
      {children}
    </TourContext.Provider>
  );
}
