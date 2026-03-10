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

/** Compute a cubic bezier path between two card centers. */
export function computeBezierPath(from: CardPosition, to: CardPosition): string {
  const x1 = from.x + from.width / 2;
  const y1 = from.y + from.height / 2;
  const x2 = to.x + to.width / 2;
  const y2 = to.y + to.height / 2;
  const dx = x2 - x1;
  const cpOffset = Math.min(Math.abs(dx) * 0.4, 120);
  return `M ${x1} ${y1} C ${x1 + cpOffset} ${y1}, ${x2 - cpOffset} ${y2}, ${x2} ${y2}`;
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
          {depLines.map((line) => {
            const from = positions.get(line.fromId)!;
            const to = positions.get(line.toId)!;
            return (
              <path
                key={`dep-${line.fromId}-${line.toId}`}
                d={computeBezierPath(from, to)}
                className={`${styles.depLine} ${
                  line.satisfied ? styles.depLineSatisfied : styles.depLinePending
                }`}
              />
            );
          })}
          {conflictLines.map((line) => {
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
                d={computeBezierPath(from, to)}
                className={`${styles.conflictLine} ${riskClass}`}
              />
            );
          })}
        </svg>
      )}
    </>
  );
}
