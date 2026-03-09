import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { OverviewPage } from "./OverviewPage";
import type { Agent, QuotaResponse, Track } from "../types/api";

// Mock hooks/modules used by OverviewPage
vi.mock("../hooks/useProjects", () => ({
  useProjects: () => ({
    projects: mockProjects,
    loading: false,
    adding: false,
    removing: null,
    error: null,
    addProject: vi.fn().mockResolvedValue(true),
    removeProject: vi.fn().mockResolvedValue(true),
    clearError: vi.fn(),
    handleProjectUpdate: vi.fn(),
    handleProjectRemoved: vi.fn(),
  }),
  useSSHKeys: () => ({
    keys: [],
    loading: false,
    fetchKeys: vi.fn(),
  }),
}));

vi.mock("../hooks/useTraces", () => ({
  useTraces: () => ({ traces: [], loading: false, handleTraceUpdate: vi.fn() }),
}));

vi.mock("../components/tour/TourProvider", () => ({
  useTourContextSafe: () => null,
}));

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn().mockResolvedValue(null),
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

let mockProjects = [
  { slug: "my-project", repo_name: "my-project", origin_remote: "https://github.com/user/my-project.git", active: true },
];

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: "agent-1",
    role: "developer",
    ref: "",
    status: "running",
    session_id: "sess-1",
    pid: 1234,
    worktree_dir: "/tmp/wt",
    log_file: "/tmp/log",
    started_at: "2026-03-10T00:00:00Z",
    updated_at: "2026-03-10T00:00:00Z",
    ...overrides,
  };
}

const mockQuota: QuotaResponse = {
  input_tokens: 50000,
  output_tokens: 10000,
  cache_read_tokens: 5000,
  cache_creation_tokens: 1000,
  estimated_cost_usd: 1.5,
  agent_count: 2,
  rate_limited: false,
};

function renderPage(props?: Partial<Parameters<typeof OverviewPage>[0]>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const defaultProps = {
    agents: [makeAgent(), makeAgent({ id: "agent-2", role: "reviewer", status: "completed" })],
    agentsLoading: false,
    quota: mockQuota,
    tracks: [
      { id: "track-1", title: "Track One", status: "complete", project: "my-project" },
      { id: "track-2", title: "Track Two", status: "in-progress", project: "my-project" },
    ] as Track[],
    onViewLog: vi.fn(),
    onAttach: vi.fn(),
    onSpawnInteractive: vi.fn(),
    spawningInteractive: false,
    ...props,
  };

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OverviewPage {...defaultProps} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OverviewPage", () => {
  beforeEach(() => {
    mockProjects = [
      { slug: "my-project", repo_name: "my-project", origin_remote: "https://github.com/user/my-project.git", active: true },
    ];
  });

  it("renders stat cards section", () => {
    renderPage();
    // StatCards renders agent count; section header says "Agents"
    expect(screen.getAllByText("Agents").length).toBeGreaterThanOrEqual(1);
  });

  it("renders agent grid with agents", () => {
    renderPage();
    // Agent names/ids should appear
    expect(screen.getByText("agent-1")).toBeInTheDocument();
    expect(screen.getByText("agent-2")).toBeInTheDocument();
  });

  it("shows loading state when agents are loading", () => {
    renderPage({ agentsLoading: true });
    expect(screen.getByText("Loading agents...")).toBeInTheDocument();
  });

  it("renders projects section with project rows", () => {
    renderPage();
    expect(screen.getByText("Projects")).toBeInTheDocument();
    expect(screen.getByText("my-project")).toBeInTheDocument();
  });

  it("shows empty state when no projects", () => {
    mockProjects = [];
    renderPage();
    expect(screen.getByText(/No projects registered yet/)).toBeInTheDocument();
  });

  it("renders spawn interactive button", () => {
    renderPage();
    expect(screen.getByText("Start Interactive Agent")).toBeInTheDocument();
  });

  it("disables spawn button when spawning", () => {
    renderPage({ spawningInteractive: true });
    expect(screen.getByText("Starting...")).toBeDisabled();
  });

  it("renders filter chips for roles and statuses", () => {
    renderPage();
    // Role filter chips are buttons
    const buttons = screen.getAllByRole("button");
    const chipLabels = buttons.map((b) => b.textContent);
    expect(chipLabels).toContain("developer");
    expect(chipLabels).toContain("reviewer");
    expect(chipLabels).toContain("interactive");
    expect(chipLabels).toContain("Active");
    expect(chipLabels).toContain("Completed");
    expect(chipLabels).toContain("Failed");
  });

  it("filters agents by role when chip is clicked", async () => {
    const user = userEvent.setup();
    renderPage();
    // Both agents visible initially
    expect(screen.getByText("agent-1")).toBeInTheDocument();
    expect(screen.getByText("agent-2")).toBeInTheDocument();

    // Click "developer" filter chip (use getAllByText since role also appears in agent card)
    const devChips = screen.getAllByText("developer");
    // The chip button is the one with role=button
    const chipButton = devChips.find((el) => el.tagName === "BUTTON")!;
    await user.click(chipButton);
    // agent-1 is developer, agent-2 is reviewer
    expect(screen.getByText("agent-1")).toBeInTheDocument();
    expect(screen.queryByText("agent-2")).not.toBeInTheDocument();
  });

  it("renders tracks section", () => {
    renderPage();
    expect(screen.getByText("All Tracks")).toBeInTheDocument();
  });

  it("renders traces section", () => {
    renderPage();
    expect(screen.getByText("Traces")).toBeInTheDocument();
  });
});
