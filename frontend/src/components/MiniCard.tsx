import { useRef, useCallback, useState } from "react";
import type { NotificationType } from "../hooks/useWindowManager";
import styles from "./MiniCard.module.css";

interface MiniCardProps {
  agentId: string;
  name?: string;
  role?: string;
  unreadCount: number;
  notificationType: NotificationType;
  initialX: number;
  initialY: number;
  onRestore: () => void;
  onClose: () => void;
}

type Edge = "bottom" | "left" | "right";

const CARD_W = 200;
const CARD_H = 56;
const MARGIN = 8;

function clampToEdge(x: number, y: number): { x: number; y: number; edge: Edge } {
  const vw = window.innerWidth;
  const vh = window.innerHeight;

  // Distance to each edge
  const distLeft = x;
  const distRight = vw - (x + CARD_W);
  const distBottom = vh - (y + CARD_H);

  const min = Math.min(distLeft, distRight, distBottom);

  if (min === distBottom || y + CARD_H > vh - MARGIN) {
    // Snap to bottom
    return {
      x: Math.max(MARGIN, Math.min(x, vw - CARD_W - MARGIN)),
      y: vh - CARD_H - MARGIN,
      edge: "bottom",
    };
  }
  if (min === distLeft) {
    return {
      x: MARGIN,
      y: Math.max(MARGIN, Math.min(y, vh - CARD_H - MARGIN)),
      edge: "left",
    };
  }
  return {
    x: vw - CARD_W - MARGIN,
    y: Math.max(MARGIN, Math.min(y, vh - CARD_H - MARGIN)),
    edge: "right",
  };
}

const roleClasses: Record<string, string> = {
  developer: styles.roleDeveloper,
  reviewer: styles.roleReviewer,
  interactive: styles.roleInteractive,
};

export function MiniCard({
  agentId,
  name,
  role,
  unreadCount,
  notificationType,
  initialX,
  initialY,
  onRestore,
  onClose,
}: MiniCardProps) {
  const [pos, setPos] = useState(() => {
    const clamped = clampToEdge(initialX, initialY);
    return { x: clamped.x, y: clamped.y };
  });
  const [isDragging, setIsDragging] = useState(false);
  const dragRef = useRef({ startX: 0, startY: 0, posX: 0, posY: 0, moved: false });

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    dragRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      posX: pos.x,
      posY: pos.y,
      moved: false,
    };
    setIsDragging(true);
  }, [pos.x, pos.y]);

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    if (!isDragging) return;
    const dx = e.clientX - dragRef.current.startX;
    const dy = e.clientY - dragRef.current.startY;
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
      dragRef.current.moved = true;
    }
    const newX = dragRef.current.posX + dx;
    const newY = dragRef.current.posY + dy;
    setPos({ x: newX, y: newY });
  }, [isDragging]);

  const handlePointerUp = useCallback(() => {
    if (!isDragging) return;
    setIsDragging(false);
    // Snap to nearest edge
    setPos((prev) => {
      const snapped = clampToEdge(prev.x, prev.y);
      return { x: snapped.x, y: snapped.y };
    });
    // If not dragged, treat as click → restore
    if (!dragRef.current.moved) {
      onRestore();
    }
  }, [isDragging, onRestore]);

  const displayName = name || agentId.slice(0, 8);

  let badgeEl: React.ReactNode = null;
  if (notificationType === "waiting") {
    badgeEl = <span className={`${styles.badge} ${styles.badgeWaiting}`}>&#x23F3;</span>;
  } else if (notificationType === "done") {
    badgeEl = <span className={`${styles.badge} ${styles.badgeDone}`}>&#x2713;</span>;
  } else if (unreadCount > 0) {
    badgeEl = (
      <span className={`${styles.badge} ${styles.badgeUnread}`}>
        {unreadCount > 99 ? "99+" : unreadCount}
      </span>
    );
  }

  return (
    <div
      className={`${styles.miniCard} ${isDragging ? styles.miniCardDragging : ""}`}
      style={{ left: pos.x, top: pos.y }}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      data-testid={`mini-card-${agentId}`}
    >
      <div className={`${styles.roleIndicator} ${roleClasses[role ?? ""] ?? ""}`} />
      <div className={styles.content}>
        <span className={styles.name}>{displayName}</span>
        <span className={styles.status}>{role || "agent"}</span>
      </div>
      {badgeEl}
      <button
        className={styles.closeBtn}
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
        title="Close"
      >
        &times;
      </button>
    </div>
  );
}
