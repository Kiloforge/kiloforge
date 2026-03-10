/**
 * Clamp a forward column move to at most one step, never beyond "approved".
 * Backward moves and same-column moves pass through unchanged.
 * Mirrors backend domain.ClampForwardMove.
 */
export function clampForwardMove(
  fromCol: string,
  toCol: string,
  columns: string[],
): string {
  const from = columns.indexOf(fromCol);
  const to = columns.indexOf(toCol);
  if (from === -1 || to === -1 || to <= from) {
    return toCol; // not forward or invalid — pass through
  }

  const approvedIdx = columns.indexOf("approved");
  if (approvedIdx === -1) {
    return toCol; // no approved column — pass through
  }

  if (from >= approvedIdx) {
    // Already at or beyond approved — no manual forward move allowed.
    return fromCol;
  }

  // Clamp to at most one step ahead, capped at approved.
  const clamped = Math.min(from + 1, approvedIdx);
  return columns[clamped];
}
