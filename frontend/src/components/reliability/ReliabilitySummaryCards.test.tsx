import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ReliabilitySummaryCards } from "./ReliabilitySummaryCards";
import type { ReliabilitySummary } from "../../types/api";

describe("ReliabilitySummaryCards", () => {
  it("renders cards for each event type with counts", () => {
    const summary: ReliabilitySummary = {
      buckets: [],
      totals: {
        lock_contention: 5,
        agent_timeout: 3,
        merge_conflict: 1,
      },
    };
    render(<ReliabilitySummaryCards summary={summary} />);
    expect(screen.getByText("Lock Contention")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("Agent Timeout")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("Merge Conflict")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("returns null when summary has all zero counts", () => {
    const summary: ReliabilitySummary = {
      buckets: [],
      totals: { lock_contention: 0 },
    };
    const { container } = render(<ReliabilitySummaryCards summary={summary} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders placeholder cards when summary is null", () => {
    const { container } = render(<ReliabilitySummaryCards summary={null} />);
    // Should render default type cards
    expect(container.querySelectorAll("[class*=card]").length).toBeGreaterThan(0);
  });
});
