import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { AgentCard } from "./AgentCard";
import type { Agent } from "../types/api";

vi.mock("../hooks/useTracks", () => ({
  useTracks: () => ({
    tracks: [{ id: "track-abc", title: "Test Track", status: "in-progress", project: "my-proj" }],
    loading: false,
    handleTrackUpdate: vi.fn(),
    handleTrackRemoved: vi.fn(),
  }),
}));

vi.mock("../hooks/useAgentActions", () => ({
  useAgentActions: () => ({
    stop: { mutate: vi.fn(), isPending: false },
    resume: { mutate: vi.fn(), isPending: false },
    replace: { mutate: vi.fn(), isPending: false },
    del: { mutate: vi.fn(), isPending: false },
  }),
  canStop: (a: Agent) => a.status === "running" || a.status === "waiting",
  canResume: (a: Agent) => {
    if (a.status === "suspended" || a.status === "force-killed") return true;
    return (a.status === "stopped" || a.status === "completed" || a.status === "failed") && a.role === "interactive";
  },
  canReplace: (a: Agent) => a.status === "resume-failed" || a.status === "force-killed",
  canDelete: (a: Agent) => a.status !== "running" && a.status !== "waiting",
}));

const baseAgent: Agent = {
  id: "agent-xyz",
  name: "Sleepy Owl",
  role: "developer",
  ref: "track-abc",
  status: "running",
  session_id: "sess-1",
  pid: 1234,
  worktree_dir: "/tmp/wt-1",
  log_file: "/tmp/agent.log",
  started_at: "2026-03-10T00:00:00Z",
  updated_at: "2026-03-10T00:01:00Z",
  uptime_seconds: 300,
  input_tokens: 25000,
  output_tokens: 5000,
  estimated_cost_usd: 0.75,
  model: "claude-opus-4-6",
};

function renderCard(agentOverrides: Partial<Agent> = {}, props?: { onAttach?: (id: string) => void }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const agent = { ...baseAgent, ...agentOverrides };

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AgentCard agent={agent} onViewLog={vi.fn()} onAttach={props?.onAttach} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AgentCard", () => {
  it("renders agent name and id", () => {
    renderCard();
    expect(screen.getByText("Sleepy Owl")).toBeInTheDocument();
    expect(screen.getByText("agent-xyz")).toBeInTheDocument();
  });

  it("renders agent role badge", () => {
    renderCard();
    expect(screen.getByText("developer")).toBeInTheDocument();
  });

  it("renders status badge", () => {
    renderCard();
    expect(screen.getByText("running")).toBeInTheDocument();
  });

  it("renders token counts", () => {
    renderCard();
    expect(screen.getByText(/25\.0k/)).toBeInTheDocument();
  });

  it("renders model info", () => {
    renderCard();
    expect(screen.getByText(/claude-opus-4-6/)).toBeInTheDocument();
  });

  it("renders cost estimate", () => {
    renderCard();
    expect(screen.getByText(/\$0\.75/)).toBeInTheDocument();
  });

  it("links agent name to detail page", () => {
    renderCard();
    const link = screen.getByText("Sleepy Owl");
    expect(link.closest("a")).toHaveAttribute("href", "/agents/agent-xyz");
  });

  it("renders track ref link to project", () => {
    renderCard();
    const refLink = screen.getByText(/ref: track-abc/);
    expect(refLink.closest("a")).toHaveAttribute("href", "/projects/my-proj");
  });

  it("shows view log button when log_file is present", () => {
    renderCard();
    expect(screen.getByText("View Log")).toBeInTheDocument();
  });

  it("shows stop button for running agent", () => {
    renderCard({ status: "running" });
    expect(screen.getByText("Stop")).toBeInTheDocument();
  });

  it("shows delete button for completed agent", () => {
    renderCard({ status: "completed" });
    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("shows attach button for interactive agent with onAttach", () => {
    renderCard({ role: "interactive" }, { onAttach: vi.fn() });
    expect(screen.getByText("Attach")).toBeInTheDocument();
  });

  it("does not show attach for non-interactive agent", () => {
    renderCard({ role: "developer" }, { onAttach: vi.fn() });
    expect(screen.queryByText("Attach")).not.toBeInTheDocument();
  });

  it("renders uptime when present", () => {
    renderCard({ uptime_seconds: 3661 });
    // formatUptime(3661) = "1h 1m"
    expect(screen.getByText(/uptime:/)).toBeInTheDocument();
  });

  it("uses agent id as name when name is missing", () => {
    renderCard({ name: undefined });
    // The link text should be the id
    const link = screen.getByText("agent-xyz");
    expect(link.closest("a")).toHaveAttribute("href", "/agents/agent-xyz");
  });

  it("handles missing optional fields gracefully", () => {
    renderCard({
      ref: "",
      model: undefined,
      uptime_seconds: undefined,
      estimated_cost_usd: undefined,
      input_tokens: 0,
      output_tokens: 0,
    });
    // Should still render without errors
    expect(screen.getByText("Sleepy Owl")).toBeInTheDocument();
    // No ref link
    expect(screen.queryByText(/ref:/)).not.toBeInTheDocument();
  });

  it("shows resume button for suspended developer agent", () => {
    renderCard({ status: "suspended", role: "developer" });
    expect(screen.getByText("Resume")).toBeInTheDocument();
  });

  it("shows replace button for resume-failed agent", () => {
    renderCard({ status: "resume-failed", role: "developer" });
    expect(screen.getByText("Replace")).toBeInTheDocument();
  });

  it("does not show resume for resume-failed agent", () => {
    renderCard({ status: "resume-failed", role: "developer" });
    expect(screen.queryByText("Resume")).not.toBeInTheDocument();
  });

  it("shows shutdown reason for suspended agent", () => {
    renderCard({ status: "suspended", shutdown_reason: "cortex shutdown" });
    expect(screen.getByText(/cortex shutdown/)).toBeInTheDocument();
  });
});
