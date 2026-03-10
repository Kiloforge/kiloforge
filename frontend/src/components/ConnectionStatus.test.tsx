import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConnectionStatus } from "./ConnectionStatus";

describe("ConnectionStatus", () => {
  it("renders connected state", () => {
    render(<ConnectionStatus state="connected" />);
    const badge = screen.getByTestId("sse-status");
    expect(badge).toHaveTextContent("connected");
    expect(badge).toHaveAttribute("data-status", "connected");
  });

  it("renders disconnected state", () => {
    render(<ConnectionStatus state="disconnected" />);
    const badge = screen.getByTestId("sse-status");
    expect(badge).toHaveTextContent("disconnected");
    expect(badge).toHaveAttribute("data-status", "disconnected");
  });

  it("renders reconnecting state", () => {
    render(<ConnectionStatus state="reconnecting" />);
    const badge = screen.getByTestId("sse-status");
    expect(badge).toHaveTextContent("reconnecting");
    expect(badge).toHaveAttribute("data-status", "reconnecting");
  });

  it("has sse-status test id", () => {
    render(<ConnectionStatus state="connected" />);
    expect(screen.getByTestId("sse-status")).toBeInTheDocument();
  });
});
