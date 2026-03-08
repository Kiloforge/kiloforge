import { useState } from "react";
import { Link } from "react-router-dom";
import type { BoardState, BoardCard } from "../types/api";
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
  onMoveCard: (trackId: string, toColumn: string) => void;
  onDeleteTrack?: (trackId: string) => void;
}

export function KanbanBoard({ board, onMoveCard, onDeleteTrack }: KanbanBoardProps) {
  const [dragTrackId, setDragTrackId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const [confirmReject, setConfirmReject] = useState<string | null>(null);

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
    }
    setDragTrackId(null);
    setDropTarget(null);
  };

  return (
    <div className={styles.board}>
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
              {cards.map((card) => (
                <CardItem
                  key={card.track_id}
                  card={card}
                  isDragging={dragTrackId === card.track_id}
                  isBacklog={col === "backlog"}
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
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

interface CardItemProps {
  card: BoardCard;
  isDragging: boolean;
  isBacklog: boolean;
  confirmingReject: boolean;
  onDragStart: () => void;
  onDragEnd: () => void;
  onApprove: () => void;
  onReject: () => void;
  onConfirmReject: () => void;
  onCancelReject: () => void;
}

function CardItem({ card, isDragging, isBacklog, confirmingReject, onDragStart, onDragEnd, onApprove, onReject, onConfirmReject, onCancelReject }: CardItemProps) {
  return (
    <div
      className={`${styles.card} ${isDragging ? styles.cardDragging : ""} ${isBacklog ? styles.cardBacklog : ""}`}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
    >
      <div className={styles.cardHeader}>
        {card.type && <span className={styles.cardType}>{card.type}</span>}
        {isBacklog && <span className={styles.reviewBadge}>Pending Review</span>}
        {card.pr_number && card.pr_number > 0 && (
          <span className={styles.cardPR}>PR #{card.pr_number}</span>
        )}
      </div>
      <div className={styles.cardTitle}>{card.title}</div>
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
