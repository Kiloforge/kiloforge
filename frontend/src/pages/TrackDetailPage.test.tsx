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
});
