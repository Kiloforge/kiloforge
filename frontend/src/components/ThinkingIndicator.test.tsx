import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ThinkingIndicator } from "./ThinkingIndicator";

describe("ThinkingIndicator", () => {
  it("renders with default 'Thinking' label", () => {
    render(<ThinkingIndicator />);
    expect(screen.getByRole("status")).toBeTruthy();
    expect(screen.getByText("Thinking")).toBeTruthy();
  });

  it("renders with custom label", () => {
    render(<ThinkingIndicator label="Initializing" />);
    expect(screen.getByText("Initializing")).toBeTruthy();
  });

  it("has aria-live polite for accessibility", () => {
    render(<ThinkingIndicator />);
    const el = screen.getByRole("status");
    expect(el.getAttribute("aria-live")).toBe("polite");
  });

  it("renders animated dots", () => {
    const { container } = render(<ThinkingIndicator />);
    const dots = container.querySelectorAll("span[class*='dot']");
    expect(dots.length).toBe(3);
  });
});
