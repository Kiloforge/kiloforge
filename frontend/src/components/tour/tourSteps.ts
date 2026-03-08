export interface TourStep {
  id: string;
  target: string; // CSS selector for highlight target
  title: string;
  content: string;
  page?: string; // route to navigate to
  placement?: "top" | "bottom" | "left" | "right";
  action?: "prefill-project" | "simulate-generate" | "wait-for-drag";
}

export const TOUR_STEPS: TourStep[] = [
  {
    id: "welcome",
    target: "body",
    title: "Welcome to Kiloforge!",
    content:
      "Let's walk through how to set up your first project and start automating development with AI agents.",
  },
  {
    id: "add-project",
    target: '[data-tour="add-project-form"]',
    title: "Add a Project",
    content:
      'Start by adding a Git repository. We\'ve prefilled an example URL for you \u2014 click "Add Project" to continue.',
    action: "prefill-project",
    placement: "bottom",
  },
  {
    id: "open-project",
    target: '[data-tour="project-card"]',
    title: "Open Your Project",
    content: "Click the project to open its dashboard and see the kanban board.",
    placement: "bottom",
  },
  {
    id: "setup-notice",
    target: '[data-tour="board-section"]',
    title: "Project Setup",
    content:
      "When you run 'kf add' from the CLI, Kiloforge sets up a local Gitea repo, SSH keys, and webhooks automatically. In the dashboard you can manage tracks and board cards.",
    page: "project",
    placement: "top",
  },
  {
    id: "generate-tracks",
    target: '[data-tour="generate-tracks"]',
    title: "Generate Tracks",
    content:
      'Describe what you want to build and Kiloforge generates implementation tracks. We\'ve prefilled an example prompt \u2014 click "Generate" to see how it works.',
    action: "simulate-generate",
    placement: "bottom",
  },
  {
    id: "board-explanation",
    target: '[data-tour="kanban-board"]',
    title: "The Kanban Board",
    content:
      "Tracks flow through columns: Backlog (new) \u2192 Approved (ready for dev) \u2192 In Progress \u2192 In Review \u2192 Done. Approving a track in the CLI triggers an AI agent to implement it.",
    placement: "top",
  },
  {
    id: "move-card",
    target: '[data-tour="board-card-first"]',
    title: "Try It: Move a Card",
    content:
      'Drag this card from Backlog to the Approved column. In production, approving a track kicks off a developer agent.',
    action: "wait-for-drag",
    placement: "right",
  },
];

export const TOTAL_STEPS = TOUR_STEPS.length;
