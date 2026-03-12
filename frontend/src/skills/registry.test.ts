import { describe, it, expect } from "vitest";
import { SKILL_REGISTRY } from "./registry";

describe("SKILL_REGISTRY", () => {
  it("exports all known roles with required fields", () => {
    expect(SKILL_REGISTRY.length).toBeGreaterThanOrEqual(4);
    for (const entry of SKILL_REGISTRY) {
      expect(entry.role).toBeTruthy();
      expect(entry.label).toBeTruthy();
      expect(entry.description).toBeTruthy();
      expect(entry.slashCommand).toBeTruthy();
      expect(entry.requiredSkill).toBeTruthy();
      expect(entry.placeholder).toBeTruthy();
      expect(typeof entry.requiresProject).toBe("boolean");
    }
  });

  it("includes interactive, architect, and advisor roles", () => {
    const roles = SKILL_REGISTRY.map((e) => e.role);
    expect(roles).toContain("interactive");
    expect(roles).toContain("architect");
    expect(roles).toContain("advisor-product");
    expect(roles).toContain("advisor-reliability");
  });

  it("has unique role values", () => {
    const roles = SKILL_REGISTRY.map((e) => e.role);
    expect(new Set(roles).size).toBe(roles.length);
  });
});
