import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { BoardState, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseBoardResult {
  board: BoardState | null;
  loading: boolean;
  moveCard: (trackId: string, toColumn: string) => Promise<void>;
  refresh: () => void;
  handleBoardUpdate: (raw: unknown) => void;
}

export function useBoard(project?: string): UseBoardResult {
  const queryClient = useQueryClient();
  const key = queryKeys.board(project ?? "");

  const { data: board = null, isLoading, refetch } = useQuery({
    queryKey: key,
    queryFn: () => fetcher<BoardState>(`/api/board/${encodeURIComponent(project!)}`),
    enabled: !!project,
    refetchInterval: 300_000,
  });

  const handleBoardUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as Record<string, unknown>;
      if (data && "columns" in data && "cards" in data) {
        queryClient.setQueryData<BoardState>(key, data as unknown as BoardState);
        return;
      }
      if (data && "track_id" in data && "to_column" in data) {
        queryClient.setQueryData<BoardState>(key, (prev) => {
          if (!prev) return prev;
          const trackId = data.track_id as string;
          const card = prev.cards[trackId];
          if (!card) return prev;
          return {
            ...prev,
            cards: {
              ...prev.cards,
              [trackId]: { ...card, column: data.to_column as string },
            },
          };
        });
      }
    },
    [queryClient, key],
  );

  const moveMutation = useMutation({
    mutationFn: ({ trackId, toColumn }: { trackId: string; toColumn: string }) =>
      fetcher<void>(`/api/board/${encodeURIComponent(project!)}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ track_id: trackId, to_column: toColumn }),
      }),
    onMutate: async ({ trackId, toColumn }) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<BoardState>(key);
      queryClient.setQueryData<BoardState>(key, (prev) => {
        if (!prev) return prev;
        const card = prev.cards[trackId];
        if (!card || card.column === toColumn) return prev;
        return {
          ...prev,
          cards: {
            ...prev.cards,
            [trackId]: { ...card, column: toColumn },
          },
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData<BoardState>(key, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: key });
    },
  });

  const moveCard = async (trackId: string, toColumn: string) => {
    if (!project) return;
    await moveMutation.mutateAsync({ trackId, toColumn });
  };

  return {
    board,
    loading: isLoading,
    moveCard,
    refresh: () => { refetch(); },
    handleBoardUpdate,
  };
}
