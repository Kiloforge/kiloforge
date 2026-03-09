import { useCallback, useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useBoard } from "../hooks/useBoard";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";
import { KanbanBoard } from "../components/KanbanBoard";
import { AgentTerminal } from "../components/AgentTerminal";
import { useTourContextSafe } from "../components/tour/TourProvider";
import { TOUR_STEPS } from "../components/tour/tourSteps";
import appStyles from "../App.module.css";
import styles from "../pages/ProjectPage.module.css";

interface BoardContainerProps {
  slug: string;
  actionsDisabled: boolean;
  disabledReason?: string;
  onConsentRequired: (retry: () => void) => void;
  onSkillsRequired: (retry: () => void) => void;
  onSetupRequired: (slug: string, retry: () => void) => void;
}

export function BoardContainer({ slug, actionsDisabled, disabledReason, onConsentRequired, onSkillsRequired, onSetupRequired }: BoardContainerProps) {
  const queryClient = useQueryClient();
  const { board, loading: boardLoading, moveCard, syncBoard, syncing } = useBoard(slug);
  const tour = useTourContextSafe();

  const [showPrompt, setShowPrompt] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [terminalAgentId, setTerminalAgentId] = useState<string | null>(null);

  // Tour: auto-show prompt and prefill when on generate-tracks step
  const tourStep = tour?.isActive ? TOUR_STEPS[tour.currentStep] : null;
  useEffect(() => {
    if (tourStep?.id === "generate-tracks" && !showPrompt) {
      setShowPrompt(true);
      setPrompt("Add user authentication with login, registration, and password reset");
    }
  }, [tourStep?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const generateMutation = useMutation({
    mutationFn: (p: string) =>
      fetcher<{ agent_id: string; ws_url: string }>("/api/tracks/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt: p, project: slug }),
      }),
    onSuccess: (data) => {
      setTerminalAgentId(data.agent_id);
      setShowPrompt(false);
      setPrompt("");
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 403) {
        onConsentRequired(() => handleGenerateTracks());
      } else if (err instanceof FetchError && err.status === 412) {
        onSkillsRequired(() => handleGenerateTracks());
      } else if (err instanceof FetchError && err.status === 428) {
        onSetupRequired(slug, () => handleGenerateTracks());
      }
    },
  });

  const handleGenerateTracks = useCallback(() => {
    if (!prompt.trim()) return;
    generateMutation.mutate(prompt.trim());
  }, [prompt, generateMutation]);

  const deleteMutation = useMutation({
    mutationFn: (trackId: string) =>
      fetcher<void>(
        `/api/tracks/${encodeURIComponent(trackId)}?project=${encodeURIComponent(slug)}`,
        { method: "DELETE" },
      ),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.board(slug) });
    },
  });

  const handleDeleteTrack = useCallback(
    (trackId: string) => {
      deleteMutation.mutate(trackId);
    },
    [deleteMutation],
  );

  const handleTerminalClose = useCallback(() => {
    setTerminalAgentId(null);
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug) });
  }, [queryClient, slug]);

  return (
    <>
      <section className={appStyles.panel} data-tour="board-section">
        <div className={styles.boardHeader}>
          <h2 className={appStyles.panelTitle}>Board</h2>
          <div className={styles.boardActions}>
            <button
              className={styles.syncBtn}
              onClick={syncBoard}
              disabled={syncing || actionsDisabled}
              title={disabledReason}
            >
              {syncing ? "Syncing..." : "Sync"}
            </button>
            <button
              className={styles.generateBtn}
              onClick={() => { if (!actionsDisabled) setShowPrompt((v) => !v); }}
              disabled={actionsDisabled}
              title={disabledReason}
              data-tour="generate-tracks"
            >
              Generate Tracks
            </button>
          </div>
        </div>
        {showPrompt && (
          <div className={styles.promptForm}>
            <textarea
              className={styles.promptInput}
              placeholder="Describe the features or changes you want to generate tracks for..."
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={3}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  handleGenerateTracks();
                }
              }}
            />
            <div className={styles.promptActions}>
              <button
                className={styles.promptSubmit}
                disabled={!prompt.trim() || generateMutation.isPending}
                onClick={handleGenerateTracks}
              >
                {generateMutation.isPending ? "Starting..." : "Generate"}
              </button>
              <button className={styles.promptCancel} onClick={() => { setShowPrompt(false); setPrompt(""); }}>
                Cancel
              </button>
            </div>
          </div>
        )}
        {boardLoading ? (
          <p className={appStyles.empty}>Loading board...</p>
        ) : (
          <KanbanBoard
            board={board ?? { columns: ["backlog", "approved", "in_progress", "in_review", "done"], cards: {} }}
            projectSlug={slug}
            onMoveCard={moveCard}
            onDeleteTrack={handleDeleteTrack}
          />
        )}
      </section>

      {terminalAgentId && (
        <AgentTerminal agentId={terminalAgentId} onClose={handleTerminalClose} />
      )}
    </>
  );
}
