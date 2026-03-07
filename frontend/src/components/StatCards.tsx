import type { QuotaResponse } from "../types/api";
import { formatUSD, formatTokens } from "../utils/format";
import styles from "./StatCards.module.css";

export function StatCards({ agentCount, quota }: { agentCount: number; quota: QuotaResponse | null }) {
  const rateLimited = quota?.rate_limited ?? false;

  return (
    <section className={styles.stats}>
      <div className={styles.card}>
        <div className={styles.label}>Agents</div>
        <div className={styles.value}>{agentCount}</div>
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Total Cost</div>
        <div className={styles.value}>{formatUSD(quota?.total_cost_usd ?? 0)}</div>
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Rate Limit</div>
        <div className={`${styles.value} ${rateLimited ? styles.danger : ""}`}>
          {rateLimited ? "LIMITED" : "OK"}
        </div>
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Tokens In / Out</div>
        <div className={styles.value}>
          {formatTokens(quota?.input_tokens ?? 0)} / {formatTokens(quota?.output_tokens ?? 0)}
        </div>
      </div>
    </section>
  );
}
