import type { QuotaResponse } from "../types/api";
import { formatUSD, formatTokens } from "../utils/format";
import styles from "./StatCards.module.css";

export function StatCards({ agentCount, quota }: { agentCount: number; quota: QuotaResponse | null }) {
  const rateLimited = quota?.rate_limited ?? false;
  const cacheRead = quota?.cache_read_tokens ?? 0;
  const cacheCreate = quota?.cache_creation_tokens ?? 0;
  const hasCache = cacheRead > 0 || cacheCreate > 0;

  return (
    <section className={styles.stats}>
      <div className={styles.card}>
        <div className={styles.label}>Agents</div>
        <div className={styles.value}>{agentCount}</div>
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Tokens</div>
        <div className={styles.value}>
          {formatTokens(quota?.input_tokens ?? 0)} / {formatTokens(quota?.output_tokens ?? 0)}
        </div>
        {hasCache && (
          <div className={styles.sub}>
            cache: {formatTokens(cacheRead)} read
            {cacheCreate > 0 && <> · {formatTokens(cacheCreate)} create</>}
          </div>
        )}
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Rate Limit</div>
        <div className={`${styles.value} ${rateLimited ? styles.danger : ""}`}>
          {rateLimited ? "LIMITED" : "OK"}
        </div>
      </div>
      <div className={styles.card}>
        <div className={styles.label}>Est. API Cost</div>
        <div className={`${styles.value} ${styles.secondary}`}>
          {formatUSD(quota?.estimated_cost_usd ?? 0)}
        </div>
      </div>
    </section>
  );
}
