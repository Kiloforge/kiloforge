import type { Project, BoardState, Track } from "../../types/api";
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
  "generate-tracks": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "board-explanation": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
  "move-card": {
    route: `/projects/${DEMO_PROJECT_SLUG}`,
    inject: projectInjections,
  },
};
