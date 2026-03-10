import { Link } from "react-router-dom";
import { useReliabilitySummary } from "../hooks/useReliability";
import { ReliabilitySummaryCards } from "../components/reliability/ReliabilitySummaryCards";
import { ReliabilityChart } from "../components/reliability/ReliabilityChart";
import { ReliabilityEventTable } from "../components/reliability/ReliabilityEventTable";
import styles from "./ReliabilityPage.module.css";

export function ReliabilityPage() {
  const { summary, loading } = useReliabilitySummary();

  const totalEvents = summary
    ? Object.values(summary.totals).reduce((sum, c) => sum + c, 0)
    : 0;

  const isEmpty = !loading && totalEvents === 0;

  return (
    <div className={styles.page}>
      <div className={styles.topBar}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <h2 className={styles.title}>Reliability</h2>
      </div>

      {isEmpty ? (
        <div className={styles.section}>
          <div className={styles.emptyState}>
            <div className={styles.emptyIcon}>{"\u2713"}</div>
            <h3 className={styles.emptyTitle}>No reliability events</h3>
            <p className={styles.emptyText}>
              When issues like lock contention, agent timeouts, merge conflicts,
              or quota limits occur, they will appear here with severity levels
              and historical trends.
            </p>
          </div>
        </div>
      ) : (
        <>
          <section className={styles.section}>
            <h3 className={styles.sectionTitle}>Summary</h3>
            <ReliabilitySummaryCards summary={summary} />
          </section>

          <section className={styles.section}>
            <h3 className={styles.sectionTitle}>Event Frequency</h3>
            <ReliabilityChart />
          </section>

          <section className={styles.section}>
            <h3 className={styles.sectionTitle}>Event Log</h3>
            <ReliabilityEventTable />
          </section>
        </>
      )}
    </div>
  );
}
