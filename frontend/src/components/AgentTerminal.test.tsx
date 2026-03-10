import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AgentTerminal } from "./AgentTerminal";
import type { WSMessage } from "../hooks/useAgentWebSocket";

// jsdom doesn't implement scrollIntoView
Element.prototype.scrollIntoView = vi.fn();

vi.mock("../hooks/useAgentWebSocket", () => ({
  useAgentWebSocket: vi.fn(),
}));

vi.mock("../hooks/useFloatingWindow", () => ({
  useFloatingWindow: () => ({
    x: 100,
    y: 100,
    width: 720,
    height: 500,
    zIndex: 1,
    isDragging: false,
    isResizing: false,
    hoverEdge: null,
    resizeEdge: null,
    setRect: vi.fn(),
    getRect: vi.fn(() => ({ x: 100, y: 100, width: 720, height: 500, zIndex: 1 })),
    bringToFront: vi.fn(),
    reset: vi.fn(),
    setHoverEdge: vi.fn(),
    onDragStart: vi.fn(),
    onDragMove: vi.fn(),
    onDragEnd: vi.fn(),
    onResizeStart: vi.fn(),
    onResizeMove: vi.fn(),
    onResizeEnd: vi.fn(),
  }),
  detectEdge: vi.fn(),
  cursorForEdge: vi.fn(),
}));

vi.mock("./terminal", () => ({
  MessageDispatch: ({ msg }: { msg: { type: string; text: string } }) => (
    <div data-testid="message">{msg.text}</div>
  ),
}));

import { useAgentWebSocket } from "../hooks/useAgentWebSocket";

const mockUseAgentWebSocket = vi.mocked(useAgentWebSocket);

function setup(overrides: Partial<ReturnType<typeof useAgentWebSocket>> = {}, props: Partial<Parameters<typeof AgentTerminal>[0]> = {}) {
  const defaultWs = {
    messages: [],
    sendMessage: vi.fn(),
    clearMessages: vi.fn(),
    status: "connected" as const,
    agentStatus: "running",
    ...overrides,
  };
  mockUseAgentWebSocket.mockReturnValue(defaultWs);

  const defaultProps = {
    agentId: "agent-abc123",
    onClose: vi.fn(),
    ...props,
  };

  return { ...render(<AgentTerminal {...defaultProps} />), props: defaultProps, ws: defaultWs };
}

describe("AgentTerminal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders agent id in header", () => {
    setup();
    expect(screen.getAllByText("agent-ab").length).toBeGreaterThan(0);
  });

  it("renders custom name when provided", () => {
    setup({}, { name: "My Agent" });
    expect(screen.getByText("My Agent")).toBeInTheDocument();
  });

  it("renders role badge when provided", () => {
    setup({}, { role: "developer" });
    expect(screen.getByText("developer")).toBeInTheDocument();
  });

  it("shows Connected status when connected", () => {
    setup({ status: "connected" });
    expect(screen.getByText("Connected")).toBeInTheDocument();
  });

  it("shows Connecting status", () => {
    setup({ status: "connecting" });
    expect(screen.getByText("Connecting...")).toBeInTheDocument();
  });

  it("shows Reconnecting status", () => {
    setup({ status: "reconnecting" });
    expect(screen.getByText("Reconnecting...")).toBeInTheDocument();
  });

  it("shows Disconnected status", () => {
    setup({ status: "disconnected" });
    expect(screen.getByText("Disconnected")).toBeInTheDocument();
  });

  it("shows empty state when connecting with no messages", () => {
    setup({ status: "connecting", messages: [] });
    expect(screen.getByText("Connecting to agent...")).toBeInTheDocument();
  });

  it("shows waiting state when connected with no messages", () => {
    setup({ status: "connected", messages: [] });
    expect(screen.getByText("Waiting for agent output...")).toBeInTheDocument();
  });

  it("renders messages via MessageDispatch", () => {
    setup({
      messages: [
        { type: "output", text: "Hello world", timestamp: new Date() },
      ] as WSMessage[],
    });
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("disables input when disconnected", () => {
    setup({ status: "disconnected" });
    const textarea = screen.getByPlaceholderText("Connecting...");
    expect(textarea).toBeDisabled();
  });

  it("disables input when agent is completed", () => {
    setup({ agentStatus: "completed" });
    const textarea = screen.getByPlaceholderText("Agent has exited");
    expect(textarea).toBeDisabled();
  });

  it("enables input when connected and agent running", () => {
    setup({ status: "connected", agentStatus: "running" });
    const textarea = screen.getByPlaceholderText("Type a message... (Enter to send)");
    expect(textarea).not.toBeDisabled();
  });

  it("sends message on Enter key", async () => {
    const user = userEvent.setup();
    const { ws } = setup({ status: "connected", agentStatus: "running" });
    const textarea = screen.getByPlaceholderText("Type a message... (Enter to send)");
    await user.type(textarea, "hello{Enter}");
    expect(ws.sendMessage).toHaveBeenCalledWith("hello");
  });

  it("does not send empty messages", async () => {
    const user = userEvent.setup();
    const { ws } = setup({ status: "connected", agentStatus: "running" });
    const textarea = screen.getByPlaceholderText("Type a message... (Enter to send)");
    await user.type(textarea, "{Enter}");
    expect(ws.sendMessage).not.toHaveBeenCalled();
  });

  it("disables Send button when input is empty", () => {
    setup({ status: "connected", agentStatus: "running" });
    expect(screen.getByText("Send")).toBeDisabled();
  });

  it("calls onClose when close button clicked", async () => {
    const user = userEvent.setup();
    const { props } = setup();
    await user.click(screen.getByTitle("Close (⌘⇧W)"));
    expect(props.onClose).toHaveBeenCalled();
  });

  it("calls onMinimize when minimize button clicked", async () => {
    const user = userEvent.setup();
    const onMinimize = vi.fn();
    setup({}, { onMinimize });
    await user.click(screen.getByTitle("Minimize (⌘⇧M)"));
    expect(onMinimize).toHaveBeenCalled();
  });

  it("does not show minimize button when onMinimize not provided", () => {
    setup();
    expect(screen.queryByTitle("Minimize (⌘⇧M)")).not.toBeInTheDocument();
  });

  it("hides panel when minimized", () => {
    const { container } = setup({}, { minimized: true });
    const panel = container.firstElementChild as HTMLElement;
    expect(panel.style.display).toBe("none");
  });
});
