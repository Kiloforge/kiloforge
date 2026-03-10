import { useCallback, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { SwarmCapacity, SSEEventData } from "../types/api";
import { fetcher } from "../api/fetcher";

const CAPACITY_KEY = ["swarm", "capacity"] as const;

interface UseSwarmCapacityResult {
  capacity: SwarmCapacity | null;
  loading: boolean;
  handleCapacityChanged: (raw: unknown) => void;
  waitForSlot: () => Promise<void>;
  cancelWait: () => void;
}

export function useSwarmCapacity(): UseSwarmCapacityResult {
  const queryClient = useQueryClient();
  const [, setWaitResolve] = useState<(() => void) | null>(null);
  const waitResolveRef = useRef<(() => void) | null>(null);

  const { data: capacity = null, isLoading } = useQuery({
    queryKey: CAPACITY_KEY,
    queryFn: () => fetcher<SwarmCapacity>("/api/swarm/capacity"),
  });

  const handleCapacityChanged = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as SwarmCapacity;
      if (data && typeof data.available === "number") {
        queryClient.setQueryData<SwarmCapacity>(CAPACITY_KEY, data);
        if (data.available > 0 && waitResolveRef.current) {
          waitResolveRef.current();
          waitResolveRef.current = null;
          setWaitResolve(null);
        }
      } else {
        queryClient.invalidateQueries({ queryKey: CAPACITY_KEY });
      }
    },
    [queryClient],
  );

  const waitForSlot = useCallback((): Promise<void> => {
    return new Promise<void>((resolve) => {
      waitResolveRef.current = resolve;
      setWaitResolve(() => resolve);
    });
  }, []);

  const cancelWait = useCallback(() => {
    if (waitResolveRef.current) {
      waitResolveRef.current = null;
      setWaitResolve(null);
    }
  }, []);

  return {
    capacity,
    loading: isLoading,
    handleCapacityChanged,
    waitForSlot,
    cancelWait,
  };
}
