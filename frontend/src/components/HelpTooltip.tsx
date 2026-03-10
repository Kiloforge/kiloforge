import { useState } from "react";
import styles from "./HelpTooltip.module.css";

interface HelpTooltipProps {
  term: string;
  definition: string;
}

export function HelpTooltip({ term, definition }: HelpTooltipProps) {
  const [visible, setVisible] = useState(false);
  const [pinned, setPinned] = useState(false);

  const show = visible || pinned;

  return (
    <span className={styles.wrapper}>
      <button
        className={styles.trigger}
        aria-label={`What is ${term}?`}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
        onClick={() => setPinned((v) => !v)}
      >
        ?
      </button>
      {show && (
        <div className={styles.card} role="tooltip">
          <span className={styles.term}>{term}</span>
          <span className={styles.definition}>{definition}</span>
        </div>
      )}
    </span>
  );
}
