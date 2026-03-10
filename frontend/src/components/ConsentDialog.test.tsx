import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConsentDialog } from "./ConsentDialog";

function renderDialog(overrides: Partial<{ onAccept: () => void; onDeny: () => void }> = {}) {
  const props = {
    onAccept: vi.fn(),
    onDeny: vi.fn(),
    ...overrides,
  };
  return { ...render(<ConsentDialog {...props} />), props };
}

describe("ConsentDialog", () => {
  it("renders title and permission message", () => {
    renderDialog();
    expect(screen.getByText("Agent Permissions Required")).toBeInTheDocument();
    expect(screen.getByText(/Dangerously bypass permissions/)).toBeInTheDocument();
  });

  it("renders Accept and Deny buttons", () => {
    renderDialog();
    expect(screen.getByText("Accept")).toBeInTheDocument();
    expect(screen.getByText("Deny")).toBeInTheDocument();
  });

  it("calls onAccept when Accept clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Accept"));
    expect(props.onAccept).toHaveBeenCalled();
  });

  it("calls onDeny when Deny clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Deny"));
    expect(props.onDeny).toHaveBeenCalled();
  });

  it("calls onDeny when overlay clicked", async () => {
    const user = userEvent.setup();
    const { props, container } = renderDialog();
    // Click the overlay (outermost div)
    await user.click(container.firstElementChild!);
    expect(props.onDeny).toHaveBeenCalled();
  });

  it("does not call onDeny when dialog content clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByText("Agent Permissions Required"));
    expect(props.onDeny).not.toHaveBeenCalled();
  });

  it("shows Accepting... and disables buttons after accept", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByText("Accept"));
    expect(screen.getByText("Accepting...")).toBeInTheDocument();
    expect(screen.getByText("Accepting...")).toBeDisabled();
    expect(screen.getByText("Deny")).toBeDisabled();
  });
});
