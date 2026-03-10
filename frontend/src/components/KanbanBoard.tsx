import { useState, useRef, useCallback, useEffect } from "react";
import { Link } from "react-router-dom";
import type { BoardState, BoardCard, TrackDependency, TrackConflict } from "../types/api";
import { useTourContextSafe } from "./tour/TourProvider";
import { TOUR_STEPS } from "./tour/tourSteps";
import { RelationshipOverlay } from "./RelationshipOverlay";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { clampForwardMove } from "../utils/board";
import styles from "./KanbanBoard.module.css";

/** Columns visible on the board — "done" is filtered out (cards go to track list). */
const BOARD_COLUMNS = ["backlog", "approved", "in_progress"] as const;

const COLUMN_LABELS: Record<string, string> = {
  backlog: "Backlog",
  approved: "Approved",
  in_progress: "In Progress",
};

const COLUMN_COLORS: Record<string, string> = {
  backlog: "var(--text-dim)",
  approved: "var(--accent)",
  in_progress: "var(--yellow)",
};

/** Flex weight per column — backlog/approved share equal space, in_progress is narrower. */
const COLUMN_FLEX: Record<string, number> = {
  backlog: 3,
  approved: 3,
  in_progress: 1.5,
};

interface KanbanBoardProps {
  board: BoardState;
  projectSlug?: string;
  onMoveCard: (trackId: string, toColumn: string) => void;
  onDeleteTrack?: (trackId: string) => void;
  dependencies?: Map<string, TrackDependency[]>;
  conflicts?: Map<string, TrackConflict[]>;
}

