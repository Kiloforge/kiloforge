import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { AddProjectForm } from "./AddProjectForm";

// Mock fetcher to prevent real API calls (SSH key fetching)
vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn().mockResolvedValue({ keys: [] }),
  FetchError: class FetchError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, body: unknown) {
      super(`Request failed with status ${status}`);
      this.status = status;
      this.body = body;
    }
  },
}));

function renderForm(props?: Partial<Parameters<typeof AddProjectForm>[0]>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const defaultProps = {
    adding: false,
    error: null,
    onAdd: vi.fn().mockResolvedValue(true),
    onClearError: vi.fn(),
    ...props,
  };

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AddProjectForm {...defaultProps} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AddProjectForm", () => {
  it("shows add button initially", () => {
    renderForm();
    expect(screen.getByText("+ Add Project")).toBeInTheDocument();
  });

  it("expands on button click", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByText("+ Add Project"));
    expect(screen.getByLabelText("Remote URL")).toBeInTheDocument();
  });

  it("shows validation error for empty URL on submit", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.click(screen.getByText("Add Project"));
    expect(screen.getByText("Remote URL is required")).toBeInTheDocument();
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("shows validation error for invalid URL", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "not a url");
    await user.click(screen.getByText("Add Project"));
    expect(screen.getByText("Must be a git remote URL (SSH or HTTPS)")).toBeInTheDocument();
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("calls onAdd with valid HTTPS URL", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "https://github.com/user/repo.git");
    await user.click(screen.getByText("Add Project"));
    expect(onAdd).toHaveBeenCalledWith({ remote_url: "https://github.com/user/repo.git" });
  });

  it("accepts SSH URLs and calls onAdd", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "git@github.com:user/repo.git");
    await user.click(screen.getByText("Add Project"));
    expect(onAdd).toHaveBeenCalled();
  });

  it("collapses on cancel", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByText("+ Add Project"));
    expect(screen.getByLabelText("Remote URL")).toBeInTheDocument();
    await user.click(screen.getByText("Cancel"));
    expect(screen.getByText("+ Add Project")).toBeInTheDocument();
  });
});
