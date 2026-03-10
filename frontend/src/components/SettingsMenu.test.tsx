import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SettingsMenu } from "./SettingsMenu";

const mockRestartTour = vi.fn();

vi.mock("./tour/TourProvider", () => ({
  useTourContextSafe: () => ({ restartTour: mockRestartTour }),
}));

describe("SettingsMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders a gear icon button", () => {
    render(<SettingsMenu />);
    expect(screen.getByTitle("Settings")).toBeInTheDocument();
  });

  it("dropdown is hidden by default", () => {
    render(<SettingsMenu />);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("opens dropdown on click", () => {
    render(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
  });

  it("closes dropdown on second click", () => {
    render(<SettingsMenu />);
    const btn = screen.getByTitle("Settings");
    fireEvent.click(btn);
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
    fireEvent.click(btn);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("closes dropdown when clicking outside", () => {
    render(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
    fireEvent.mouseDown(document.body);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("calls restartTour and closes dropdown on Take Tour click", () => {
    render(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    fireEvent.click(screen.getByText("Take Tour"));
    expect(mockRestartTour).toHaveBeenCalledOnce();
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });
});
