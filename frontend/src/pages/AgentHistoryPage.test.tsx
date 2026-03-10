import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { AgentHistoryPage } from "./AgentHistoryPage";

vi.mock("../hooks/useAgents", () => ({
  useAgents: () => ({
    agents: [],
    loading: false,
    remainingCount: 0,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
}));

vi.mock("../hooks/useAgentActions", () => ({
  useAgentActions: () => ({
    stop: { mutate: vi.fn(), isPending: false },
    resume: { mutate: vi.fn(), isPending: false },
    replace: { mutate: vi.fn(), isPending: false },
    del: { mutate: vi.fn(), isPending: false },
  }),
  canStop: vi.fn(() => false),
  canResume: vi.fn(() => false),
  canReplace: vi.fn(() => false),
  canDelete: vi.fn(() => false),
}));

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AgentHistoryPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AgentHistoryPage", () => {
  it("renders status filter chips including new recovery statuses", () => {
    renderPage();
    expect(screen.getByText("suspended")).toBeInTheDocument();
    expect(screen.getByText("resume-failed")).toBeInTheDocument();
    expect(screen.getByText("force-killed")).toBeInTheDocument();
    expect(screen.getByText("replaced")).toBeInTheDocument();
  });

  it("renders all expected filter options", () => {
    renderPage();
    const expected = ["All", "running", "waiting", "completed", "failed", "stopped", "suspended", "resume-failed", "force-killed", "replaced"];
    for (const label of expected) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });
});
