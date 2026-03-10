import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AdvisorHub } from "./AdvisorHub";

function setup(overrides = {}) {
  const props = {
    agents: [],
    onLaunch: vi.fn(),
    onViewLog: vi.fn(),
    ...overrides,
  };
  render(<AdvisorHub {...props} />);
  return props;
}

function openDialog() {
  // Click the first advisor card to open the dialog
  const cards = screen.getAllByText("Click to launch");
  fireEvent.click(cards[0].closest("button")!);
}

describe("AdvisorHub", () => {
  it("renders advisor cards", () => {
    setup();
    expect(screen.getByText("Product Advisor")).toBeTruthy();
    expect(screen.getByText("Reliability Advisor")).toBeTruthy();
  });

  it("shows project name input in launch dialog", () => {
    setup();
    openDialog();
    expect(screen.getByPlaceholderText("my-project")).toBeTruthy();
    expect(screen.getByText("Project Name")).toBeTruthy();
  });

  it("disables Start button when project name is empty", () => {
    setup();
    openDialog();
    const startBtn = screen.getByText("Start Advisor");
    expect(startBtn).toBeDisabled();
  });

  it("shows validation error for invalid project name", () => {
    setup();
    openDialog();
    const input = screen.getByPlaceholderText("my-project");
    fireEvent.change(input, { target: { value: "INVALID NAME!" } });
    expect(screen.getByText(/lowercase letters/i)).toBeTruthy();
  });

  it("enables Start button with valid project name", () => {
    setup();
    openDialog();
    const input = screen.getByPlaceholderText("my-project");
    fireEvent.change(input, { target: { value: "my-project" } });
    const startBtn = screen.getByText("Start Advisor");
    expect(startBtn).not.toBeDisabled();
  });

  it("calls onLaunch with role, prompt, and project name", () => {
    const { onLaunch } = setup();
    openDialog();
    const projectInput = screen.getByPlaceholderText("my-project");
    fireEvent.change(projectInput, { target: { value: "test-proj" } });
    const textarea = document.querySelector("textarea")!;
    fireEvent.change(textarea, { target: { value: "analyze this" } });
    fireEvent.click(screen.getByText("Start Advisor"));
    expect(onLaunch).toHaveBeenCalledWith("advisor-product", "analyze this", "test-proj");
  });

  it("resets project name when dialog closes", () => {
    setup();
    openDialog();
    const input = screen.getByPlaceholderText("my-project");
    fireEvent.change(input, { target: { value: "test-proj" } });
    fireEvent.click(screen.getByText("Cancel"));
    // Re-open
    openDialog();
    const newInput = screen.getByPlaceholderText("my-project");
    expect((newInput as HTMLInputElement).value).toBe("");
  });
});
