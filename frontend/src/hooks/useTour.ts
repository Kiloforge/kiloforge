import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import type { BoardState } from "../types/api";

export interface TourState {
  status: "pending" | "active" | "dismissed" | "completed";
  current_step: number;
  started_at?: string;
  dismissed_at?: string;
  completed_at?: string;
}

const DEFAULT_STATE: TourState = { status: "pending", current_step: 0 };

export function useTour() {
  const queryClient = useQueryClient();

  const { data: tourState = DEFAULT_STATE, isLoading } = useQuery({
    queryKey: queryKeys.tour,
    queryFn: () => fetcher<TourState>("/api/tour").catch(() => DEFAULT_STATE),
  });

  const updateMutation = useMutation({
    mutationFn: (state: Partial<TourState>) =>
      fetcher<TourState>("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(state),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<TourState>(queryKeys.tour, data);
    },
  });

  const startTour = useCallback(() => {
    updateMutation.mutate({ status: "active", current_step: 0 });
  }, [updateMutation]);

  const advanceStep = useCallback(
    (step: number) => {
      updateMutation.mutate({ status: "active", current_step: step });
    },
    [updateMutation],
  );

  const dismissTour = useCallback(() => {
    updateMutation.mutate({ status: "dismissed" });
  }, [updateMutation]);

  const completeTour = useCallback(() => {
    updateMutation.mutate({ status: "completed" });
  }, [updateMutation]);

  const restartTour = useCallback(() => {
    updateMutation.mutate({ status: "pending", current_step: 0 });
  }, [updateMutation]);

  return {
    tourState,
    loading: isLoading,
    startTour,
    advanceStep,
    dismissTour,
    completeTour,
    restartTour,
    isActive: tourState.status === "active",
    isPending: tourState.status === "pending",
  };
}

export function useDemoBoard() {
  return useQuery({
    queryKey: queryKeys.tourDemoBoard,
    queryFn: () => fetcher<BoardState>("/api/tour/demo-board"),
    enabled: false,
  });
}
