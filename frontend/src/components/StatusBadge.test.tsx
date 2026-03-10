import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders the status text", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("running")).toBeInTheDocument();
  });

  it("renders different statuses", () => {
    const { rerender } = render(<StatusBadge status="idle" />);
    expect(screen.getByText("idle")).toBeInTheDocument();

    rerender(<StatusBadge status="error" />);
    expect(screen.getByText("error")).toBeInTheDocument();
  });

  it("renders as a span element", () => {
    render(<StatusBadge status="completed" />);
    const el = screen.getByText("completed");
    expect(el.tagName).toBe("SPAN");
  });
});
