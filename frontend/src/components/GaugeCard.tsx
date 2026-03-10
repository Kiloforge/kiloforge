import styles from "./GaugeCard.module.css";

interface GaugeCardProps {
  label: string;
  value: number;
  max?: number;
  unit?: string;
  subtitle?: string;
}

const ARC_RADIUS = 40;
const ARC_STROKE = 6;
const ARC_LENGTH = Math.PI * ARC_RADIUS; // semicircle

function colorClass(pct: number): string {
  if (pct >= 85) return styles.red;
  if (pct >= 60) return styles.yellow;
  return styles.green;
}

export function GaugeCard({ label, value, max, unit, subtitle }: GaugeCardProps) {
  const hasGauge = max !== undefined && max > 0;
  const pct = hasGauge ? Math.min((value / max) * 100, 100) : 0;
  const offset = hasGauge ? ARC_LENGTH * (1 - pct / 100) : ARC_LENGTH;

  const formatted = unit ? `${unit}${value}` : String(value);

  if (!hasGauge) {
    return (
      <div className={styles.card}>
        <div className={styles.label}>{label}</div>
        <div className={styles.value}>{formatted}</div>
        {subtitle && <div className={styles.sub}>{subtitle}</div>}
      </div>
    );
  }

  return (
    <div className={styles.card}>
      <div className={styles.gaugeWrap}>
        <svg viewBox="0 0 100 56" className={styles.svg}>
          <path
            d={`M ${50 - ARC_RADIUS} 50 A ${ARC_RADIUS} ${ARC_RADIUS} 0 0 1 ${50 + ARC_RADIUS} 50`}
            fill="none"
            stroke="var(--border)"
            strokeWidth={ARC_STROKE}
            strokeLinecap="round"
          />
          <path
            data-testid="gauge-fill"
            className={`${styles.fill} ${colorClass(pct)}`}
            d={`M ${50 - ARC_RADIUS} 50 A ${ARC_RADIUS} ${ARC_RADIUS} 0 0 1 ${50 + ARC_RADIUS} 50`}
            fill="none"
            strokeWidth={ARC_STROKE}
            strokeLinecap="round"
            strokeDasharray={`${ARC_LENGTH},${ARC_LENGTH}`}
            strokeDashoffset={offset}
          />
        </svg>
        <div className={styles.gaugeValue}>{formatted}</div>
      </div>
      <div className={styles.label}>{label}</div>
      {subtitle && <div className={styles.sub}>{subtitle}</div>}
    </div>
  );
}
