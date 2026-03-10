import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ProjectMetadataView } from "./ProjectMetadataView";
import type { ProjectMetadata } from "../types/api";

function makeMetadata(
  overrides: Partial<ProjectMetadata["track_summary"]> = {},
): ProjectMetadata {
  return {
    product: "# Test Product",
    tech_stack: "# Tech Stack",
    quick_links: [],
    track_summary: {
      total: 20,
      pending: 1,
      in_progress: 2,
      completed: 5,
      archived: 12,
      ...overrides,
    },
  };
}

describe("ProjectMetadataView", () => {
  it("shows combined done count (completed + archived)", () => {
    render(<ProjectMetadataView metadata={makeMetadata()} />);
    // done = 5 + 12 = 17, total = 20
    expect(screen.getByText("17/20")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
  });

  it("shows in-progress and pending as secondary stats", () => {
    render(<ProjectMetadataView metadata={makeMetadata()} />);
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Pending")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("does not show separate Completed or Archived counters", () => {
    render(<ProjectMetadataView metadata={makeMetadata()} />);
    expect(screen.queryByText("Completed")).not.toBeInTheDocument();
    expect(screen.queryByText("Archived")).not.toBeInTheDocument();
  });

  it("shows 0 done when nothing completed or archived", () => {
    render(
      <ProjectMetadataView
        metadata={makeMetadata({
          total: 5,
          pending: 3,
          in_progress: 2,
          completed: 0,
          archived: 0,
        })}
      />,
    );
    expect(screen.getByText("0/5")).toBeInTheDocument();
  });
});
