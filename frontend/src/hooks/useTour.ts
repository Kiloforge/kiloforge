import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

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
    mutationFn: (body: { action: string; step?: number }) =>
      fetcher<TourState>("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<TourState>(queryKeys.tour, data);
    },
  });

  const startTour = useCallback(() => {
    updateMutation.mutate({ action: "accept" });
  }, [updateMutation]);

  const advanceStep = useCallback(
    (step: number) => {
      updateMutation.mutate({ action: "advance", step });
    },
    [updateMutation],
  );

  const dismissTour = useCallback(() => {
    updateMutation.mutate({ action: "dismiss" });
  }, [updateMutation]);

  const completeTour = useCallback(() => {
    updateMutation.mutate({ action: "complete" });
  }, [updateMutation]);

  const restartTour = useCallback(() => {
    updateMutation.mutate({ action: "accept" });
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
