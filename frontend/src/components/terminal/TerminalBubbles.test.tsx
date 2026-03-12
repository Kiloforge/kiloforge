import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
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

  it("renders tool_use with human-readable summary", () => {
    render(
      <MessageDispatch
        msg={makeMsg({
          type: "tool_use",
          text: "Bash",
          toolName: "Bash",
          toolInput: { command: "ls -la", description: "List files" },
        })}
      />
    );
    expect(screen.getByText("Bash")).toBeTruthy();
    expect(screen.getByText("List files")).toBeTruthy();
  });

  it("renders tool_use with file path summary for Read", () => {
    render(
      <MessageDispatch
        msg={makeMsg({
          type: "tool_use",
          text: "Read",
          toolName: "Read",
          toolInput: { file_path: "/src/main.ts" },
        })}
      />
    );
    expect(screen.getByText("Read")).toBeTruthy();
    expect(screen.getByText("/src/main.ts")).toBeTruthy();
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

  it("renders AskUserQuestion tool_use as question bubble when onSend provided", () => {
    const onSend = vi.fn();
    render(
      <MessageDispatch
        msg={makeMsg({
          type: "tool_use",
          text: "AskUserQuestion",
          toolName: "AskUserQuestion",
          toolInput: {
            question: "Pick one",
            options: [{ label: "A", description: "Desc A" }],
          },
        })}
        onSend={onSend}
      />
    );
    expect(screen.getByText("Pick one")).toBeTruthy();
    expect(screen.getByText("A")).toBeTruthy();
    expect(screen.getByText("Desc A")).toBeTruthy();
  });

  it("falls back to ToolUseBubble for AskUserQuestion when onSend is not provided", () => {
    render(
      <MessageDispatch
        msg={makeMsg({
          type: "tool_use",
          text: "AskUserQuestion",
          toolName: "AskUserQuestion",
          toolInput: {
            question: "Pick one",
            options: [{ label: "A", description: "Desc A" }],
          },
        })}
      />
    );
    // Should render as generic tool_use bubble (shows tool name in toolName span)
    const toolNames = screen.getAllByText("AskUserQuestion");
    expect(toolNames.length).toBeGreaterThanOrEqual(1);
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

  it("hides system init messages", () => {
    const { container } = render(
      <MessageDispatch msg={makeMsg({ type: "system", subtype: "init" })} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("hides system debug messages", () => {
    const { container } = render(
      <MessageDispatch msg={makeMsg({ type: "system", subtype: "debug" })} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("still renders system error messages", () => {
    render(
      <MessageDispatch msg={makeMsg({ type: "system", subtype: "error" })} />
    );
    expect(screen.getByText("error")).toBeTruthy();
  });

  it("still renders system warning messages", () => {
    render(
      <MessageDispatch msg={makeMsg({ type: "system", subtype: "warning" })} />
    );
    expect(screen.getByText("warning")).toBeTruthy();
  });
});
