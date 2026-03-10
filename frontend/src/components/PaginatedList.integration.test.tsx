import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { PaginatedList } from "./PaginatedList";
import { AgentGrid } from "./AgentGrid";
import { TrackList } from "./TrackList";
import type { Agent, Track } from "../types/api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(MemoryRouter, null, children),
    );
}

function makeAgents(count: number): Agent[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `agent-${i}`,
    role: "developer",
    ref: "",
    status: i < 2 ? "running" : "completed",
    session_id: `sess-${i}`,
    pid: 1000 + i,
    worktree_dir: `/tmp/wt-${i}`,
    log_file: `/tmp/log-${i}`,
    started_at: "2026-03-10T00:00:00Z",
    updated_at: "2026-03-10T00:00:00Z",
  }));
}

describe("PaginatedList integration", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows +N more for AgentGrid when there are remaining items", () => {
    const agents = makeAgents(3);
    render(
      <PaginatedList
        remainingCount={47}
        hasNextPage={true}
        isFetchingNextPage={false}
        onLoadMore={() => {}}
      >
        <AgentGrid agents={agents} onViewLog={() => {}} />
      </PaginatedList>,
      { wrapper: createWrapper() },
    );

    expect(screen.getByText("agent-0")).toBeInTheDocument();
    expect(screen.getByText("+47 more")).toBeInTheDocument();
  });

  it("calls onLoadMore when +N button is clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    const agents = makeAgents(2);

    render(
      <PaginatedList
        remainingCount={10}
        hasNextPage={true}
        isFetchingNextPage={false}
        onLoadMore={onLoadMore}
      >
        <AgentGrid agents={agents} onViewLog={() => {}} />
      </PaginatedList>,
      { wrapper: createWrapper() },
    );

    await user.click(screen.getByText("+10 more"));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("shows custom remaining label", () => {
    const tracks: Track[] = [
      { id: "t1", title: "Track One", status: "complete" },
      { id: "t2", title: "Track Two", status: "pending" },
    ];

    render(
      <PaginatedList
        remainingCount={8}
        remainingLabel="completed"
        hasNextPage={true}
        isFetchingNextPage={false}
        onLoadMore={() => {}}
      >
        <TrackList tracks={tracks} />
      </PaginatedList>,
      { wrapper: createWrapper() },
    );

    expect(screen.getByText("+8 completed")).toBeInTheDocument();
  });

  it("shows loading state during fetch", () => {
    render(
      <PaginatedList
        remainingCount={5}
        hasNextPage={true}
        isFetchingNextPage={true}
        onLoadMore={() => {}}
      >
        <AgentGrid agents={makeAgents(1)} onViewLog={() => {}} />
      </PaginatedList>,
      { wrapper: createWrapper() },
    );

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText(/more/)).not.toBeInTheDocument();
  });

  it("hides footer when all items are loaded", () => {
    render(
      <PaginatedList
        remainingCount={0}
        hasNextPage={false}
        isFetchingNextPage={false}
        onLoadMore={() => {}}
      >
        <AgentGrid agents={makeAgents(3)} onViewLog={() => {}} />
      </PaginatedList>,
      { wrapper: createWrapper() },
    );

    expect(screen.queryByText(/more/)).not.toBeInTheDocument();
    expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
  });
});
