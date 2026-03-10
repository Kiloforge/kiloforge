import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SyncPanel } from "./SyncPanel";
import type { SyncStatus } from "../types/api";
import type { SyncConflict } from "../hooks/useOriginSync";

const baseSyncStatus: SyncStatus = {
  status: "synced",
  ahead: 0,
  behind: 0,
  local_branch: "main",
  remote_url: "https://gitea.local/repo.git",
};

function renderPanel(overrides: Partial<Parameters<typeof SyncPanel>[0]> = {}) {
  const props = {
    syncStatus: baseSyncStatus,
    loading: false,
    pushing: false,
    pulling: false,
    error: null,
    conflict: null as SyncConflict | null,
    onPush: vi.fn(),
    onPull: vi.fn(),
    onRefresh: vi.fn(),
    onClearError: vi.fn(),
    onResolveConflict: vi.fn(),
    ...overrides,
  };
  return { ...render(<SyncPanel {...props} />), props };
}

describe("SyncPanel", () => {
  it("shows synced status", () => {
    renderPanel();
    expect(screen.getByText("Synced")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    renderPanel({ loading: true });
    expect(screen.getByText("Loading sync status...")).toBeInTheDocument();
  });

  it("shows no remote message when syncStatus is null", () => {
    renderPanel({ syncStatus: null });
    expect(screen.getByText("No origin remote configured")).toBeInTheDocument();
  });

  it("shows ahead count", () => {
    renderPanel({ syncStatus: { ...baseSyncStatus, status: "ahead", ahead: 3 } });
    expect(screen.getByText("3 ahead")).toBeInTheDocument();
  });

  it("shows behind count", () => {
    renderPanel({ syncStatus: { ...baseSyncStatus, status: "behind", behind: 2 } });
    expect(screen.getByText("2 behind")).toBeInTheDocument();
  });

  it("shows branch name", () => {
    renderPanel();
    expect(screen.getByText(/branch: main/)).toBeInTheDocument();
  });

  it("shows error and dismiss button", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ error: "Push failed" });
    expect(screen.getByText("Push failed")).toBeInTheDocument();
    await user.click(screen.getByText("×"));
    expect(props.onClearError).toHaveBeenCalled();
  });

  it("calls onRefresh when refresh clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    await user.click(screen.getByTitle("Refresh status"));
    expect(props.onRefresh).toHaveBeenCalled();
  });

  it("shows Push to Upstream and Pull buttons", () => {
    renderPanel();
    expect(screen.getByText("Push to Upstream")).toBeInTheDocument();
    expect(screen.getByText("Pull from Upstream")).toBeInTheDocument();
  });

  it("calls onPull when Pull clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    await user.click(screen.getByText("Pull from Upstream"));
    expect(props.onPull).toHaveBeenCalled();
  });

  it("shows push form when Push to Upstream clicked", async () => {
    const user = userEvent.setup();
    renderPanel();
    await user.click(screen.getByText("Push to Upstream"));
    expect(screen.getByDisplayValue("kf/main")).toBeInTheDocument();
    expect(screen.getByText("Push")).toBeInTheDocument();
  });

  it("calls onPush with branch when push form submitted", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    await user.click(screen.getByText("Push to Upstream"));
    await user.click(screen.getByText("Push"));
    expect(props.onPush).toHaveBeenCalledWith("kf/main");
  });

  it("hides actions when syncStatus is null", () => {
    renderPanel({ syncStatus: null });
    expect(screen.queryByText("Push to Upstream")).not.toBeInTheDocument();
  });

  it("shows conflict banner with Resolve button for push conflict", () => {
    renderPanel({ conflict: { active: true, direction: "push" } });
    expect(screen.getByText(/push conflict/i)).toBeInTheDocument();
    expect(screen.getByText("Resolve Conflicts")).toBeInTheDocument();
  });

  it("shows conflict banner with Resolve button for pull conflict", () => {
    renderPanel({ conflict: { active: true, direction: "pull" } });
    expect(screen.getByText(/pull conflict/i)).toBeInTheDocument();
    expect(screen.getByText("Resolve Conflicts")).toBeInTheDocument();
  });

  it("calls onResolveConflict when Resolve Conflicts clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ conflict: { active: true, direction: "push" } });
    await user.click(screen.getByText("Resolve Conflicts"));
    expect(props.onResolveConflict).toHaveBeenCalled();
  });

  it("does not show generic error when conflict is active", () => {
    renderPanel({ conflict: { active: true, direction: "push" }, error: null });
    expect(screen.queryByText("Push failed")).not.toBeInTheDocument();
  });
});
