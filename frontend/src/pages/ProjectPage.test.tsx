import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ProjectPage } from "./ProjectPage";
import type { Project, Track, BoardState } from "../types/api";

const mockProject: Project = {
  slug: "test-proj",
  repo_name: "test-proj",
  origin_remote: "https://github.com/user/test-proj.git",
  active: true,
};

const mockTracks: Track[] = [
  { id: "track-1", title: "Track One", status: "complete", project: "test-proj" },
  { id: "track-2", title: "Track Two", status: "in-progress", project: "test-proj" },
];

const mockBoard: BoardState = {
  columns: ["backlog", "approved", "in_progress", "in_review", "done"],
  cards: {
    "track-1": {
      track_id: "track-1",
      title: "Track One",
      type: "feature",
      column: "done",
      position: 0,
      moved_at: "2026-03-10T00:00:00Z",
      created_at: "2026-03-10T00:00:00Z",
    },
    "track-2": {
      track_id: "track-2",
      title: "Track Two",
      type: "chore",
      column: "in_progress",
      position: 0,
      moved_at: "2026-03-10T00:00:00Z",
      created_at: "2026-03-10T00:00:00Z",
    },
  },
};

vi.mock("../hooks/useTracks", () => ({
  useTracks: () => ({
    tracks: mockTracks,
    loading: false,
    handleTrackUpdate: vi.fn(),
    handleTrackRemoved: vi.fn(),
  }),
}));

vi.mock("../hooks/useProjects", () => ({
  useProjects: () => ({
    projects: [mockProject],
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
  useSSHKeys: () => ({ keys: [], loading: false, fetchKeys: vi.fn() }),
}));

vi.mock("../hooks/useBoard", () => ({
  useBoard: () => ({
    board: mockBoard,
    loading: false,
    moveCard: vi.fn(),
    syncBoard: vi.fn(),
    syncing: false,
  }),
}));

vi.mock("../hooks/useOriginSync", () => ({
  useOriginSync: () => ({
    syncStatus: { ahead: 0, behind: 0, status: "synced", local_branch: "main" },
    loading: false,
    pushing: false,
    pulling: false,
    error: null,
    push: vi.fn(),
    pull: vi.fn(),
    refresh: vi.fn(),
    clearError: vi.fn(),
  }),
}));

vi.mock("../hooks/useConsent", () => ({
  useConsent: () => ({
    showDialog: false,
    requestConsent: vi.fn(),
    accept: vi.fn(),
    deny: vi.fn(),
  }),
}));

vi.mock("../hooks/useSkillsPrompt", () => ({
  useSkillsPrompt: () => ({
    showDialog: false,
    updating: false,
    error: null,
    requestInstall: vi.fn(),
    install: vi.fn(),
    cancel: vi.fn(),
  }),
}));

vi.mock("../hooks/useSetupPrompt", () => ({
  useSetupPrompt: () => ({
    showDialog: false,
    projectSlug: null,
    agentId: null,
    starting: false,
    error: null,
    requestSetup: vi.fn(),
    startSetup: vi.fn(),
    handleSetupComplete: vi.fn(),
    cancel: vi.fn(),
  }),
}));

vi.mock("../components/tour/TourProvider", () => ({
  useTourContextSafe: () => null,
}));

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn().mockImplementation((url: string) => {
    if (url.includes("setup-status")) return Promise.resolve({ setup_complete: true, project_slug: "test-proj" });
    if (url.includes("preflight")) return Promise.resolve({ claude_authenticated: true, skills_ok: true, consent_given: true, setup_required: false });
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

function renderPage(slug = "test-proj") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/projects/${slug}`]}>
        <Routes>
          <Route path="/projects/:slug" element={<ProjectPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ProjectPage", () => {
  it("renders breadcrumb with project slug", () => {
    renderPage();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    // Slug appears in breadcrumb, metadata, and other places
    expect(screen.getAllByText("test-proj").length).toBeGreaterThanOrEqual(1);
  });

  it("renders project metadata section", () => {
    renderPage();
    expect(screen.getByText("Project")).toBeInTheDocument();
    expect(screen.getByText("Slug")).toBeInTheDocument();
    expect(screen.getByText("Repo")).toBeInTheDocument();
    expect(screen.getByText("Remote")).toBeInTheDocument();
  });

  it("renders board section with columns", () => {
    renderPage();
    expect(screen.getByText("Board")).toBeInTheDocument();
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
  });

  it("renders board cards", () => {
    renderPage();
    // Track titles appear in both board cards and track list
    expect(screen.getAllByText("Track One").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Track Two").length).toBeGreaterThanOrEqual(1);
  });

  it("renders sync section", () => {
    renderPage();
    expect(screen.getByText("Origin Sync")).toBeInTheDocument();
  });

  it("renders generate tracks button", () => {
    renderPage();
    expect(screen.getByText("Generate Tracks")).toBeInTheDocument();
  });

  it("shows prompt form when generate tracks is clicked", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(screen.getByText("Generate Tracks"));
    expect(screen.getByPlaceholderText(/Describe the features/)).toBeInTheDocument();
    expect(screen.getByText("Generate")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("renders tracks section", () => {
    renderPage();
    expect(screen.getAllByText("Tracks").length).toBeGreaterThanOrEqual(1);
  });

  it("renders admin operations section", () => {
    renderPage();
    expect(screen.getByText("Admin Operations")).toBeInTheDocument();
  });
});
