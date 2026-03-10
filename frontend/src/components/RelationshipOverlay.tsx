import { useState, useEffect, useCallback, useRef } from "react";
import type { TrackDependency, TrackConflict } from "../types/api";
import styles from "./RelationshipOverlay.module.css";

interface CardPosition {
  x: number;
  y: number;
  width: number;
  height: number;
}

interface DepLine {
  fromId: string;
  toId: string;
  satisfied: boolean;
}

interface ConflictLine {
  idA: string;
  idB: string;
  risk: "high" | "medium" | "low";
}

interface RelationshipOverlayProps {
  cardRefs: Map<string, HTMLElement>;
  containerRef: React.RefObject<HTMLElement | null>;
  dependencies: Map<string, TrackDependency[]>;
  conflicts: Map<string, TrackConflict[]>;
  visible: boolean;
  onToggle: () => void;
}

/** Compute a straight-line path between the nearest edges of two cards. */
export function computeEdgePath(
  from: CardPosition,
  to: CardPosition,
  lineIndex = 0,
): string {
  // Spread overlapping lines vertically
  const spread = lineIndex * 6;

  const fromCx = from.x + from.width / 2;
  const toCx = to.x + to.width / 2;
  const dx = toCx - fromCx;

  let x1: number, y1: number, x2: number, y2: number;

  if (Math.abs(dx) < 20) {
    // Same column: connect via top/bottom edges
    const fromCy = from.y + from.height / 2;
    const toCy = to.y + to.height / 2;
    if (fromCy < toCy) {
      // from is above to
      x1 = fromCx + spread;
      y1 = from.y + from.height;
      x2 = toCx + spread;
      y2 = to.y;
    } else {
      // from is below to
      x1 = fromCx + spread;
      y1 = from.y;
      x2 = toCx + spread;
      y2 = to.y + to.height;
    }
  } else if (dx > 0) {
    // to is to the right: from right edge → to left edge
    x1 = from.x + from.width;
    y1 = from.y + from.height / 2 + spread;
    x2 = to.x;
    y2 = to.y + to.height / 2 + spread;
  } else {
    // to is to the left: from left edge → to right edge
    x1 = from.x;
    y1 = from.y + from.height / 2 + spread;
    x2 = to.x + to.width;
    y2 = to.y + to.height / 2 + spread;
  }

  return `M ${x1} ${y1} L ${x2} ${y2}`;
}

export function RelationshipOverlay({
  cardRefs,
  containerRef,
  dependencies,
  conflicts,
  visible,
  onToggle,
}: RelationshipOverlayProps) {
  const [positions, setPositions] = useState<Map<string, CardPosition>>(new Map());
  const rafRef = useRef(0);

  const computePositions = useCallback(() => {
    if (!containerRef.current) return;
    const containerRect = containerRef.current.getBoundingClientRect();
    const next = new Map<string, CardPosition>();
    cardRefs.forEach((el, id) => {
      const rect = el.getBoundingClientRect();
      next.set(id, {
        x: rect.left - containerRect.left,
        y: rect.top - containerRect.top,
        width: rect.width,
        height: rect.height,
      });
    });
    setPositions(next);
  }, [cardRefs, containerRef]);

  // Debounced position recomputation via rAF
  const scheduleUpdate = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    rafRef.current = requestAnimationFrame(computePositions);
  }, [computePositions]);

  useEffect(() => {
    if (!visible || !containerRef.current) return;
    computePositions();
    const observer = new ResizeObserver(scheduleUpdate);
    observer.observe(containerRef.current);
    window.addEventListener("resize", scheduleUpdate);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", scheduleUpdate);
      cancelAnimationFrame(rafRef.current);
    };
  }, [visible, containerRef, computePositions, scheduleUpdate]);

  // Recalculate when card refs change (cards moved between columns)
  useEffect(() => {
    if (visible) scheduleUpdate();
  }, [visible, cardRefs.size, scheduleUpdate]);

  // Build line data
  const depLines: DepLine[] = [];
  const conflictLines: ConflictLine[] = [];
  const seenConflicts = new Set<string>();

  if (visible) {
    dependencies.forEach((deps, trackId) => {
      for (const dep of deps) {
        if (positions.has(trackId) && positions.has(dep.id)) {
          depLines.push({
            fromId: dep.id, // prerequisite
            toId: trackId,  // dependent
            satisfied: dep.status === "complete",
          });
        }
      }
    });

    conflicts.forEach((pairs, trackId) => {
      for (const pair of pairs) {
        const key = [trackId, pair.track_id].sort().join("/");
        if (seenConflicts.has(key)) continue;
        seenConflicts.add(key);
        if (positions.has(trackId) && positions.has(pair.track_id)) {
          conflictLines.push({
            idA: trackId,
            idB: pair.track_id,
            risk: pair.risk,
          });
        }
      }
    });
  }

  const hasLines = depLines.length > 0 || conflictLines.length > 0;
  const hasData = dependencies.size > 0 || conflicts.size > 0;

  if (!hasData) return null;

  return (
    <>
      <div className={styles.toggle} onClick={onToggle}>
        <span className={`${styles.toggleIcon} ${!visible ? styles.toggleIconOff : ""}`}>
          &#x2194;
        </span>
        {visible ? "Hide relations" : "Show relations"}
      </div>
      {visible && hasLines && (
        <svg className={styles.overlay}>
          <defs>
            <filter id="glow-green" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur in="SourceGraphic" stdDeviation="4" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
            <filter id="glow-gray" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
            <filter id="glow-red" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur in="SourceGraphic" stdDeviation="5" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
            <filter id="glow-orange" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur in="SourceGraphic" stdDeviation="4" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
            <filter id="glow-yellow" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
          </defs>
          {depLines.map((line, i) => {
            const from = positions.get(line.fromId)!;
            const to = positions.get(line.toId)!;
            return (
              <path
                key={`dep-${line.fromId}-${line.toId}`}
                d={computeEdgePath(from, to, i)}
                className={`${styles.depLine} ${
                  line.satisfied ? styles.depLineSatisfied : styles.depLinePending
                }`}
              />
            );
          })}
          {conflictLines.map((line, i) => {
            const from = positions.get(line.idA)!;
            const to = positions.get(line.idB)!;
            const riskClass =
              line.risk === "high"
                ? styles.conflictHigh
                : line.risk === "medium"
                ? styles.conflictMedium
                : styles.conflictLow;
            return (
              <path
                key={`conflict-${line.idA}-${line.idB}`}
                d={computeEdgePath(from, to, depLines.length + i)}
                className={`${styles.conflictLine} ${riskClass}`}
              />
            );
          })}
        </svg>
      )}
    </>
  );
}
