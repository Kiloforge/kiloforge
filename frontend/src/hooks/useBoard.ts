import { useState, useEffect, useCallback } from "react";
import type { BoardState } from "../types/api";

interface UseBoardResult {
  board: BoardState | null;
  loading: boolean;
  moveCard: (trackId: string, toColumn: string) => Promise<void>;
  refresh: () => void;
}

export function useBoard(project?: string): UseBoardResult {
  const [board, setBoard] = useState<BoardState | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchBoard = useCallback(() => {
    if (!project) {
      setLoading(false);
      return;
    }
    fetch(`/-/api/board/${encodeURIComponent(project)}`)
      .then((r) => r.json())
      .then((data: BoardState) => {
        setBoard(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [project]);

  useEffect(() => {
    fetchBoard();
    const interval = setInterval(fetchBoard, 15000);
    return () => clearInterval(interval);
  }, [fetchBoard]);

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
          `/-/api/board/${encodeURIComponent(project)}/move`,
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

  return { board, loading, moveCard, refresh: fetchBoard };
}
