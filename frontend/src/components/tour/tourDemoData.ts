import type { Project, BoardState, Track, TrackDetail, SwarmStatus } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";
import type { DemoState } from "./tourSteps";

/** Constant slug for the demo project shown during the guided tour. */
export const DEMO_PROJECT_SLUG = "example-project";

// ---------------------------------------------------------------------------
// Demo fixtures
// ---------------------------------------------------------------------------

export const DEMO_PROJECTS: Project[] = [
  {
    slug: DEMO_PROJECT_SLUG,
    repo_name: "example-project",
    origin_remote: "https://github.com/kiloforge/example-project",
    active: true,
  },
];

const now = new Date().toISOString();

export const DEMO_BOARD: BoardState = {
  columns: ["backlog", "approved", "in_progress", "done"],
  cards: {
    "auth-login_demo": {
      track_id: "auth-login_demo",
      title: "User authentication with login flow",
      type: "feature",
      column: "backlog",
      position: 0,
      moved_at: now,
      created_at: now,
    },
    "auth-register_demo": {
      track_id: "auth-register_demo",
      title: "User registration and onboarding",
      type: "feature",
      column: "backlog",
      position: 1,
      moved_at: now,
      created_at: now,
    },
    "auth-reset_demo": {
      track_id: "auth-reset_demo",
      title: "Password reset via email",
      type: "feature",
      column: "approved",
      position: 0,
      moved_at: now,
      created_at: now,
    },
    "api-middleware_demo": {
      track_id: "api-middleware_demo",
      title: "API auth middleware and JWT validation",
      type: "feature",
      column: "in_progress",
      position: 0,
      agent_status: "running",
      assigned_worker: "developer-1",
      trace_id: "demo-trace-001",
      moved_at: now,
      created_at: now,
    },
  },
};

export const DEMO_TRACKS: Track[] = [
  { id: "auth-login_demo", title: "User authentication with login flow", status: "pending", project: DEMO_PROJECT_SLUG },
  { id: "auth-register_demo", title: "User registration and onboarding", status: "pending", project: DEMO_PROJECT_SLUG },
  { id: "auth-reset_demo", title: "Password reset via email", status: "approved", project: DEMO_PROJECT_SLUG },
  { id: "api-middleware_demo", title: "API auth middleware and JWT validation", status: "in-progress", project: DEMO_PROJECT_SLUG },
];

/** Demo track details with dependency and conflict data for relationship visualization. */
export const DEMO_TRACK_DETAILS: TrackDetail[] = [
  {
    id: "auth-login_demo",
    title: "User authentication with login flow",
    status: "pending",
    type: "feature",
  },
  {
    id: "auth-register_demo",
    title: "User registration and onboarding",
    status: "pending",
    type: "feature",
    conflicts: [
      { track_id: "auth-login_demo", track_title: "User authentication with login flow", risk: "low", note: "Shared auth module" },
    ],
  },
  {
    id: "auth-reset_demo",
    title: "Password reset via email",
    status: "approved",
    type: "feature",
    dependencies: [
      { id: "auth-login_demo", title: "User authentication with login flow", status: "pending" },
    ],
  },
  {
    id: "api-middleware_demo",
    title: "API auth middleware and JWT validation",
    status: "in-progress",
    type: "feature",
    dependencies: [
      { id: "auth-login_demo", title: "User authentication with login flow", status: "pending" },
    ],
    conflicts: [
      { track_id: "auth-reset_demo", track_title: "Password reset via email", risk: "medium", note: "Shared auth routes" },
    ],
  },
];

/** Demo swarm status showing one active agent. */
export const DEMO_SWARM: SwarmStatus = {
  running: true,
  max_workers: 3,
  active_workers: 1,
  items: [
    {
      track_id: "api-middleware_demo",
      project_slug: DEMO_PROJECT_SLUG,
      status: "assigned",
      agent_id: "developer-1",
      enqueued_at: now,
      assigned_at: now,
    },
  ],
};

// ---------------------------------------------------------------------------
// Per-step demo state configurations
// ---------------------------------------------------------------------------

/** Injection entry: a query key and its demo data value. */
export interface DemoInjection {
  queryKey: readonly unknown[];
  data: unknown;
}

/** All injections needed to show the overview page with a demo project. */
const overviewInjections: DemoInjection[] = [
  { queryKey: queryKeys.projects, data: DEMO_PROJECTS },
  { queryKey: queryKeys.tracks(undefined), data: DEMO_TRACKS },
];

/** All injections needed to show the project page with a populated board. */
const projectInjections: DemoInjection[] = [
  ...overviewInjections,
  { queryKey: queryKeys.board(DEMO_PROJECT_SLUG), data: DEMO_BOARD },
  { queryKey: queryKeys.tracks(DEMO_PROJECT_SLUG), data: DEMO_TRACKS },
  { queryKey: queryKeys.swarm(DEMO_PROJECT_SLUG), data: DEMO_SWARM },
  { queryKey: [...queryKeys.tracks(DEMO_PROJECT_SLUG), "relations"] as const, data: DEMO_TRACK_DETAILS },
];

export const DEMO_STATES: Record<string, DemoState> = {
  welcome: {
    route: "/",
    inject: overviewInjections,
  },
  "add-project": {
    route: "/",
    inject: overviewInjections,
  },
  "open-project": {
    route: "/",
    inject: overviewInjections,
  },
  "setup-notice": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "swarm-capacity": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "generate-tracks": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "board-explanation": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "track-states": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "move-card": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "deps-conflicts": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "agent-types": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "notification-center": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  traces: {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  finish: {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
};