export function KanbanBoard({ board, projectSlug, onMoveCard, onDeleteTrack, dependencies, conflicts }: KanbanBoardProps) {
  const tour = useTourContextSafe();
  const [dragTrackId, setDragTrackId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const [confirmReject, setConfirmReject] = useState<string | null>(null);
  const [showRelations, setShowRelations] = useState(true);
  const [activeColumn, setActiveColumn] = useState<string>(BOARD_COLUMNS[0]);
  const [clampedCardId, setClampedCardId] = useState<string | null>(null);
  const [enteringCards, setEnteringCards] = useState<Set<string>>(new Set());
  const [completingCards, setCompletingCards] = useState<Set<string>>(new Set());
  const prevCardColumns = useRef<Map<string, string>>(new Map());
  const knownCardIds = useRef<Set<string>>(new Set());
  const isMobile = useMediaQuery("(max-width: 767px)");

  // Clear clamped animation after it plays
  useEffect(() => {
    if (!clampedCardId) return;
    const timer = setTimeout(() => setClampedCardId(null), 600);
    return () => clearTimeout(timer);
  }, [clampedCardId]);

  // Detect new cards appearing on the board and trigger entry animation
  useEffect(() => {
    const currentIds = new Set(Object.keys(board.cards));
    const newIds = new Set<string>();
    for (const id of currentIds) {
      if (!knownCardIds.current.has(id)) {
        newIds.add(id);
      }
    }
    knownCardIds.current = currentIds;
    if (newIds.size > 0) {
      setEnteringCards(newIds);
      const timer = setTimeout(() => setEnteringCards(new Set()), 450);
      return () => clearTimeout(timer);
    }
  }, [board.cards]);

  // Detect cards transitioning to "done" and trigger completion animation
  useEffect(() => {
    const newCompleting = new Set<string>();
    for (const [trackId, card] of Object.entries(board.cards)) {
      const prevCol = prevCardColumns.current.get(trackId);
      if (prevCol && prevCol !== "done" && card.column === "done") {
        newCompleting.add(trackId);
      }
    }
    // Update previous column snapshot
    const nextMap = new Map<string, string>();
    for (const [trackId, card] of Object.entries(board.cards)) {
      nextMap.set(trackId, card.column);
    }
    prevCardColumns.current = nextMap;

    if (newCompleting.size > 0) {
      setCompletingCards(newCompleting);
      const timer = setTimeout(() => setCompletingCards(new Set()), 900);
      return () => clearTimeout(timer);
    }
  }, [board.cards]);

  const boardRef = useRef<HTMLDivElement>(null);
  const cardRefsMap = useRef(new Map<string, HTMLElement>());

  const registerCardRef = useCallback((trackId: string, el: HTMLElement | null) => {
    if (el) {
      cardRefsMap.current.set(trackId, el);
    } else {
      cardRefsMap.current.delete(trackId);
    }
  }, []);

  const cardsByColumn = (col: string): BoardCard[] => {
    return Object.values(board.cards)
      .filter((c) => {
        if (c.column === col) return true;
        // Show completing cards in in_progress column during animation
        if (col === "in_progress" && completingCards.has(c.track_id)) return true;
        return false;
      })
      .sort((a, b) => a.position - b.position);
  };

  const handleDragStart = (trackId: string) => {
    setDragTrackId(trackId);
  };

  const handleDragEnd = () => {
    setDragTrackId(null);
    setDropTarget(null);
  };

  const handleDragOver = (e: React.DragEvent, col: string) => {
    e.preventDefault();
    setDropTarget(col);
  };

  const handleDragLeave = () => {
    setDropTarget(null);
  };

  const handleDrop = (e: React.DragEvent, col: string) => {
    e.preventDefault();
    if (dragTrackId) {
      const fromCol = board.cards[dragTrackId]?.column;
      if (fromCol) {
        const effective = clampForwardMove(fromCol, col, board.columns);
        if (effective !== fromCol) {
          onMoveCard(dragTrackId, effective);
          if (effective !== col) {
            setClampedCardId(dragTrackId);
          }
        }
        // Tour: detect drag to approved during move-card step
        if (tour?.isActive && effective === "approved") {
          const step = TOUR_STEPS[tour.currentStep];
          if (step?.id === "move-card") {
            tour.completeTour();
          }
        }
      }
    }
    setDragTrackId(null);
    setDropTarget(null);
  };

  const emptyMap = emptyMapRef.current;

  const handleMobileMove = useCallback((trackId: string, toColumn: string) => {
    const fromCol = board.cards[trackId]?.column;
    if (!fromCol) return;
    const effective = clampForwardMove(fromCol, toColumn, board.columns);
    if (effective !== fromCol) {
      onMoveCard(trackId, effective);
      if (effective !== toColumn) {
        setClampedCardId(trackId);
      }
    }
  }, [board, onMoveCard]);

  return (
    <div className={styles.boardWrapper} style={{ position: "relative" }}>
      {!isMobile && (
        <RelationshipOverlay
          cardRefs={cardRefsMap.current}
          containerRef={boardRef}
          dependencies={dependencies ?? emptyMap}
          conflicts={conflicts ?? emptyMap}
          visible={showRelations}
          onToggle={() => setShowRelations((v) => !v)}
        />
      )}
      {isMobile && (
        <div className={styles.tabBar}>
          {BOARD_COLUMNS.map((col) => (
            <button
              key={col}
              className={`${styles.tab} ${activeColumn === col ? styles.tabActive : ""}`}
              onClick={() => setActiveColumn(col)}
            >
              <span className={styles.tabDot} style={{ background: COLUMN_COLORS[col] }} />
              {COLUMN_LABELS[col] || col}
              <span className={styles.tabBadge}>{cardsByColumn(col).length}</span>
            </button>
          ))}
        </div>
      )}
      <div className={styles.board} data-tour="kanban-board" ref={boardRef}>
        {BOARD_COLUMNS.map((col) => {
          const cards = cardsByColumn(col);
          const isOver = dropTarget === col;
          const isActive = !isMobile || activeColumn === col;
          return (
            <div
              key={col}
              className={`${styles.column} ${isOver ? styles.columnOver : ""} ${isMobile && isActive ? styles.columnActive : ""}`}
              style={isMobile && !isActive ? { display: "none" } : { flex: COLUMN_FLEX[col] ?? 1 }}
              onDragOver={isMobile ? undefined : (e) => handleDragOver(e, col)}
              onDragLeave={isMobile ? undefined : handleDragLeave}
              onDrop={isMobile ? undefined : (e) => handleDrop(e, col)}
            >
              <div className={styles.columnHeader}>
                <span
                  className={styles.columnDot}
                  style={{ background: COLUMN_COLORS[col] }}
                />
                <span className={styles.columnTitle}>
                  {COLUMN_LABELS[col] || col}
                </span>
                <span className={styles.columnCount}>{cards.length}</span>
              </div>
              <div className={styles.cards}>
                {cards.map((card, idx) => (
                  <CardItem
                    key={card.track_id}
                    card={card}
                    projectSlug={projectSlug}
                    isDragging={dragTrackId === card.track_id}
                    isClamped={clampedCardId === card.track_id}
                    isEntering={enteringCards.has(card.track_id)}
                    isCompleting={completingCards.has(card.track_id)}
                    isBacklog={col === "backlog"}
                    isReady={col === "backlog" && (
                      !dependencies?.get(card.track_id)?.length ||
                      dependencies?.get(card.track_id)?.every(d => d.status === "completed") === true
                    )}
                    dataTour={col === "backlog" && idx === 0 ? "board-card-first" : undefined}
                    confirmingReject={confirmReject === card.track_id}
                    onDragStart={isMobile ? undefined : () => handleDragStart(card.track_id)}
                    onDragEnd={isMobile ? undefined : handleDragEnd}
                    onApprove={() => onMoveCard(card.track_id, "approved")}
                    onReject={() => setConfirmReject(card.track_id)}
                    onConfirmReject={() => {
                      onDeleteTrack?.(card.track_id);
                      setConfirmReject(null);
                    }}
                    onCancelReject={() => setConfirmReject(null)}
                    cardRef={(el) => registerCardRef(card.track_id, el)}
                    isMobile={isMobile}
                    currentColumn={col}
                    columns={[...BOARD_COLUMNS]}
                    onMobileMove={handleMobileMove}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Stable empty map to avoid re-renders
const emptyMapRef = { current: new Map<string, never[]>() };

interface CardItemProps {
  card: BoardCard;
  projectSlug?: string;
  isDragging: boolean;
  isClamped?: boolean;
  isEntering?: boolean;
  isCompleting?: boolean;
  isBacklog: boolean;
  isReady?: boolean;
  confirmingReject: boolean;
  dataTour?: string;
  cardRef?: (el: HTMLDivElement | null) => void;
  onDragStart?: () => void;
  onDragEnd?: () => void;
  onApprove: () => void;
  onReject: () => void;
  onConfirmReject: () => void;
  onCancelReject: () => void;
  isMobile?: boolean;
  currentColumn?: string;
  columns?: string[];
  onMobileMove?: (trackId: string, toColumn: string) => void;
}

function CardItem({ card, projectSlug, isDragging, isClamped, isEntering, isCompleting, isBacklog, isReady, confirmingReject, dataTour, cardRef, onDragStart, onDragEnd, onApprove, onReject, onConfirmReject, onCancelReject, isMobile, currentColumn, columns, onMobileMove }: CardItemProps) {
  return (
    <div
      className={`${styles.card} ${isDragging ? styles.cardDragging : ""} ${isClamped ? styles.cardClamped : ""} ${isEntering ? styles.cardEntering : ""} ${isCompleting ? styles.cardCompleting : ""} ${isReady ? styles.cardReady : isBacklog ? styles.cardBacklog : ""}`}
      draggable={!isMobile}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      data-tour={dataTour}
      ref={cardRef}
      data-track-id={card.track_id}
    >
      <div className={styles.cardHeader}>
        {card.type && <span className={styles.cardType}>{card.type}</span>}
        {isBacklog && !isReady && <span className={styles.reviewBadge}>Pending Review</span>}
        {isReady && <span className={styles.readyBadge}>Ready</span>}
        {card.pr_number && card.pr_number > 0 && (
          <span className={styles.cardPR}>PR #{card.pr_number}</span>
        )}
      </div>
      <div className={styles.cardTitle}>
        {projectSlug ? (
          <Link
            to={`/projects/${projectSlug}/tracks/${card.track_id}`}
            className={styles.cardTitleLink}
            onClick={(e) => e.stopPropagation()}
          >
            {card.title}
          </Link>
        ) : card.title}
      </div>
      <div className={styles.cardMeta}>
        <span className={styles.cardId}>{card.track_id}</span>
        {card.agent_status && (
          <span className={styles.cardAgent}>{card.agent_status}</span>
        )}
        {card.trace_id && (
          <Link
            to={`/traces/${card.trace_id}`}
            className={styles.cardTrace}
            onClick={(e) => e.stopPropagation()}
            data-tour="trace-link"
          >
            Trace
          </Link>
        )}
      </div>
      {isBacklog && (
        <div className={styles.cardActions}>
          {confirmingReject ? (
            <div className={styles.confirmRow}>
              <span className={styles.confirmText}>Delete track?</span>
              <button className={styles.confirmYes} onClick={(e) => { e.stopPropagation(); onConfirmReject(); }}>Yes</button>
              <button className={styles.confirmNo} onClick={(e) => { e.stopPropagation(); onCancelReject(); }}>No</button>
            </div>
          ) : (
            <>
              <button className={styles.approveBtn} onClick={(e) => { e.stopPropagation(); onApprove(); }} title="Approve">&#x2713;</button>
              <button className={styles.rejectBtn} onClick={(e) => { e.stopPropagation(); onReject(); }} title="Reject">&#x2717;</button>
            </>
          )}
        </div>
      )}
      {isMobile && columns && currentColumn && onMobileMove && (
        <select
          className={styles.moveButton}
          value=""
          onChange={(e) => {
            if (e.target.value) onMobileMove(card.track_id, e.target.value);
          }}
        >
          <option value="">Move to...</option>
          {columns.filter((c) => {
            if (c === currentColumn) return false;
            // Exclude forward columns that would clamp to a no-op
            return clampForwardMove(currentColumn, c, columns) !== currentColumn;
          }).map((c) => (
            <option key={c} value={c}>{COLUMN_LABELS[c] || c}</option>
          ))}
        </select>
      )}
    </div>
  );
}
