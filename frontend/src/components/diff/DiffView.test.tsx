import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DiffStats } from "./DiffStats";
import { FileList } from "./FileList";
import { FileDiff } from "./FileDiff";
import type { FileDiff as FileDiffType, DiffStats as DiffStatsType } from "../../types/api";

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

const mockFile: FileDiffType = {
  path: "src/main.go",
  status: "modified",
  insertions: 5,
  deletions: 2,
  is_binary: false,
  hunks: [
    {
      old_start: 1,
      old_lines: 3,
      new_start: 1,
      new_lines: 4,
      header: "@@ -1,3 +1,4 @@",
      lines: [
        { type: "context", content: "package main", old_no: 1, new_no: 1 },
        { type: "add", content: "// new comment", old_no: null, new_no: 2 },
        { type: "context", content: "", old_no: 2, new_no: 3 },
        { type: "delete", content: "old line", old_no: 3, new_no: null },
      ],
    },
  ],
};

describe("DiffStats", () => {
  it("renders file count, insertions, and deletions", () => {
    const stats: DiffStatsType = { files_changed: 3, insertions: 10, deletions: 4 };
    wrap(<DiffStats stats={stats} />);
    expect(screen.getByText("3 files changed")).toBeInTheDocument();
    expect(screen.getByText("+10")).toBeInTheDocument();
    expect(screen.getByText("-4")).toBeInTheDocument();
  });

  it("singular 'file' for 1 file changed", () => {
    const stats: DiffStatsType = { files_changed: 1, insertions: 1, deletions: 0 };
    wrap(<DiffStats stats={stats} />);
    expect(screen.getByText("1 file changed")).toBeInTheDocument();
  });

  it("shows truncated indicator", () => {
    const stats: DiffStatsType = { files_changed: 50, insertions: 100, deletions: 50 };
    wrap(<DiffStats stats={stats} truncated />);
    expect(screen.getByText("(truncated)")).toBeInTheDocument();
  });

  it("hides zero insertions/deletions", () => {
    const stats: DiffStatsType = { files_changed: 1, insertions: 0, deletions: 0 };
    wrap(<DiffStats stats={stats} />);
    expect(screen.queryByText("+0")).not.toBeInTheDocument();
    expect(screen.queryByText("-0")).not.toBeInTheDocument();
  });
});

describe("FileList", () => {
  const files: FileDiffType[] = [
    { ...mockFile, path: "a.go", status: "added", insertions: 10, deletions: 0 },
    { ...mockFile, path: "b.go", status: "modified" },
    { ...mockFile, path: "c.go", status: "deleted", insertions: 0, deletions: 8 },
  ];

  it("renders files with status badges", () => {
    const onSelect = vi.fn();
    wrap(<FileList files={files} activeIndex={0} onSelect={onSelect} />);
    expect(screen.getByText("A")).toBeInTheDocument();
    expect(screen.getByText("M")).toBeInTheDocument();
    expect(screen.getByText("D")).toBeInTheDocument();
    expect(screen.getByText("a.go")).toBeInTheDocument();
    expect(screen.getByText("b.go")).toBeInTheDocument();
    expect(screen.getByText("c.go")).toBeInTheDocument();
  });

  it("calls onSelect when a file is clicked", () => {
    const onSelect = vi.fn();
    wrap(<FileList files={files} activeIndex={0} onSelect={onSelect} />);
    fireEvent.click(screen.getByText("b.go"));
    expect(onSelect).toHaveBeenCalledWith(1);
  });

  it("renders per-file +/- counts", () => {
    const onSelect = vi.fn();
    wrap(<FileList files={files} activeIndex={0} onSelect={onSelect} />);
    expect(screen.getByText("+10")).toBeInTheDocument();
    expect(screen.getByText("-8")).toBeInTheDocument();
  });
});

describe("FileDiff", () => {
  it("renders file header with path and status", () => {
    wrap(<FileDiff file={mockFile} />);
    expect(screen.getByText("src/main.go")).toBeInTheDocument();
    expect(screen.getByText("M")).toBeInTheDocument();
  });

  it("renders hunk header", () => {
    wrap(<FileDiff file={mockFile} />);
    expect(screen.getByText("@@ -1,3 +1,4 @@")).toBeInTheDocument();
  });

  it("renders diff lines with correct prefixes", () => {
    wrap(<FileDiff file={mockFile} />);
    expect(screen.getByText(/\+\/\/ new comment/)).toBeInTheDocument();
    expect(screen.getByText(/-old line/)).toBeInTheDocument();
  });

  it("shows binary label for binary files", () => {
    const binary: FileDiffType = { ...mockFile, is_binary: true, hunks: [] };
    wrap(<FileDiff file={binary} />);
    expect(screen.getByText(/Binary file/)).toBeInTheDocument();
  });

  it("collapses when header is clicked", () => {
    wrap(<FileDiff file={mockFile} />);
    const header = screen.getByText("src/main.go");
    fireEvent.click(header.closest("div")!);
    expect(screen.queryByText("@@ -1,3 +1,4 @@")).not.toBeInTheDocument();
  });

  it("renders renamed file path", () => {
    const renamed: FileDiffType = { ...mockFile, old_path: "old.go", path: "new.go", status: "renamed" };
    wrap(<FileDiff file={renamed} />);
    expect(screen.getByText("old.go → new.go")).toBeInTheDocument();
    expect(screen.getByText("R")).toBeInTheDocument();
  });
});
