import { useState, useRef, useEffect } from "react";
import { useTourContextSafe } from "./tour/TourProvider";
import { useConfig } from "../hooks/useConfig";
import { useUIScale } from "../hooks/useUIScale";
import styles from "./SettingsMenu.module.css";

export function SettingsMenu() {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const tourCtx = useTourContextSafe();
  const { config, updateConfig } = useConfig();
  const { scale, setScale } = useUIScale();

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  const handleTakeTour = () => {
    tourCtx?.restartTour();
    setOpen(false);
  };

  const handleToggleAnalytics = () => {
    const next = !(config?.analytics_enabled ?? true);
    updateConfig({ analytics_enabled: next });
  };

  const analyticsEnabled = config?.analytics_enabled ?? true;

  return (
    <div className={styles.wrapper} ref={ref}>
      <button
        className={styles.trigger}
        onClick={() => setOpen((v) => !v)}
        title="Settings"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="8" cy="8" r="2.5" />
          <path d="M13.5 8a5.5 5.5 0 0 0-.08-.88l1.38-1.08a.33.33 0 0 0 .08-.42l-1.3-2.26a.33.33 0 0 0-.4-.15l-1.63.66a5.2 5.2 0 0 0-1.52-.88L9.7 1.3A.33.33 0 0 0 9.37 1H6.77a.33.33 0 0 0-.33.28l-.24 1.7a5.2 5.2 0 0 0-1.52.88l-1.63-.66a.33.33 0 0 0-.4.15L1.34 5.6a.33.33 0 0 0 .08.42l1.38 1.08A5.5 5.5 0 0 0 2.64 8c0 .3.03.59.08.88L1.34 9.96a.33.33 0 0 0-.08.42l1.3 2.26c.08.14.25.2.4.15l1.63-.66c.47.35.98.65 1.52.88l.24 1.7c.03.16.17.28.33.28h2.6a.33.33 0 0 0 .33-.28l.24-1.7a5.2 5.2 0 0 0 1.52-.88l1.63.66c.15.06.32 0 .4-.15l1.3-2.26a.33.33 0 0 0-.08-.42l-1.38-1.08c.05-.29.08-.58.08-.88z" />
        </svg>
      </button>
      {open && (
        <div className={styles.dropdown}>
          <label className={styles.toggleItem}>
            <span className={styles.toggleLabel}>
              <span className={styles.toggleTitle}>Anonymous usage data</span>
              <span className={styles.toggleDesc}>Help improve Kiloforge</span>
            </span>
            <button
              role="switch"
              aria-checked={analyticsEnabled}
              className={`${styles.toggle} ${analyticsEnabled ? styles.toggleOn : ""}`}
              onClick={handleToggleAnalytics}
            >
              <span className={styles.toggleThumb} />
            </button>
          </label>
          <div className={styles.separator} />
          <div className={styles.scaleItem}>
            <div className={styles.scaleHeader}>
              <span className={styles.toggleTitle}>Display scale</span>
              <span className={styles.scaleValue}>{scale}%</span>
            </div>
            <input
              type="range"
              role="slider"
              min={75}
              max={150}
              step={5}
              value={scale}
              onChange={(e) => setScale(Number(e.target.value))}
              className={styles.scaleSlider}
            />
            <div className={styles.scaleLabels}>
              <span>Compact</span>
              <span>Default</span>
              <span>Large</span>
            </div>
          </div>
          <div className={styles.separator} />
          <button className={styles.item} onClick={handleTakeTour}>
            Take Tour
          </button>
        </div>
      )}
    </div>
  );
}
