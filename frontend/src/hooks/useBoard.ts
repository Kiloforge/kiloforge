import { useState, useEffect, useCallback } from "react";
import type { BoardState, SSEEventData } from "../types/api";

interface UseBoardResult {
  board: BoardState | null;
  loading: boolean;
  moveCard: (trackId: string, toColumn: string) => Promise<void>;
  refresh: () => void;
  handleBoardUpdate: (raw: unknown) => void;
}

export function useBoard(project?: string): UseBoardResult {
  const [board, setBoard] = useState<BoardState | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchBoard = useCallback(() => {
    if (!project) {
      setLoading(false);
      return;
    }
    fetch(`/api/board/${encodeURIComponent(project)}`)
      .then((r) => r.json())
      .then((data: BoardState) => {
        setBoard(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [project]);

  useEffect(() => {
    fetchBoard();
    // Long-interval background sync as drift protection.
    const interval = setInterval(fetchBoard, 300000);
    return () => clearInterval(interval);
  }, [fetchBoard]);

  const handleBoardUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as Record<string, unknown>;
    // If the event contains full board state (columns + cards), replace entirely.
    if (data && "columns" in data && "cards" in data) {
      setBoard(data as unknown as BoardState);
      return;
    }
    // If the event is a card move (track_id, from_column, to_column), update in place.
    if (data && "track_id" in data && "to_column" in data) {
      setBoard((prev) => {
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
  }, []);

  const moveCard = useCallback(
    async (trackId: string, toColumn: string) => {
      if (!project) return;

      // Optimistic update.
      setBoard((prev) => {
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

      try {
        const resp = await fetch(
          `/api/board/${encodeURIComponent(project)}/move`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ track_id: trackId, to_column: toColumn }),
          },
        );
        if (!resp.ok) {
          fetchBoard(); // Revert on error.
        }
      } catch {
        fetchBoard(); // Revert on error.
      }
    },
    [project, fetchBoard],
  );

  return { board, loading, moveCard, refresh: fetchBoard, handleBoardUpdate };
}
