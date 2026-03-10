import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReliabilityEventTable } from "./ReliabilityEventTable";

vi.mock("../../hooks/useReliability", () => ({
  useReliabilityEvents: () => ({
    events: [
      {
        id: "evt-1",
        event_type: "agent_timeout",
        severity: "error",
        agent_id: "agent-abc",
        scope: "track-1",
        detail: { timeout_seconds: 120 },
        created_at: new Date().toISOString(),
      },
      {
        id: "evt-2",
        event_type: "lock_contention",
        severity: "warn",
        created_at: new Date(Date.now() - 3600000).toISOString(),
      },
    ],
    items: [],
    isLoading: false,
    totalCount: 2,
    shownCount: 2,
    remainingCount: 0,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
}));

function renderTable() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ReliabilityEventTable />
    </QueryClientProvider>,
  );
}

describe("ReliabilityEventTable", () => {
  it("renders filter chips for event types", () => {
    renderTable();
    expect(screen.getByText("All Types")).toBeInTheDocument();
    // Type labels appear in both chips and table rows
    expect(screen.getAllByText("agent timeout").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("lock contention").length).toBeGreaterThanOrEqual(1);
  });

  it("renders severity filter chips", () => {
    renderTable();
    expect(screen.getByText("All Severities")).toBeInTheDocument();
    // "warn" and "error" appear in both chips and table rows
    expect(screen.getAllByText("warn").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("error").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("critical")).toBeInTheDocument();
  });

  it("renders event rows in the table", () => {
    renderTable();
    expect(screen.getByText("agent-abc")).toBeInTheDocument();
    expect(screen.getByText("track-1")).toBeInTheDocument();
  });

  it("renders table headers", () => {
    renderTable();
    expect(screen.getByText("Time")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("Scope")).toBeInTheDocument();
    expect(screen.getByText("Detail")).toBeInTheDocument();
  });
});
