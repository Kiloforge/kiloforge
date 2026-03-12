import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AskUserQuestionBubble } from "./AskUserQuestionBubble";
import type { WSMessage } from "../../hooks/useAgentWebSocket";

function makeAskMsg(overrides?: Partial<WSMessage>): WSMessage {
  return {
    type: "tool_use",
    text: "AskUserQuestion",
    toolName: "AskUserQuestion",
    toolInput: {
      question: "Which option do you prefer?",
      options: [
        { label: "Option A", description: "Does thing A" },
        { label: "Option B", description: "Does thing B" },
      ],
    },
    timestamp: new Date(),
    ...overrides,
  };
}

describe("AskUserQuestionBubble", () => {
  it("renders question text and option buttons with label and description", () => {
    render(<AskUserQuestionBubble msg={makeAskMsg()} onSend={vi.fn()} />);

    expect(screen.getByText("Which option do you prefer?")).toBeTruthy();
    expect(screen.getByText("Option A")).toBeTruthy();
    expect(screen.getByText("Does thing A")).toBeTruthy();
    expect(screen.getByText("Option B")).toBeTruthy();
    expect(screen.getByText("Does thing B")).toBeTruthy();
  });

  it("calls onSend with the option label when clicked", async () => {
    const onSend = vi.fn();
    const user = userEvent.setup();

    render(<AskUserQuestionBubble msg={makeAskMsg()} onSend={onSend} />);

    await user.click(screen.getByRole("button", { name: /Option A/i }));

    expect(onSend).toHaveBeenCalledTimes(1);
    expect(onSend).toHaveBeenCalledWith("Option A");
  });

  it("disables buttons and highlights selected option after answering", async () => {
    const onSend = vi.fn();
    const user = userEvent.setup();

    render(<AskUserQuestionBubble msg={makeAskMsg()} onSend={onSend} />);

    await user.click(screen.getByRole("button", { name: /Option A/i }));

    const buttons = screen.getAllByRole("button");
    for (const btn of buttons) {
      expect(btn).toBeDisabled();
    }
  });

  it("renders gracefully with missing options", () => {
    const msg = makeAskMsg({
      toolInput: { question: "No options here" },
    });
    render(<AskUserQuestionBubble msg={msg} onSend={vi.fn()} />);

    expect(screen.getByText("No options here")).toBeTruthy();
  });

  it("renders gracefully with empty options array", () => {
    const msg = makeAskMsg({
      toolInput: { question: "Empty options", options: [] },
    });
    render(<AskUserQuestionBubble msg={msg} onSend={vi.fn()} />);

    expect(screen.getByText("Empty options")).toBeTruthy();
  });
});
