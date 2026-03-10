import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { TrackDetailPage } from "./TrackDetailPage";
import type { TrackDetail } from "../types/api";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockFetcher = vi.fn();
vi.mock("../api/fetcher", () => ({
  fetcher: (...args: unknown[]) => mockFetcher(...args),
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

const pendingTrack: TrackDetail = {
  id: "track-1",
  title: "Test Feature Track",
  status: "pending",
  type: "feature",
  spec: "Some spec content",
  plan: "Some plan content",
  phases_total: 3,
  phases_completed: 0,
  tasks_total: 10,
  tasks_completed: 0,
};

const activeTrack: TrackDetail = {
  ...pendingTrack,
  status: "in-progress",
};

function renderPage(trackId = "track-1", slug = "test-proj") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/projects/${slug}/tracks/${trackId}`]}>
        <Routes>
          <Route path="/projects/:slug/tracks/:trackId" element={<TrackDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockFetcher.mockImplementation((url: string) => {
    if (url.includes("/api/tracks/")) return Promise.resolve(pendingTrack);
    return Promise.resolve(null);
  });
});

describe("TrackDetailPage", () => {
  it("shows approve/reject buttons for pending track", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Approve")).toBeInTheDocument();
    });
    expect(screen.getByText("Reject")).toBeInTheDocument();
  });

  it("does not show approve/reject buttons for non-pending track", async () => {
    mockFetcher.mockImplementation(() => Promise.resolve(activeTrack));
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Test Feature Track")).toBeInTheDocument();
    });
    expect(screen.queryByText("Approve")).not.toBeInTheDocument();
    expect(screen.queryByText("Reject")).not.toBeInTheDocument();
  });

  it("calls move API and navigates on approve", async () => {
    mockFetcher.mockImplementation((url: string) => {
      if (url.includes("/api/board/")) return Promise.resolve(undefined);
      return Promise.resolve(pendingTrack);
    });
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Approve")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Approve"));
    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith(
        "/api/board/test-proj/move",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ track_id: "track-1", to_column: "approved" }),
        }),
      );
    });
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/projects/test-proj");
    });
  });

  it("shows confirmation dialog on reject", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Reject")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Reject"));
    expect(screen.getByText("Delete this track?")).toBeInTheDocument();
    expect(screen.getByText("Yes, delete")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("calls delete API and navigates on confirm reject", async () => {
    mockFetcher.mockImplementation(() => Promise.resolve(pendingTrack));
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Reject")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Reject"));
    // Now the mockFetcher should handle delete
    mockFetcher.mockImplementation(() => Promise.resolve(undefined));
    await user.click(screen.getByText("Yes, delete"));
    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith(
        "/api/tracks/track-1?project=test-proj",
        expect.objectContaining({ method: "DELETE" }),
      );
    });
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/projects/test-proj");
    });
  });

  it("cancels reject confirmation", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Reject")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Reject"));
    expect(screen.getByText("Delete this track?")).toBeInTheDocument();
    await user.click(screen.getByText("Cancel"));
    expect(screen.queryByText("Delete this track?")).not.toBeInTheDocument();
    expect(screen.getByText("Approve")).toBeInTheDocument();
  });

  it("renders track spec and plan", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Specification")).toBeInTheDocument();
    });
    expect(screen.getByText("Some spec content")).toBeInTheDocument();
    expect(screen.getByText("Implementation Plan")).toBeInTheDocument();
    expect(screen.getByText("Some plan content")).toBeInTheDocument();
  });

  it("renders progress bar", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("0/10 tasks complete")).toBeInTheDocument();
    });
  });

  it("renders agent register when data present", async () => {
    const trackWithRegister: TrackDetail = {
      ...activeTrack,
      agent_register: {
        created_by: {
          role: "architect",
          agent_id: "arch-abc123def456",
          session_id: "sess-architect-1",
          timestamp: "2026-03-12T14:00:00Z",
        },
        claimed_by: {
          role: "developer",
          agent_id: "dev-xyz789",
          session_id: "sess-developer-1",
          worktree: "worker-3",
          branch: "feature/track-1",
          model: "claude-opus-4-6",
          timestamp: "2026-03-12T15:00:00Z",
        },
      },
    };
    mockFetcher.mockImplementation(() => Promise.resolve(trackWithRegister));
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Agent Register")).toBeInTheDocument();
    });
    expect(screen.getByText("architect")).toBeInTheDocument();
    expect(screen.getByText("developer")).toBeInTheDocument();
    expect(screen.getByText("worker-3")).toBeInTheDocument();
    expect(screen.getByText("feature/track-1")).toBeInTheDocument();
    expect(screen.getByText("sess-developer-1")).toBeInTheDocument();
    // Copy buttons should be present for session IDs
    expect(screen.getAllByText("Copy")).toHaveLength(2);
  });

  it("hides agent register when data absent", async () => {
    mockFetcher.mockImplementation(() => Promise.resolve(activeTrack));
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Test Feature Track")).toBeInTheDocument();
    });
    expect(screen.queryByText("Agent Register")).not.toBeInTheDocument();
  });

  it("renders traces section with links", async () => {
    const trackWithTraces: TrackDetail = {
      ...activeTrack,
      traces: [
        {
          trace_id: "trace-abc",
          root_name: "track/track-1",
          span_count: 5,
          start_time: "2026-03-12T14:00:00Z",
          end_time: "2026-03-12T14:01:00Z",
        },
        {
          trace_id: "trace-def",
          root_name: "build",
          span_count: 3,
          start_time: "2026-03-12T15:00:00Z",
          end_time: "2026-03-12T15:00:30Z",
        },
      ],
    };
    mockFetcher.mockImplementation(() => Promise.resolve(trackWithTraces));
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Traces")).toBeInTheDocument();
    });
    // TraceList renders root_name as links
    const link = screen.getByText("track/track-1");
    expect(link.closest("a")).toHaveAttribute("href", "/traces/trace-abc");
    expect(screen.getByText("build")).toBeInTheDocument();
  });

  it("hides traces section when no traces", async () => {
    mockFetcher.mockImplementation(() => Promise.resolve(activeTrack));
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Test Feature Track")).toBeInTheDocument();
    });
    expect(screen.queryByText("Traces")).not.toBeInTheDocument();
  });
});
