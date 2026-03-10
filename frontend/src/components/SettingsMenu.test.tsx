import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SettingsMenu } from "./SettingsMenu";

const mockRestartTour = vi.fn();
const mockUpdateConfig = vi.fn().mockResolvedValue(true);

vi.mock("./tour/TourProvider", () => ({
  useTourContextSafe: () => ({ restartTour: mockRestartTour }),
}));

vi.mock("../hooks/useConfig", () => ({
  useConfig: () => ({
    config: { dashboard_enabled: true, analytics_enabled: true },
    loading: false,
    updating: false,
    updateConfig: mockUpdateConfig,
  }),
}));

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("SettingsMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders a gear icon button", () => {
    renderWithQuery(<SettingsMenu />);
    expect(screen.getByTitle("Settings")).toBeInTheDocument();
  });

  it("dropdown is hidden by default", () => {
    renderWithQuery(<SettingsMenu />);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("opens dropdown on click", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
  });

  it("shows analytics toggle in dropdown", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    expect(screen.getByText("Anonymous usage data")).toBeInTheDocument();
    expect(screen.getByText("Help improve kiloforge")).toBeInTheDocument();
    expect(screen.getByRole("switch")).toBeInTheDocument();
  });

  it("analytics toggle reflects enabled state", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    const toggle = screen.getByRole("switch");
    expect(toggle.getAttribute("aria-checked")).toBe("true");
  });

  it("clicking analytics toggle calls updateConfig with false", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    fireEvent.click(screen.getByRole("switch"));
    expect(mockUpdateConfig).toHaveBeenCalledWith({ analytics_enabled: false });
  });

  it("closes dropdown on second click", () => {
    renderWithQuery(<SettingsMenu />);
    const btn = screen.getByTitle("Settings");
    fireEvent.click(btn);
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
    fireEvent.click(btn);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("closes dropdown when clicking outside", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    expect(screen.getByText("Take Tour")).toBeInTheDocument();
    fireEvent.mouseDown(document.body);
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });

  it("calls restartTour and closes dropdown on Take Tour click", () => {
    renderWithQuery(<SettingsMenu />);
    fireEvent.click(screen.getByTitle("Settings"));
    fireEvent.click(screen.getByText("Take Tour"));
    expect(mockRestartTour).toHaveBeenCalledOnce();
    expect(screen.queryByText("Take Tour")).not.toBeInTheDocument();
  });
});
