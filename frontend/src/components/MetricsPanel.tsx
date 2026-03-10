import type { QuotaResponse } from "../types/api";
import { formatUSD, formatTokens } from "../utils/format";
import { GaugeCard } from "./GaugeCard";
import styles from "./MetricsPanel.module.css";

interface MetricsPanelProps {
  agentCount: number;
  quota: QuotaResponse | null;
}

export function MetricsPanel({ agentCount, quota }: MetricsPanelProps) {
  const budgetUsd = quota?.budget_usd ?? 0;
  const budgetPct = quota?.budget_used_pct ?? 0;
  const tokPerMin = quota?.rate_tokens_per_min ?? 0;
  const costPerHour = quota?.rate_cost_per_hour ?? 0;
  const costUsd = quota?.estimated_cost_usd ?? 0;

  const budgetSubtitle = budgetUsd > 0
    ? `of ${formatUSD(budgetUsd)}`
    : undefined;

  const costSubtitle = costPerHour > 0
    ? `${formatUSD(costPerHour)}/hr`
    : undefined;

  return (
    <section className={styles.grid}>
      <GaugeCard
        label="Budget"
        value={Math.round(budgetPct)}
        max={budgetUsd > 0 ? 100 : undefined}
        unit={budgetUsd > 0 ? "" : "$"}
        subtitle={budgetSubtitle ?? (budgetUsd === 0 ? formatUSD(costUsd) : undefined)}
      />
      <GaugeCard
        label="Token Rate"
        value={tokPerMin}
        max={tokPerMin > 0 ? Math.max(tokPerMin * 1.5, 1000) : undefined}
        subtitle={tokPerMin > 0 ? `${formatTokens(tokPerMin)} tok/min` : "no data"}
      />
      <GaugeCard
        label="Cost Rate"
        value={costPerHour > 0 ? Math.round(costPerHour * 100) : 0}
        max={costPerHour > 0 ? Math.round(costPerHour * 100 * 1.5) : undefined}
        unit="$"
        subtitle={costSubtitle ?? formatUSD(costUsd)}
      />
      <GaugeCard
        label="Agents"
        value={agentCount}
        subtitle={agentCount === 1 ? "1 active" : `${agentCount} active`}
      />
    </section>
  );
}
