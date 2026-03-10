import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AdminPanel } from "./AdminPanel";

vi.mock("../hooks/useConsent", () => ({
  useConsent: () => ({
    showDialog: false,
    requestConsent: vi.fn(),
    accept: vi.fn(),
    deny: vi.fn(),
  }),
}));

function renderPanel(overrides: Partial<Parameters<typeof AdminPanel>[0]> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const props = {
    running: false,
    onStartOperation: vi.fn(),
    ...overrides,
  };

  return {
    ...render(
      <QueryClientProvider client={queryClient}>
        <AdminPanel {...props} />
      </QueryClientProvider>,
    ),
    props,
  };
}

describe("AdminPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all three operation buttons", () => {
    renderPanel();
    expect(screen.getByText("Bulk Archive")).toBeInTheDocument();
    expect(screen.getByText("Compact Archive")).toBeInTheDocument();
    expect(screen.getByText("Generate Report")).toBeInTheDocument();
  });

  it("disables buttons when running is true", () => {
    renderPanel({ running: true });
    expect(screen.getByText("Bulk Archive")).toBeDisabled();
    expect(screen.getByText("Compact Archive")).toBeDisabled();
    expect(screen.getByText("Generate Report")).toBeDisabled();
  });

  it("disables buttons when disabled prop is true", () => {
    renderPanel({ disabled: true, disabledReason: "No project selected" });
    const btn = screen.getByText("Bulk Archive");
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute("title", "No project selected");
  });

  it("buttons are enabled by default", () => {
    renderPanel();
    expect(screen.getByText("Bulk Archive")).not.toBeDisabled();
    expect(screen.getByText("Compact Archive")).not.toBeDisabled();
    expect(screen.getByText("Generate Report")).not.toBeDisabled();
  });

  it("buttons are clickable", async () => {
    const user = userEvent.setup();
    renderPanel();
    await user.click(screen.getByText("Bulk Archive"));
    // No crash — mutation will be triggered
  });
});
