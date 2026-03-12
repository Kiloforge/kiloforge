import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SetupRequiredDialog } from "./SetupRequiredDialog";

let lastTerminalProps: Record<string, unknown> = {};

vi.mock("./AgentTerminal", () => ({
  AgentTerminal: (props: Record<string, unknown>) => {
    lastTerminalProps = props;
    const name = props.name as string | undefined;
    const role = props.role as string | undefined;
    return (
      <div data-testid="agent-terminal" data-agent-id={props.agentId as string}>
        <button onClick={props.onClose as () => void}>close-terminal</button>
        {typeof props.onMinimize === "function" && (
          <button onClick={props.onMinimize as () => void}>minimize-terminal</button>
        )}
        {name && <span data-testid="terminal-name">{name}</span>}
        {role && <span data-testid="terminal-role">{role}</span>}
      </div>
    );
  },
}));

vi.mock("./MiniCard", () => ({
  MiniCard: (props: Record<string, unknown>) => (
    <div data-testid={`mini-card-${props.agentId}`}>
      <button onClick={props.onRestore as () => void}>restore</button>
      <button onClick={props.onClose as () => void}>close-minicard</button>
    </div>
  ),
}));

function renderDialog(overrides: Partial<Parameters<typeof SetupRequiredDialog>[0]> = {}) {
  lastTerminalProps = {};
  const props = {
    projectSlug: "my-project",
    agentId: null as string | null,
    starting: false,
    error: null as string | null,
    onRunSetup: vi.fn(),
    onSetupComplete: vi.fn(),
    onCancel: vi.fn(),
    ...overrides,
  };
  return { ...render(<SetupRequiredDialog {...props} />), props };
}

describe("SetupRequiredDialog", () => {
  it("renders setup dialog when no agentId", () => {
    renderDialog();
    expect(screen.getByText("Kiloforge Setup Required")).toBeInTheDocument();
    expect(screen.getByText(/my-project/)).toBeInTheDocument();
  });

  it("renders Run Setup and Cancel buttons", () => {
    renderDialog();
    expect(screen.getByText("Run Setup")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("calls onRunSetup when Run Setup clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Run Setup"));
    expect(props.onRunSetup).toHaveBeenCalled();
  });

  it("calls onCancel when Cancel clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Cancel"));
    expect(props.onCancel).toHaveBeenCalled();
  });

  it("disables buttons when starting", () => {
    renderDialog({ starting: true });
    expect(screen.getByText("Starting...")).toBeDisabled();
    expect(screen.getByText("Cancel")).toBeDisabled();
  });

  it("shows error when provided", () => {
    renderDialog({ error: "Setup failed" });
    expect(screen.getByText("Setup failed")).toBeInTheDocument();
  });

  it("renders AgentTerminal when agentId is provided", () => {
    renderDialog({ agentId: "agent-123" });
    expect(screen.getByTestId("agent-terminal")).toBeInTheDocument();
    expect(screen.getByTestId("agent-terminal")).toHaveAttribute("data-agent-id", "agent-123");
  });

  it("calls onSetupComplete when terminal is closed", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog({ agentId: "agent-123" });
    await user.click(screen.getByText("close-terminal"));
    expect(props.onSetupComplete).toHaveBeenCalled();
  });

  it("provides onMinimize callback to AgentTerminal", () => {
    renderDialog({ agentId: "agent-123" });
    expect(lastTerminalProps.onMinimize).toBeDefined();
  });

  it("shows MiniCard when terminal is minimized", async () => {
    const user = userEvent.setup();
    renderDialog({ agentId: "agent-123" });
    await user.click(screen.getByText("minimize-terminal"));
    expect(screen.getByTestId("mini-card-agent-123")).toBeInTheDocument();
  });

  it("restores terminal when MiniCard restore is clicked", async () => {
    const user = userEvent.setup();
    renderDialog({ agentId: "agent-123" });
    await user.click(screen.getByText("minimize-terminal"));
    await user.click(screen.getByText("restore"));
    expect(screen.getByTestId("agent-terminal")).toBeInTheDocument();
    expect(screen.queryByTestId("mini-card-agent-123")).not.toBeInTheDocument();
  });

  it("passes agent name to AgentTerminal when provided", () => {
    renderDialog({ agentId: "agent-123", agentName: "curiously brave luna" });
    expect(screen.getByTestId("terminal-name")).toHaveTextContent("curiously brave luna");
  });

  it("passes agent role to AgentTerminal when provided", () => {
    renderDialog({ agentId: "agent-123", agentRole: "setup" });
    expect(screen.getByTestId("terminal-role")).toHaveTextContent("setup");
  });
});
