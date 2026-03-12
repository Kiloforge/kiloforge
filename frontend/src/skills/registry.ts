export interface SkillEntry {
  role: string;
  label: string;
  description: string;
  slashCommand: string;
  requiredSkill: string;
  requiresProject: boolean;
  placeholder: string;
}

export const SKILL_REGISTRY: SkillEntry[] = [
  {
    role: "interactive",
    label: "Interactive",
    description: "General-purpose kf-aware assistant",
    slashCommand: "/kf-interactive",
    requiredSkill: "kf-interactive",
    requiresProject: false,
    placeholder: "Ask anything about the project...",
  },
  {
    role: "architect",
    label: "Architect",
    description: "Research codebase and generate implementation tracks",
    slashCommand: "/kf-architect",
    requiredSkill: "kf-architect",
    requiresProject: true,
    placeholder: "Describe the feature or change you want to plan...",
  },
  {
    role: "advisor-product",
    label: "Product Advisor",
    description: "Product design, branding, and competitive analysis",
    slashCommand: "/kf-advisor-product",
    requiredSkill: "kf-advisor-product",
    requiresProject: true,
    placeholder: "Describe what you need product guidance on...",
  },
  {
    role: "advisor-reliability",
    label: "Reliability Advisor",
    description: "Testing coverage, linting, type safety, and CI gate audits",
    slashCommand: "/kf-advisor-reliability",
    requiredSkill: "kf-advisor-reliability",
    requiresProject: true,
    placeholder: "Describe what you want audited (e.g., testing gaps, CI coverage)...",
  },
];
