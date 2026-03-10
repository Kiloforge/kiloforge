import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SetupRequiredDialog } from "./SetupRequiredDialog";

vi.mock("./AgentTerminal", () => ({
  AgentTerminal: ({ agentId, onClose }: { agentId: string; onClose: () => void }) => (
    <div data-testid="agent-terminal" data-agent-id={agentId}>
      <button onClick={onClose}>close-terminal</button>
    </div>
  ),
}));

function renderDialog(overrides: Partial<Parameters<typeof SetupRequiredDialog>[0]> = {}) {
  const props = {
    projectSlug: "my-project",
    agentId: null,
    starting: false,
    error: null,
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
});
