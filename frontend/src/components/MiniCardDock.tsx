import type { WindowEntry } from "../hooks/useWindowManager";
import { MiniCard } from "./MiniCard";

interface Props {
  windows: WindowEntry[];
  onRestore: (agentId: string) => void;
  onClose: (agentId: string) => void;
}

const STACK_GAP = 8;
const CARD_W = 200;
const CARD_H = 56;
const MARGIN = 8;

/**
 * Computes initial positions for mini-cards stacked along the bottom edge.
 * Each card is offset horizontally by (CARD_W + STACK_GAP).
 */
function stackPositions(count: number): Array<{ x: number; y: number }> {
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  const totalW = count * CARD_W + (count - 1) * STACK_GAP;
  const startX = Math.max(MARGIN, (vw - totalW) / 2);
  const y = vh - CARD_H - MARGIN;

  return Array.from({ length: count }, (_, i) => ({
    x: startX + i * (CARD_W + STACK_GAP),
    y,
  }));
}

export function MiniCardDock({ windows, onRestore, onClose }: Props) {
  if (windows.length === 0) return null;

  const positions = stackPositions(windows.length);

  return (
    <>
      {windows.map((entry, i) => (
        <MiniCard
          key={entry.agentId}
          agentId={entry.agentId}
          name={entry.name}
          role={entry.role}
          unreadCount={entry.unreadCount}
          notificationType={entry.notificationType}
          initialX={positions[i].x}
          initialY={positions[i].y}
          onRestore={() => onRestore(entry.agentId)}
          onClose={() => onClose(entry.agentId)}
        />
      ))}
    </>
  );
}
