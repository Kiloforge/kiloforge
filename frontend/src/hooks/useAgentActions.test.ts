import { describe, it, expect } from "vitest";
import { canStop, canResume, canDelete, canReplace } from "./useAgentActions";
import type { Agent } from "../types/api";

const base: Agent = {
  id: "agent-1",
  role: "developer",
  ref: "track-abc",
  status: "running",
  session_id: "sess-1",
  pid: 1234,
  worktree_dir: "/tmp/wt",
  log_file: "/tmp/log",
  started_at: "2026-03-10T00:00:00Z",
  updated_at: "2026-03-10T00:01:00Z",
};

function agent(overrides: Partial<Agent> = {}): Agent {
  return { ...base, ...overrides };
}

describe("canStop", () => {
  it("returns true for running", () => {
    expect(canStop(agent({ status: "running" }))).toBe(true);
  });
  it("returns true for waiting", () => {
    expect(canStop(agent({ status: "waiting" }))).toBe(true);
  });
  it("returns false for suspended", () => {
    expect(canStop(agent({ status: "suspended" }))).toBe(false);
  });
});

describe("canResume", () => {
  it("returns true for suspended developer", () => {
    expect(canResume(agent({ status: "suspended", role: "developer" }))).toBe(true);
  });
  it("returns true for suspended interactive", () => {
    expect(canResume(agent({ status: "suspended", role: "interactive" }))).toBe(true);
  });
  it("returns true for suspended reviewer", () => {
    expect(canResume(agent({ status: "suspended", role: "reviewer" }))).toBe(true);
  });
  it("returns true for force-killed agent", () => {
    expect(canResume(agent({ status: "force-killed" }))).toBe(true);
  });
  it("returns true for stopped interactive", () => {
    expect(canResume(agent({ status: "stopped", role: "interactive" }))).toBe(true);
  });
  it("returns true for completed interactive", () => {
    expect(canResume(agent({ status: "completed", role: "interactive" }))).toBe(true);
  });
  it("returns true for failed interactive", () => {
    expect(canResume(agent({ status: "failed", role: "interactive" }))).toBe(true);
  });
  it("returns false for resume-failed", () => {
    expect(canResume(agent({ status: "resume-failed" }))).toBe(false);
  });
  it("returns false for replaced", () => {
    expect(canResume(agent({ status: "replaced" }))).toBe(false);
  });
  it("returns false for running", () => {
    expect(canResume(agent({ status: "running" }))).toBe(false);
  });
  it("returns false for stopped developer (non-interactive one-shot)", () => {
    expect(canResume(agent({ status: "stopped", role: "developer" }))).toBe(false);
  });
});

describe("canReplace", () => {
  it("returns true for resume-failed", () => {
    expect(canReplace(agent({ status: "resume-failed" }))).toBe(true);
  });
  it("returns true for force-killed", () => {
    expect(canReplace(agent({ status: "force-killed" }))).toBe(true);
  });
  it("returns false for running", () => {
    expect(canReplace(agent({ status: "running" }))).toBe(false);
  });
  it("returns false for suspended", () => {
    expect(canReplace(agent({ status: "suspended" }))).toBe(false);
  });
  it("returns false for completed", () => {
    expect(canReplace(agent({ status: "completed" }))).toBe(false);
  });
});

describe("canDelete", () => {
  it("returns false for running", () => {
    expect(canDelete(agent({ status: "running" }))).toBe(false);
  });
  it("returns true for suspended", () => {
    expect(canDelete(agent({ status: "suspended" }))).toBe(true);
  });
  it("returns true for resume-failed", () => {
    expect(canDelete(agent({ status: "resume-failed" }))).toBe(true);
  });
});
