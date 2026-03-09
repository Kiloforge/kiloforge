import { useCallback } from "react";
import { useOriginSync } from "../hooks/useOriginSync";
import { SyncPanel } from "../components/SyncPanel";
import appStyles from "../App.module.css";

interface SyncContainerProps {
  slug: string;
}

export function SyncContainer({ slug }: SyncContainerProps) {
  const { syncStatus, loading, pushing, pulling, error, push, pull, refresh, clearError } = useOriginSync(slug);

  const handlePush = useCallback((remoteBranch: string) => {
    push({ remote_branch: remoteBranch });
  }, [push]);

  const handlePull = useCallback((remoteBranch?: string) => {
    pull(remoteBranch);
  }, [pull]);

  return (
    <section className={appStyles.panel}>
      <h2 className={appStyles.panelTitle}>Origin Sync</h2>
      <SyncPanel
        syncStatus={syncStatus}
        loading={loading}
        pushing={pushing}
        pulling={pulling}
        error={error}
        onPush={handlePush}
        onPull={handlePull}
        onRefresh={refresh}
        onClearError={clearError}
      />
    </section>
  );
}
