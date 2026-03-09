import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import { AdminPanel } from "../components/AdminPanel";
import { AgentTerminal } from "../components/AgentTerminal";
import appStyles from "../App.module.css";

interface AdminContainerProps {
  slug: string;
  actionsDisabled: boolean;
  disabledReason?: string;
  onSetupRequired: () => void;
  onSkillsRequired: () => void;
}

export function AdminContainer({ slug, actionsDisabled, disabledReason, onSetupRequired, onSkillsRequired }: AdminContainerProps) {
  const queryClient = useQueryClient();
  const [adminAgentId, setAdminAgentId] = useState<string | null>(null);

  const handleAdminTerminalClose = useCallback(() => {
    setAdminAgentId(null);
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug) });
  }, [queryClient, slug]);

  return (
    <>
      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Admin Operations</h2>
        <AdminPanel
          projectSlug={slug}
          running={adminAgentId !== null}
          disabled={actionsDisabled}
          disabledReason={disabledReason}
          onStartOperation={setAdminAgentId}
          onSetupRequired={onSetupRequired}
          onSkillsRequired={onSkillsRequired}
        />
      </section>

      {adminAgentId && (
        <AgentTerminal agentId={adminAgentId} onClose={handleAdminTerminalClose} />
      )}
    </>
  );
}
