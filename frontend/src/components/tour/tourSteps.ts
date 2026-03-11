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
      "When you run 'kf add' from the CLI, Kiloforge sets up a local Gitea repo, SSH keys, and webhooks automatically. In the Command Deck you can manage tracks, agents, and the board.",
    placement: "top",
    demoState: DEMO_STATES["setup-notice"],
  },
  {
    id: "swarm-capacity",
    target: '[data-tour="swarm-panel"]',
    title: "Swarm Capacity",
    content:
      "The Swarm Panel shows how many AI agents are active and your configured maximum. You can start/stop the swarm and adjust the max agent count to control parallelism.",
    placement: "bottom",
    demoState: DEMO_STATES["swarm-capacity"],
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
      "Tracks flow through three columns: Backlog (new) \u2192 Approved (ready for agents) \u2192 In Progress (being worked on). Completed tracks leave the board automatically.",
    placement: "top",
    demoState: DEMO_STATES["board-explanation"],
  },
  {
    id: "track-states",
    target: '[data-tour="kanban-board"]',
    title: "Track Lifecycle",
    content:
      "Each track moves through stages: generated tracks land in Backlog for review. Approve them to signal readiness. When an autonomous agent picks one up, it moves to In Progress. Completed work exits the board.",
    placement: "top",
    demoState: DEMO_STATES["track-states"],
  },
  {
    id: "move-card",
    target: '[data-tour="board-card-first"]',
    title: "Try It: Move a Card",
    content:
      'Try dragging this card from Backlog to the Approved column. In production, approving a track kicks off an autonomous agent. You can also skip this step below.',
    action: "wait-for-drag",
    placement: "right",
    demoState: DEMO_STATES["move-card"],
  },
  {
    id: "deps-conflicts",
    target: '[data-tour="relationship-toggle"]',
    title: "Dependencies & Conflicts",
    content:
      "Toggle this to see relationship lines between cards. Blue lines show dependencies (one track must finish before another). Red, orange, and yellow lines indicate conflict risk levels \u2014 tracks that touch overlapping files.",
    placement: "bottom",
    demoState: DEMO_STATES["deps-conflicts"],
  },
  {
    id: "agent-types",
    target: '[data-tour="swarm-panel"]',
    title: "Agent Types",
    content:
      "Kiloforge runs two kinds of agents. Autonomous agents (developers) work in background worktrees implementing tracks. Interactive agents give you a live terminal session for architecture, debugging, or ad-hoc tasks.",
    placement: "bottom",
    demoState: DEMO_STATES["agent-types"],
  },
  {
    id: "notification-center",
    target: '[data-tour="notification-bell"]',
    title: "Notification Center",
    content:
      "The notification bell alerts you when agents need attention \u2014 review requests, merge conflicts, or stuck builds. Click it to see all pending notifications at a glance.",
    placement: "bottom",
    demoState: DEMO_STATES["notification-center"],
  },
  {
    id: "traces",
    target: '[data-tour="trace-link"]',
    title: "Traces & Observability",
    content:
      "Cards with active agents show a Trace link. Click it to see an OpenTelemetry timeline of what the agent did \u2014 every tool call, file edit, and test run visualized as spans.",
    placement: "top",
    demoState: DEMO_STATES["traces"],
  },
  {
    id: "finish",
    target: "body",
    title: "You're Ready to Forge!",
    content:
      "You've seen the key features. Add a real project with 'kf add <remote>' or use the form on the overview page to get started. Happy forging, Kiloforger!",
    demoState: DEMO_STATES["finish"],
  },
];

export const TOTAL_STEPS = TOUR_STEPS.length;
