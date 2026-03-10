import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ModelWarningBanner } from "./ModelWarningBanner";

beforeEach(() => {
  localStorage.clear();
});

describe("ModelWarningBanner", () => {
  it("renders banner with notice text", () => {
    render(<ModelWarningBanner />);
    expect(screen.getByText(/Kiloforge requires Claude Code/)).toBeInTheDocument();
  });

  it("renders Understood button", () => {
    render(<ModelWarningBanner />);
    expect(screen.getByText("Understood")).toBeInTheDocument();
  });

  it("dismisses banner on Understood click", async () => {
    const user = userEvent.setup();
    render(<ModelWarningBanner />);
    await user.click(screen.getByText("Understood"));
    expect(screen.queryByText("Understood")).not.toBeInTheDocument();
  });

  it("persists dismissal to localStorage", async () => {
    const user = userEvent.setup();
    render(<ModelWarningBanner />);
    await user.click(screen.getByText("Understood"));
    expect(localStorage.getItem("kf_model_warning_dismissed")).toBe("1");
  });

  it("does not render when already dismissed", () => {
    localStorage.setItem("kf_model_warning_dismissed", "1");
    const { container } = render(<ModelWarningBanner />);
    expect(container.innerHTML).toBe("");
  });
});
