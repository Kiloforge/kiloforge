import { useState, useRef, useCallback } from "react";
import { Link } from "react-router-dom";
import type { BoardState, BoardCard, TrackDependency, TrackConflict } from "../types/api";
import { useTourContextSafe } from "./tour/TourProvider";
import { TOUR_STEPS } from "./tour/tourSteps";
import { RelationshipOverlay } from "./RelationshipOverlay";
import styles from "./KanbanBoard.module.css";

const COLUMN_LABELS: Record<string, string> = {
  backlog: "Backlog",
  approved: "Approved",
  in_progress: "In Progress",
  in_review: "In Review",
  done: "Done",
};

const COLUMN_COLORS: Record<string, string> = {
  backlog: "var(--text-dim)",
  approved: "var(--accent)",
  in_progress: "var(--yellow)",
  in_review: "var(--orange)",
  done: "var(--green)",
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
      .filter((c) => c.column === col)
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
    if (dragTrackId && board.cards[dragTrackId]?.column !== col) {
      onMoveCard(dragTrackId, col);
      // Tour: detect drag to approved during move-card step
      if (tour?.isActive && col === "approved") {
        const step = TOUR_STEPS[tour.currentStep];
        if (step?.id === "move-card") {
          tour.completeTour();
        }
      }
    }
    setDragTrackId(null);
    setDropTarget(null);
  };

  const emptyMap = emptyMapRef.current;

  return (
    <div className={styles.boardWrapper} style={{ position: "relative" }}>
      <RelationshipOverlay
        cardRefs={cardRefsMap.current}
        containerRef={boardRef}
        dependencies={dependencies ?? emptyMap}
        conflicts={conflicts ?? emptyMap}
        visible={showRelations}
        onToggle={() => setShowRelations((v) => !v)}
      />
      <div className={styles.board} data-tour="kanban-board" ref={boardRef}>
        {board.columns.map((col) => {
          const cards = cardsByColumn(col);
          const isOver = dropTarget === col;
          return (
            <div
              key={col}
              className={`${styles.column} ${isOver ? styles.columnOver : ""}`}
              onDragOver={(e) => handleDragOver(e, col)}
              onDragLeave={handleDragLeave}
              onDrop={(e) => handleDrop(e, col)}
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
                    isBacklog={col === "backlog"}
                    dataTour={col === "backlog" && idx === 0 ? "board-card-first" : undefined}
                    confirmingReject={confirmReject === card.track_id}
                    onDragStart={() => handleDragStart(card.track_id)}
                    onDragEnd={handleDragEnd}
                    onApprove={() => onMoveCard(card.track_id, "approved")}
                    onReject={() => setConfirmReject(card.track_id)}
                    onConfirmReject={() => {
                      onDeleteTrack?.(card.track_id);
                      setConfirmReject(null);
                    }}
                    onCancelReject={() => setConfirmReject(null)}
                    cardRef={(el) => registerCardRef(card.track_id, el)}
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
  isBacklog: boolean;
  confirmingReject: boolean;
  dataTour?: string;
  cardRef?: (el: HTMLDivElement | null) => void;
  onDragStart: () => void;
  onDragEnd: () => void;
  onApprove: () => void;
  onReject: () => void;
  onConfirmReject: () => void;
  onCancelReject: () => void;
}

function CardItem({ card, projectSlug, isDragging, isBacklog, confirmingReject, dataTour, cardRef, onDragStart, onDragEnd, onApprove, onReject, onConfirmReject, onCancelReject }: CardItemProps) {
  return (
    <div
      className={`${styles.card} ${isDragging ? styles.cardDragging : ""} ${isBacklog ? styles.cardBacklog : ""}`}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      data-tour={dataTour}
      ref={cardRef}
      data-track-id={card.track_id}
    >
      <div className={styles.cardHeader}>
        {card.type && <span className={styles.cardType}>{card.type}</span>}
        {isBacklog && <span className={styles.reviewBadge}>Pending Review</span>}
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
    </div>
  );
}
