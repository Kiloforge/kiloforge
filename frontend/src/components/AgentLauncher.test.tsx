import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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
  it("renders role options and prompt input", () => {
    renderLauncher();
    expect(screen.getByText("New Agent")).toBeInTheDocument();
    expect(screen.getByText("Architect")).toBeInTheDocument();
    expect(screen.getByText("Product Advisor")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("calls onLaunch with selected role and prompt", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher();

    await user.click(screen.getByText("Product Advisor"));
    await user.type(screen.getByRole("textbox"), "Help me design a logo");
    await user.click(screen.getByText("Start"));

    expect(props.onLaunch).toHaveBeenCalledWith("product-advisor", "Help me design a logo");
  });

  it("defaults to architect role", async () => {
    const user = userEvent.setup();
    const { props } = renderLauncher();

    await user.click(screen.getByText("Start"));
    expect(props.onLaunch).toHaveBeenCalledWith("architect", "");
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

  it("changes placeholder text based on selected role", async () => {
    const user = userEvent.setup();
    renderLauncher();

    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(textarea.placeholder).toContain("plan");

    await user.click(screen.getByText("Product Advisor"));
    expect(textarea.placeholder).toContain("product guidance");
  });
});
