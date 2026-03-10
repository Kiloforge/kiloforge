import { useState, useRef, useCallback, useLayoutEffect } from "react";
import { createPortal } from "react-dom";
import styles from "./HelpTooltip.module.css";

interface HelpTooltipProps {
  term: string;
  definition: string;
}

export function HelpTooltip({ term, definition }: HelpTooltipProps) {
  const [visible, setVisible] = useState(false);
  const [pinned, setPinned] = useState(false);
  const [pos, setPos] = useState<{ top: number; left: number }>({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLButtonElement>(null);

  const show = visible || pinned;

  const updatePos = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    setPos({
      top: rect.bottom + 8,
      left: rect.left + rect.width / 2,
    });
  }, []);

  useLayoutEffect(() => {
    if (show) updatePos();
  }, [show, updatePos]);

  return (
    <span className={styles.wrapper}>
      <button
        ref={triggerRef}
        className={styles.trigger}
        aria-label={`What is ${term}?`}
        onMouseEnter={() => { setVisible(true); }}
        onMouseLeave={() => setVisible(false)}
        onClick={() => setPinned((v) => !v)}
      >
        ?
      </button>
      {show && createPortal(
        <div
          className={styles.card}
          role="tooltip"
          style={{ top: pos.top, left: pos.left }}
        >
          <span className={styles.term}>{term}</span>
          <span className={styles.definition}>{definition}</span>
        </div>,
        document.body,
      )}
    </span>
  );
}
