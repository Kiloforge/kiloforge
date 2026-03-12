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
    await user.click(screen.getByText("Clone Project"));
    expect(screen.getByText("Remote URL is required")).toBeInTheDocument();
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("applies error styling to URL input on validation failure", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByText("+ Add Project"));
    await user.click(screen.getByText("Clone Project"));
    const urlInput = screen.getByLabelText("Remote URL");
    expect(urlInput.className).toMatch(/inputError/);
  });

  it("removes error styling when user types in URL input", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByText("+ Add Project"));
    await user.click(screen.getByText("Clone Project"));
    const urlInput = screen.getByLabelText("Remote URL");
    expect(urlInput.className).toMatch(/inputError/);
    await user.type(urlInput, "a");
    expect(urlInput.className).not.toMatch(/inputError/);
  });

  it("shows validation error for invalid URL", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "not a url");
    await user.click(screen.getByText("Clone Project"));
    expect(screen.getByText("Must be a git remote URL (SSH or HTTPS)")).toBeInTheDocument();
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("calls onAdd with valid HTTPS URL", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "https://github.com/user/repo.git");
    await user.click(screen.getByText("Clone Project"));
    expect(onAdd).toHaveBeenCalledWith({ remote_url: "https://github.com/user/repo.git" });
  });

  it("accepts SSH URLs and calls onAdd", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockResolvedValue(true);
    renderForm({ onAdd });
    await user.click(screen.getByText("+ Add Project"));
    await user.type(screen.getByLabelText("Remote URL"), "git@github.com:user/repo.git");
    await user.click(screen.getByText("Clone Project"));
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

  describe("Create New mode", () => {
    it("shows mode toggle with Clone and Create buttons", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      expect(screen.getByText("Clone from remote")).toBeInTheDocument();
      expect(screen.getByText("Create new")).toBeInTheDocument();
    });

    it("hides URL field when Create new is selected", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      expect(screen.getByLabelText("Remote URL")).toBeInTheDocument();
      await user.click(screen.getByText("Create new"));
      expect(screen.queryByLabelText("Remote URL")).not.toBeInTheDocument();
    });

    it("shows name field as required in create mode", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Create new"));
      // Name label should not show "(optional)"
      const nameLabel = screen.getByText("Name");
      expect(nameLabel.querySelector(".optional") ?? nameLabel.textContent).not.toContain("optional");
    });

    it("validates empty name in create mode", async () => {
      const user = userEvent.setup();
      const onAdd = vi.fn().mockResolvedValue(true);
      renderForm({ onAdd });
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Create new"));
      await user.click(screen.getByText("Create Project"));
      expect(screen.getByText("Project name is required")).toBeInTheDocument();
      expect(onAdd).not.toHaveBeenCalled();
    });

    it("applies error styling to name input on validation failure", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Create new"));
      await user.click(screen.getByText("Create Project"));
      const nameInput = screen.getByLabelText("Name");
      expect(nameInput.className).toMatch(/inputError/);
    });

    it("submits with name only in create mode", async () => {
      const user = userEvent.setup();
      const onAdd = vi.fn().mockResolvedValue(true);
      renderForm({ onAdd });
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Create new"));
      await user.type(screen.getByLabelText("Name"), "my-project");
      await user.click(screen.getByText("Create Project"));
      expect(onAdd).toHaveBeenCalledWith({ name: "my-project" });
    });

    it("restores URL field when switching back to clone mode", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Create new"));
      expect(screen.queryByLabelText("Remote URL")).not.toBeInTheDocument();
      await user.click(screen.getByText("Clone from remote"));
      expect(screen.getByLabelText("Remote URL")).toBeInTheDocument();
    });
  });

  describe("Local repo mode", () => {
    it("shows Local repo tab in mode toggle", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      expect(screen.getByText("Local repo")).toBeInTheDocument();
    });

    it("shows path input when Local repo is selected", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      expect(screen.getByLabelText("Local path")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("/path/to/repo")).toBeInTheDocument();
    });

    it("hides Remote URL field in local mode", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      expect(screen.queryByLabelText("Remote URL")).not.toBeInTheDocument();
    });

    it("does not show SSH key selector in local mode", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      expect(screen.queryByLabelText("SSH Key")).not.toBeInTheDocument();
    });

    it("shows submit button as 'Add Local Repo'", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      expect(screen.getByText("Add Local Repo")).toBeInTheDocument();
    });

    it("validates empty path on submit", async () => {
      const user = userEvent.setup();
      const onAdd = vi.fn().mockResolvedValue(true);
      renderForm({ onAdd });
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      await user.click(screen.getByText("Add Local Repo"));
      expect(screen.getByText("Local path is required")).toBeInTheDocument();
      expect(onAdd).not.toHaveBeenCalled();
    });

    it("submits with local_path field", async () => {
      const user = userEvent.setup();
      const onAdd = vi.fn().mockResolvedValue(true);
      renderForm({ onAdd });
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      await user.type(screen.getByLabelText("Local path"), "/home/user/my-repo");
      await user.click(screen.getByText("Add Local Repo"));
      expect(onAdd).toHaveBeenCalledWith({ local_path: "/home/user/my-repo" });
    });

    it("includes optional name and output_dir when provided", async () => {
      const user = userEvent.setup();
      const onAdd = vi.fn().mockResolvedValue(true);
      renderForm({ onAdd });
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      await user.type(screen.getByLabelText("Local path"), "/repos/proj");
      await user.type(screen.getByLabelText(/Name/), "my-proj");
      await user.type(screen.getByLabelText(/Output directory/), "/out");
      await user.click(screen.getByText("Add Local Repo"));
      expect(onAdd).toHaveBeenCalledWith({
        local_path: "/repos/proj",
        name: "my-proj",
        output_dir: "/out",
      });
    });

    it("shows name as optional in local mode", async () => {
      const user = userEvent.setup();
      renderForm();
      await user.click(screen.getByText("+ Add Project"));
      await user.click(screen.getByText("Local repo"));
      const nameLabel = screen.getByLabelText(/Name/).closest("div")!.querySelector("label")!;
      expect(nameLabel.textContent).toContain("optional");
    });
  });
});
