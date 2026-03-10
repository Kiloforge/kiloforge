import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { GettingStartedChecklist } from "./GettingStartedChecklist";
import type { Project, Agent } from "../types/api";

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

const STORAGE_KEY = "kf_getting_started_dismissed";

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: "a1", role: "developer", ref: "", status: "running",
    session_id: "s", pid: 1, worktree_dir: "", log_file: "",
    started_at: "", updated_at: "", estimated_cost_usd: 0,
    ...overrides,
  };
}

describe("GettingStartedChecklist", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders checklist when fewer than 2 projects", () => {
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[]} tracks={[]} />,
    );
    expect(screen.getByText("Getting Started")).toBeInTheDocument();
  });

  it("hides checklist when 2 or more projects", () => {
    const projects: Project[] = [
      { slug: "a", repo_name: "a", origin_remote: "", active: true },
      { slug: "b", repo_name: "b", origin_remote: "", active: true },
    ];
    renderWithQuery(
      <GettingStartedChecklist projects={projects} agents={[]} tracks={[]} />,
    );
    expect(screen.queryByText("Getting Started")).not.toBeInTheDocument();
  });

  it("shows kiloforge running as completed", () => {
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[]} tracks={[]} />,
    );
    expect(screen.getByText("Kiloforge is running")).toBeInTheDocument();
  });

  it("marks 'Add a project' complete when projects exist", () => {
    const projects: Project[] = [
      { slug: "test", repo_name: "test", origin_remote: "", active: true },
    ];
    renderWithQuery(
      <GettingStartedChecklist projects={projects} agents={[]} tracks={[]} />,
    );
    const item = screen.getByText("Add a project");
    expect(item.closest("[data-done='true']")).toBeInTheDocument();
  });

  it("marks 'Add a project' incomplete when no projects", () => {
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[]} tracks={[]} />,
    );
    const item = screen.getByText("Add a project");
    expect(item.closest("[data-done='false']")).toBeInTheDocument();
  });

  it("marks 'Generate tracks' complete when tracks exist", () => {
    renderWithQuery(
      <GettingStartedChecklist
        projects={[]}
        agents={[]}
        tracks={[{ id: "t1", title: "Track 1", status: "pending", project: "p", deps_count: 0, conflict_count: 0 }]}
      />,
    );
    const item = screen.getByText("Generate tracks");
    expect(item.closest("[data-done='true']")).toBeInTheDocument();
  });

  it("marks 'Spawn your first agent' complete when agents exist", () => {
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[makeAgent()]} tracks={[]} />,
    );
    const item = screen.getByText("Spawn your first agent");
    expect(item.closest("[data-done='true']")).toBeInTheDocument();
  });

  it("hides on dismiss and persists to localStorage", () => {
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[]} tracks={[]} />,
    );
    expect(screen.getByText("Getting Started")).toBeInTheDocument();
    fireEvent.click(screen.getByTitle("Dismiss checklist"));
    expect(screen.queryByText("Getting Started")).not.toBeInTheDocument();
    expect(localStorage.getItem(STORAGE_KEY)).toBe("1");
  });

  it("does not render when already dismissed in localStorage", () => {
    localStorage.setItem(STORAGE_KEY, "1");
    renderWithQuery(
      <GettingStartedChecklist projects={[]} agents={[]} tracks={[]} />,
    );
    expect(screen.queryByText("Getting Started")).not.toBeInTheDocument();
  });

  it("auto-dismisses when all items complete", () => {
    const projects: Project[] = [
      { slug: "test", repo_name: "test", origin_remote: "", active: true },
    ];
    const tracks = [{ id: "t1", title: "T", status: "pending", project: "p", deps_count: 0, conflict_count: 0 }];
    renderWithQuery(
      <GettingStartedChecklist projects={projects} agents={[makeAgent()]} tracks={tracks} />,
    );
    expect(screen.queryByText("Getting Started")).not.toBeInTheDocument();
  });
});
