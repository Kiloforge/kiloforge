import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MessageErrorBoundary } from "./MessageErrorBoundary";

function ThrowingComponent(): never {
  throw new Error("test render error");
}

function SafeComponent() {
  return <div>safe content</div>;
}

describe("MessageErrorBoundary", () => {
  it("renders children when no error", () => {
    render(
      <MessageErrorBoundary>
        <SafeComponent />
      </MessageErrorBoundary>
    );
    expect(screen.getByText("safe content")).toBeTruthy();
  });

  it("catches render errors and shows fallback", () => {
    // Suppress React error boundary console.error
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    render(
      <MessageErrorBoundary>
        <ThrowingComponent />
      </MessageErrorBoundary>
    );

    expect(screen.getByText(/Failed to render message/)).toBeTruthy();
    expect(screen.getByText(/test render error/)).toBeTruthy();

    spy.mockRestore();
    warnSpy.mockRestore();
  });
});
