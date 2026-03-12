import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SkillsPalette } from "./SkillsPalette";
import { SKILL_REGISTRY } from "../skills/registry";

// Mock useSkillsStatus
vi.mock("../hooks/useSkillsStatus", () => ({
  useSkillsStatus: () => ({
    status: {
      installed_version: "1.0.0",
      update_available: false,
      skills: [
        { name: "kf-interactive", modified: false },
        { name: "kf-architect", modified: false },
        { name: "kf-advisor-product", modified: true },
      ],
    },
    loading: false,
    updating: false,
    triggerUpdate: vi.fn(),
    refresh: vi.fn(),
  }),
}));

describe("SkillsPalette", () => {
  const defaultProps = {
    onClose: vi.fn(),
    onSelectSkill: vi.fn(),
  };

  it("renders skill cards with name, description, and install status", () => {
    render(<SkillsPalette {...defaultProps} />);

    for (const entry of SKILL_REGISTRY) {
      expect(screen.getByText(entry.label)).toBeTruthy();
      expect(screen.getByText(entry.description)).toBeTruthy();
    }
  });

  it("shows installed badge for installed skills", () => {
    render(<SkillsPalette {...defaultProps} />);
    const badges = screen.getAllByText("Installed");
    expect(badges.length).toBeGreaterThanOrEqual(1);
  });

  it("calls onSelectSkill when a skill card is clicked", () => {
    const onSelectSkill = vi.fn();
    render(<SkillsPalette {...defaultProps} onSelectSkill={onSelectSkill} />);

    fireEvent.click(screen.getByText("Interactive"));
    expect(onSelectSkill).toHaveBeenCalledWith("interactive");
  });

  it("closes on Escape key", () => {
    const onClose = vi.fn();
    render(<SkillsPalette {...defaultProps} onClose={onClose} />);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  it("marks project-required skills when no project is active", () => {
    render(<SkillsPalette {...defaultProps} hasProject={false} />);
    const projectBadges = screen.getAllByText("Requires project");
    expect(projectBadges.length).toBeGreaterThanOrEqual(1);
  });
});
