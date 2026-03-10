import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../hooks/useSkillsStatus", () => ({
  useSkillsStatus: vi.fn(),
}));

import { useSkillsStatus } from "../hooks/useSkillsStatus";
import { SkillsBanner } from "./SkillsBanner";

const mockUseSkillsStatus = vi.mocked(useSkillsStatus);

function setup(overrides: Partial<ReturnType<typeof useSkillsStatus>> = {}) {
  const defaults: ReturnType<typeof useSkillsStatus> = {
    status: { skills: [{ name: "dev", modified: false }], update_available: false, installed_version: "1.0", available_version: "1.0" },
    loading: false,
    updating: false,
    triggerUpdate: vi.fn(),
    refresh: vi.fn(),
    ...overrides,
  };
  mockUseSkillsStatus.mockReturnValue(defaults);
  return { ...render(<SkillsBanner />), ...defaults };
}

describe("SkillsBanner", () => {
  it("returns null when loading", () => {
    const { container } = setup({ loading: true, status: null });
    expect(container.innerHTML).toBe("");
  });

  it("returns null when status is null", () => {
    const { container } = setup({ status: null });
    expect(container.innerHTML).toBe("");
  });

  it("shows no-skills banner when skills array is empty", () => {
    setup({ status: { skills: [], update_available: false, installed_version: "", available_version: "1.0" } });
    expect(screen.getByText(/No skills installed/)).toBeInTheDocument();
    expect(screen.getByText("Install Skills")).toBeInTheDocument();
  });

  it("calls triggerUpdate(true) when Install Skills clicked", async () => {
    const user = userEvent.setup();
    const { triggerUpdate } = setup({ status: { skills: [], update_available: false, installed_version: "", available_version: "1.0" } });
    await user.click(screen.getByText("Install Skills"));
    expect(triggerUpdate).toHaveBeenCalledWith(true);
  });

  it("shows update banner when update available", () => {
    setup({ status: { skills: [{ name: "dev", modified: false }], update_available: true, installed_version: "1.0", available_version: "2.0" } });
    expect(screen.getByText(/Skills update available/)).toBeInTheDocument();
    expect(screen.getByText("Update")).toBeInTheDocument();
  });

  it("notes modified skills in update banner", () => {
    setup({ status: { skills: [{ name: "dev", modified: true }], update_available: true, installed_version: "1.0", available_version: "2.0" } });
    expect(screen.getByText(/local modifications/)).toBeInTheDocument();
  });

  it("returns null when skills installed and no update", () => {
    const { container } = setup();
    expect(container.innerHTML).toBe("");
  });

  it("disables button when updating", () => {
    setup({ updating: true, status: { skills: [], update_available: false, installed_version: "", available_version: "1.0" } });
    expect(screen.getByText("Installing...")).toBeDisabled();
  });
});
