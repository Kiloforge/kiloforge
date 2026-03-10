import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SkillsInstallDialog } from "./SkillsInstallDialog";

function renderDialog(overrides: Partial<Parameters<typeof SkillsInstallDialog>[0]> = {}) {
  const props = {
    updating: false,
    error: null,
    onInstall: vi.fn(),
    onCancel: vi.fn(),
    ...overrides,
  };
  return { ...render(<SkillsInstallDialog {...props} />), props };
}

describe("SkillsInstallDialog", () => {
  it("renders title and message", () => {
    renderDialog();
    expect(screen.getByText("Skills Required")).toBeInTheDocument();
    expect(screen.getByText(/requires skills that are not yet installed/)).toBeInTheDocument();
  });

  it("renders Install Skills and Cancel buttons", () => {
    renderDialog();
    expect(screen.getByText("Install Skills")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("calls onInstall when Install Skills clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Install Skills"));
    expect(props.onInstall).toHaveBeenCalled();
  });

  it("calls onCancel when Cancel clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Cancel"));
    expect(props.onCancel).toHaveBeenCalled();
  });

  it("calls onCancel when overlay clicked", async () => {
    const user = userEvent.setup();
    const { props, container } = renderDialog();
    await user.click(container.firstElementChild!);
    expect(props.onCancel).toHaveBeenCalled();
  });

  it("does not call onCancel when dialog content clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Skills Required"));
    expect(props.onCancel).not.toHaveBeenCalled();
  });

  it("disables buttons when updating", () => {
    renderDialog({ updating: true });
    expect(screen.getByText("Installing...")).toBeDisabled();
    expect(screen.getByText("Cancel")).toBeDisabled();
  });

  it("shows error when provided", () => {
    renderDialog({ error: "Network failure" });
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });
});
