import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AgentLauncher } from "./AgentLauncher";

function renderLauncher(props?: Partial<Parameters<typeof AgentLauncher>[0]>) {
  const defaultProps = {
    onLaunch: vi.fn(),
    onClose: vi.fn(),
    launching: false,
    ...props,
  };
  return { ...render(<AgentLauncher {...defaultProps} />), props: defaultProps };
}

describe("AgentLauncher", () => {
  it("renders prompt input without role selector on dashboard", () => {
    renderLauncher();
    expect(screen.getByText("New Agent")).toBeInTheDocument();
    // Single role (interactive) — role selector is hidden
    expect(screen.queryByText("Interactive")).not.toBeInTheDocument();
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("hides advisor roles and role selector when no projectSlug", () => {
    renderLauncher();
    expect(screen.queryByText("Product Advisor")).not.toBeInTheDocument();
    expect(screen.queryByText("Reliability Advisor")).not.toBeInTheDocument();
  });

  it("shows interactive and advisor roles when projectSlug is provided", () => {
    renderLauncher({ projectSlug: "my-project" });
    expect(screen.getByText("Interactive")).toBeInTheDocument();
    expect(screen.getByText("Product Advisor")).toBeInTheDocument();
    expect(screen.getByText("Reliability Advisor")).toBeInTheDocument();
  });

  it("calls onLaunch with selected role and prompt", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher({ projectSlug: "my-project" });

    await user.click(screen.getByText("Product Advisor"));
    await user.type(screen.getByRole("textbox"), "Help me design a logo");
    await user.click(screen.getByText("Start"));

    expect(props.onLaunch).toHaveBeenCalledWith("advisor-product", "Help me design a logo");
  });

  it("defaults to interactive role", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher();

    await user.click(screen.getByText("Start"));
    expect(props.onLaunch).toHaveBeenCalledWith("interactive", "");
  });

  it("calls onClose when Cancel is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher();

    await user.click(screen.getByText("Cancel"));
    expect(props.onClose).toHaveBeenCalled();
  });

  it("calls onClose when overlay is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher();

    // Click the overlay (the outermost div)
    const overlay = screen.getByText("New Agent").closest("[class*=overlay]")!;
    await user.click(overlay);
    expect(props.onClose).toHaveBeenCalled();
  });

  it("disables buttons when launching", () => {
    renderLauncher({ launching: true });
    expect(screen.getByText("Starting...")).toBeDisabled();
    expect(screen.getByText("Cancel")).toBeDisabled();
  });

  it("does not submit when isComposing (IME)", () => {
    const { props } = renderLauncher();
    const textarea = screen.getByRole("textbox");
    fireEvent.keyDown(textarea, { key: "Enter", metaKey: true, isComposing: true });
    expect(props.onLaunch).not.toHaveBeenCalled();
  });

  it("changes placeholder text based on selected role", async () => {
    const user = userEvent.setup();
    renderLauncher({ projectSlug: "my-project" });

    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(textarea.placeholder).toContain("project");

    await user.click(screen.getByText("Product Advisor"));
    expect(textarea.placeholder).toContain("product guidance");
  });

  describe("waiting for capacity", () => {
    it("shows waiting overlay when waitingForCapacity is true", () => {
      renderLauncher({
        waitingForCapacity: true,
        waitingCapacity: { max: 3, active: 3, available: 0 },
        onCancelWaiting: vi.fn(),
      });
      expect(screen.getByText("Kiloforge at max capacity")).toBeInTheDocument();
      expect(screen.getByText(/3\/3 agents active/)).toBeInTheDocument();
      expect(screen.getByText(/increase Max Swarm Size/)).toBeInTheDocument();
      expect(screen.getByText(/Will auto-retry/)).toBeInTheDocument();
    });

    it("shows cancel button in waiting state", () => {
      renderLauncher({
        waitingForCapacity: true,
        waitingCapacity: { max: 3, active: 3, available: 0 },
        onCancelWaiting: vi.fn(),
      });
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });

    it("calls onCancelWaiting when cancel is clicked in waiting state", async () => {
      const user = userEvent.setup();
      const onCancelWaiting = vi.fn();
      renderLauncher({
        waitingForCapacity: true,
        waitingCapacity: { max: 3, active: 3, available: 0 },
        onCancelWaiting,
      });
      await user.click(screen.getByText("Cancel"));
      expect(onCancelWaiting).toHaveBeenCalled();
    });

    it("handles null capacity gracefully", () => {
      renderLauncher({
        waitingForCapacity: true,
        waitingCapacity: null,
        onCancelWaiting: vi.fn(),
      });
      expect(screen.getByText("Kiloforge at max capacity")).toBeInTheDocument();
      expect(screen.getByText(/0\/0 agents active/)).toBeInTheDocument();
    });
  });
});
