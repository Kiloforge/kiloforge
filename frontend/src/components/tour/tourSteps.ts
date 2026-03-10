/** Per-step demo state: which route to navigate to and what query data to inject. */
export interface DemoState {
  route: string;
  inject: { queryKey: readonly unknown[]; data: unknown }[];
}

export interface TourStep {
  id: string;
  target: string; // CSS selector for highlight target
  title: string;
  content: string;
  page?: string; // route to navigate to (legacy, superseded by demoState.route)
  placement?: "top" | "bottom" | "left" | "right";
  action?: "prefill-project" | "simulate-generate" | "wait-for-drag";
  demoState?: DemoState;
}

// Lazy-loaded demo states — imported at the top of the module but only
// referenced via the `demoState` field, so tree-shaking is straightforward.
import { DEMO_STATES } from "./tourDemoData";

export const TOUR_STEPS: TourStep[] = [
  {
    id: "welcome",
    target: "body",
    title: "Welcome, Kiloforger!",
    content:
      "Let's walk through how to set up your first project and start forging code with AI agents.",
    demoState: DEMO_STATES["welcome"],
  },
  {
    id: "add-project",
    target: '[data-tour="add-project-form"]',
    title: "Add a Project",
    content:
      'This is the Add Project form. In the tour, a demo project is already loaded \u2014 click "Next" to continue.',
    placement: "bottom",
    demoState: DEMO_STATES["add-project"],
  },
  {
    id: "open-project",
    target: '[data-tour="project-card"]',
    title: "Open Your Project",
    content: "Click the project to open the Command Deck and see the kanban board.",
    placement: "bottom",
    demoState: DEMO_STATES["open-project"],
  },
  {
    id: "setup-notice",
    target: '[data-tour="board-section"]',
    title: "Project Setup",
    content:
      "When you run 'kf add' from the CLI, Kiloforge sets up a local Gitea repo, SSH keys, and webhooks automatically. In the Command Deck you can manage tracks and board cards.",
    placement: "top",
    demoState: DEMO_STATES["setup-notice"],
  },
  {
    id: "generate-tracks",
    target: '[data-tour="generate-tracks"]',
    title: "Generate Tracks",
    content:
      'Describe what you want to build and Kiloforge generates implementation tracks for you, Kiloforger. We\'ve prefilled an example prompt \u2014 click "Generate" to see how it works.',
    action: "simulate-generate",
    placement: "bottom",
    demoState: DEMO_STATES["generate-tracks"],
  },
  {
    id: "board-explanation",
    target: '[data-tour="kanban-board"]',
    title: "The Kanban Board",
    content:
      "Tracks flow through columns: Backlog (new) \u2192 Approved (ready for dev) \u2192 In Progress. Completed tracks leave the board automatically. Approving a track in the CLI triggers an AI agent to implement it.",
    placement: "top",
    demoState: DEMO_STATES["board-explanation"],
  },
  {
    id: "move-card",
    target: '[data-tour="board-card-first"]',
    title: "Try It: Move a Card",
    content:
      'Try dragging this card from Backlog to the Approved column. In production, approving a track kicks off a developer agent. You can also skip this step below.',
    action: "wait-for-drag",
    placement: "right",
    demoState: DEMO_STATES["move-card"],
  },
];

export const TOTAL_STEPS = TOUR_STEPS.length;
