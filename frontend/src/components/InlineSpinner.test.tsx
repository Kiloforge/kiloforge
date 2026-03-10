import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { InlineSpinner } from "./InlineSpinner";

describe("InlineSpinner", () => {
  it("renders a spinner element", () => {
    render(<InlineSpinner />);
    const spinner = screen.getByRole("status");
    expect(spinner).toBeInTheDocument();
  });

  it("renders with default sr-only label", () => {
    render(<InlineSpinner />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders with custom label", () => {
    render(<InlineSpinner label="Loading agent..." />);
    expect(screen.getByText("Loading agent...")).toBeInTheDocument();
  });
});
