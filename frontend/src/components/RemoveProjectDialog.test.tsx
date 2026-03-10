import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RemoveProjectDialog } from "./RemoveProjectDialog";

function renderDialog(overrides: Partial<Parameters<typeof RemoveProjectDialog>[0]> = {}) {
  const props = {
    slug: "test-project",
    removing: false,
    onConfirm: vi.fn().mockResolvedValue(true),
    onCancel: vi.fn(),
    ...overrides,
  };
  return { ...render(<RemoveProjectDialog {...props} />), props };
}

describe("RemoveProjectDialog", () => {
  it("renders title and project name", () => {
    renderDialog();
    expect(screen.getByText("Remove Project")).toBeInTheDocument();
    expect(screen.getByText(/test-project/)).toBeInTheDocument();
  });

  it("renders Remove and Cancel buttons", () => {
    renderDialog();
    expect(screen.getByText("Remove")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("calls onConfirm with slug and cleanup=false when Remove clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Remove"));
    expect(props.onConfirm).toHaveBeenCalledWith("test-project", false);
  });

  it("calls onCancel when Cancel clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Cancel"));
    expect(props.onCancel).toHaveBeenCalled();
  });

  it("passes cleanup=true when checkbox checked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByRole("checkbox"));
    await user.click(screen.getByText("Remove"));
    expect(props.onConfirm).toHaveBeenCalledWith("test-project", true);
  });

  it("shows warning when cleanup checked", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("checkbox"));
    expect(screen.getByText(/permanently delete/)).toBeInTheDocument();
  });

  it("disables buttons when removing", () => {
    renderDialog({ removing: true });
    expect(screen.getByText("Removing...")).toBeDisabled();
    expect(screen.getByText("Cancel")).toBeDisabled();
  });

  it("calls onCancel when overlay clicked", async () => {
    const user = userEvent.setup();
    const { props, container } = renderDialog();
    await user.click(container.firstElementChild!);
    expect(props.onCancel).toHaveBeenCalled();
  });

  it("calls onCancel after successful confirm", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Remove"));
    expect(props.onCancel).toHaveBeenCalled();
  });
});
