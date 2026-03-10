import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MessageDispatch } from "./TerminalBubbles";
import type { WSMessage } from "../../hooks/useAgentWebSocket";

function makeMsg(overrides: Partial<WSMessage>): WSMessage {
  return {
    type: "text",
    text: "",
    timestamp: new Date(),
    ...overrides,
  };
}

describe("MessageDispatch", () => {
  it("renders text message", () => {
    render(<MessageDispatch msg={makeMsg({ type: "text", text: "Hello" })} />);
    expect(screen.getByText("Hello")).toBeTruthy();
  });

  it("renders input message", () => {
    render(<MessageDispatch msg={makeMsg({ type: "input", text: "user input" })} />);
    expect(screen.getByText("user input")).toBeTruthy();
  });

  it("renders thinking message", () => {
    render(<MessageDispatch msg={makeMsg({ type: "thinking", thinking: "pondering" })} />);
    expect(screen.getByText("pondering")).toBeTruthy();
  });

  it("renders tool_use message", () => {
    render(
      <MessageDispatch
        msg={makeMsg({ type: "tool_use", text: "Read", toolName: "Read", toolId: "t1" })}
      />
    );
    expect(screen.getByText("Read")).toBeTruthy();
  });

  it("renders status message", () => {
    render(<MessageDispatch msg={makeMsg({ type: "status", text: "Agent exited (code 0)" })} />);
    expect(screen.getByText("Agent exited (code 0)")).toBeTruthy();
  });

  it("renders error message", () => {
    render(<MessageDispatch msg={makeMsg({ type: "error", text: "something broke" })} />);
    expect(screen.getByText("something broke")).toBeTruthy();
  });

  it("renders turn separator", () => {
    render(<MessageDispatch msg={makeMsg({ type: "turn_start" })} turnNumber={3} />);
    expect(screen.getByText("Turn 3")).toBeTruthy();
  });

  it("returns null for unknown message type", () => {
    const { container } = render(
      <MessageDispatch msg={makeMsg({ type: "unknown" as WSMessage["type"] })} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("handles non-string text field gracefully", () => {
    // Simulate runtime type mismatch
    const msg = makeMsg({ type: "text", text: 123 as unknown as string });
    render(<MessageDispatch msg={msg} />);
    expect(screen.getByText("123")).toBeTruthy();
  });

  it("handles non-string thinking field gracefully", () => {
    const msg = makeMsg({ type: "thinking", thinking: undefined });
    const { container } = render(<MessageDispatch msg={msg} />);
    expect(container.querySelector(".thinkingMessage")).toBeTruthy();
  });
});
