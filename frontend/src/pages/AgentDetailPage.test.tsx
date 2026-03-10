import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AgentDetailPage } from "./AgentDetailPage";
import type { Agent } from "../types/api";

const mockAgent: Agent = {
  id: "agent-abc",
  name: "Clever Fox",
  role: "developer",
  ref: "track-123",
  status: "running",
  session_id: "sess-1",
  pid: 4567,
  worktree_dir: "/tmp/wt-1",
  log_file: "/tmp/agent.log",
  started_at: "2026-03-10T00:00:00Z",
  updated_at: "2026-03-10T00:01:00Z",
  uptime_seconds: 120,
  input_tokens: 50000,
  output_tokens: 10000,
  cache_read_tokens: 5000,
  cache_creation_tokens: 1000,
  estimated_cost_usd: 1.25,
  model: "claude-opus-4-6",
};

let currentAgent: Agent | null = mockAgent;

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn().mockImplementation((url: string) => {
    if (url.includes("/api/agents/") && !url.includes("/log")) {
      if (!currentAgent) return Promise.reject(new Error("Agent not found"));
      return Promise.resolve(currentAgent);
    }
    return Promise.resolve(null);
  }),
  FetchError: class FetchError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, body: unknown) {
      super(`Request failed with status ${status}`);
      this.status = status;
      this.body = body;
    }
  },
}));

vi.mock("../hooks/useTracks", () => ({
  useTracks: () => ({
    tracks: [{ id: "track-123", title: "Test Track", status: "in-progress", project: "my-proj" }],
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

vi.mock("../hooks/useAgentWebSocket", () => ({
  useAgentWebSocket: () => ({
    messages: [],
    sendMessage: vi.fn(),
    status: "disconnected" as const,
    agentStatus: null,
  }),
}));

// Mock global fetch for log data
vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
  json: () => Promise.resolve({ lines: ["log line 1", "log line 2"], total: 2 }),
}));

function renderPage(agentId = "agent-abc") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/agents/${agentId}`]}>
        <Routes>
          <Route path="/agents/:id" element={<AgentDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AgentDetailPage", () => {
  beforeEach(() => {
    currentAgent = mockAgent;
  });

  it("renders agent name and metadata", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Clever Fox")).toBeInTheDocument();
    });
    expect(screen.getByText("developer")).toBeInTheDocument();
  });

  it("renders model info", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("claude-opus-4-6")).toBeInTheDocument();
    });
  });

  it("renders track reference with board link", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText(/track-123/)).toBeInTheDocument();
    });
    expect(screen.getByText("View on Board")).toBeInTheDocument();
  });

  it("renders PID and worktree info", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("4567")).toBeInTheDocument();
    });
    expect(screen.getByText("/tmp/wt-1")).toBeInTheDocument();
  });

  it("shows stop button for running agent", async () => {
    currentAgent = { ...mockAgent, status: "running" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Stop")).toBeInTheDocument();
    });
  });

  it("shows delete button for completed agent", async () => {
    currentAgent = { ...mockAgent, status: "completed" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", () => {
    // Before data loads
    renderPage();
    expect(screen.getByText("Loading agent...")).toBeInTheDocument();
  });

  it("shows error state when agent fetch fails", async () => {
    currentAgent = null;
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Agent not found")).toBeInTheDocument();
    });
  });

  it("renders log output section", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Log Output")).toBeInTheDocument();
    });
    expect(screen.getByText("Follow")).toBeInTheDocument();
  });

  it("renders log lines after fetch", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText(/log line 1/)).toBeInTheDocument();
    });
  });

  it("shows resume_error for resume-failed agent", async () => {
    currentAgent = { ...mockAgent, status: "resume-failed", resume_error: "worktree missing" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText(/worktree missing/)).toBeInTheDocument();
    });
  });

  it("shows suspended_at and shutdown_reason for suspended agent", async () => {
    currentAgent = {
      ...mockAgent,
      status: "suspended",
      suspended_at: "2026-03-10T12:00:00Z",
      shutdown_reason: "orchestrator shutdown",
    };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText(/orchestrator shutdown/)).toBeInTheDocument();
    });
    expect(screen.getByText("Suspended At")).toBeInTheDocument();
  });

  it("shows resume button for suspended agent", async () => {
    currentAgent = { ...mockAgent, status: "suspended" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Resume")).toBeInTheDocument();
    });
  });

  it("shows replace button for resume-failed agent", async () => {
    currentAgent = { ...mockAgent, status: "resume-failed", resume_error: "session expired" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Replace")).toBeInTheDocument();
    });
  });

  it("shows replaced banner for replaced agent", async () => {
    currentAgent = { ...mockAgent, status: "replaced" };
    renderPage();
    await waitFor(() => {
      expect(screen.getByText(/replaced by a new agent/)).toBeInTheDocument();
    });
  });
});
