import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AgentDiffPanel } from "./AgentDiffPanel";
import type { DiffResponse } from "../../types/api";

vi.mock("../../hooks/useDiff", () => ({
  useProjectDiff: vi.fn(),
}));

import { useProjectDiff } from "../../hooks/useDiff";

const mockUseProjectDiff = vi.mocked(useProjectDiff);

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

const mockDiff: DiffResponse = {
  branch: "feature/test",
  base: "main",
  stats: { files_changed: 2, insertions: 10, deletions: 3 },
  files: [
    {
      path: "src/main.go",
      status: "modified",
      insertions: 5,
      deletions: 2,
      is_binary: false,
      hunks: [
        {
          old_start: 1, old_lines: 3, new_start: 1, new_lines: 4,
          header: "@@ -1,3 +1,4 @@",
          lines: [
            { type: "context", content: "package main", old_no: 1, new_no: 1 },
            { type: "add", content: "// new comment", old_no: null, new_no: 2 },
          ],
        },
      ],
    },
    {
      path: "src/util.go",
      status: "added",
      insertions: 5,
      deletions: 1,
      is_binary: false,
      hunks: [
        {
          old_start: 0, old_lines: 0, new_start: 1, new_lines: 5,
          header: "@@ -0,0 +1,5 @@",
          lines: [
            { type: "add", content: "package util", old_no: null, new_no: 1 },
          ],
        },
      ],
    },
  ],
};

describe("AgentDiffPanel", () => {
  it("renders loading state", () => {
    mockUseProjectDiff.mockReturnValue({ data: undefined, isLoading: true, error: null } as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    expect(screen.getByText("Loading diff...")).toBeInTheDocument();
  });

  it("renders error state", () => {
    mockUseProjectDiff.mockReturnValue({ data: undefined, isLoading: false, error: new Error("network fail") } as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    expect(screen.getByText(/Failed to load diff/)).toBeInTheDocument();
  });

  it("renders empty state when no files changed", () => {
    mockUseProjectDiff.mockReturnValue({
      data: { ...mockDiff, files: [], stats: { files_changed: 0, insertions: 0, deletions: 0 } },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    expect(screen.getByText("No changes on this branch")).toBeInTheDocument();
  });

  it("renders file tree with all changed files", () => {
    mockUseProjectDiff.mockReturnValue({ data: mockDiff, isLoading: false, error: null } as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    // Both files appear in sidebar; selected file also appears in diff header
    expect(screen.getAllByText("src/main.go").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("src/util.go").length).toBeGreaterThanOrEqual(1);
  });

  it("shows only selected file diff (first file by default)", () => {
    mockUseProjectDiff.mockReturnValue({ data: mockDiff, isLoading: false, error: null } as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    // First file's hunk header should be visible
    expect(screen.getByText("@@ -1,3 +1,4 @@")).toBeInTheDocument();
    // Second file's hunk header should NOT be visible (single-file display)
    expect(screen.queryByText("@@ -0,0 +1,5 @@")).not.toBeInTheDocument();
  });

  it("switches to selected file on click", () => {
    mockUseProjectDiff.mockReturnValue({ data: mockDiff, isLoading: false, error: null } as ReturnType<typeof useProjectDiff>);
    wrap(<AgentDiffPanel slug="my-proj" branch="feature/test" />);
    // Click on second file in sidebar
    fireEvent.click(screen.getByText("src/util.go"));
    // Second file's content should now be visible
    expect(screen.getByText("@@ -0,0 +1,5 @@")).toBeInTheDocument();
    // First file's content should be hidden
    expect(screen.queryByText("@@ -1,3 +1,4 @@")).not.toBeInTheDocument();
  });
});
