import { describe, it, expect } from "vitest";
import { queryKeys } from "./queryKeys";

describe("queryKeys", () => {
  it("returns static keys for simple queries", () => {
    expect(queryKeys.agents()).toEqual(["agents", "active"]);
    expect(queryKeys.config).toEqual(["config"]);
    expect(queryKeys.projects).toEqual(["projects"]);
    expect(queryKeys.quota).toEqual(["quota"]);
    expect(queryKeys.skills).toEqual(["skills"]);
    expect(queryKeys.traces).toEqual(["traces"]);
    expect(queryKeys.status).toEqual(["status"]);
    expect(queryKeys.sshKeys).toEqual(["sshKeys"]);
    expect(queryKeys.tour).toEqual(["tour"]);
    expect(queryKeys.tourDemoBoard).toEqual(["tour", "demo-board"]);
    expect(queryKeys.preflight).toEqual(["preflight"]);
  });

  it("agent() produces scoped key", () => {
    expect(queryKeys.agent("abc123")).toEqual(["agents", "abc123"]);
  });

  it("board() produces scoped key", () => {
    expect(queryKeys.board("my-project")).toEqual(["board", "my-project"]);
  });

  it("trace() produces scoped key", () => {
    expect(queryKeys.trace("trace-1")).toEqual(["traces", "trace-1"]);
  });

  it("tracks() includes optional project", () => {
    expect(queryKeys.tracks()).toEqual(["tracks", undefined]);
    expect(queryKeys.tracks("proj")).toEqual(["tracks", "proj"]);
  });

  it("paginated keys include 'paginated' suffix", () => {
    expect(queryKeys.agentsPaginated()).toEqual(["agents", "active", "paginated"]);
    expect(queryKeys.agentsPaginated(false)).toEqual(["agents", "all", "paginated"]);
    expect(queryKeys.tracesPaginated).toEqual(["traces", "paginated"]);
    expect(queryKeys.tracksPaginated()).toEqual(["tracks", undefined, "paginated"]);
    expect(queryKeys.tracksPaginated("proj")).toEqual(["tracks", "proj", "paginated"]);
  });

  it("syncStatus() produces scoped key", () => {
    expect(queryKeys.syncStatus("my-slug")).toEqual(["syncStatus", "my-slug"]);
  });

  it("trackDetail() produces scoped key", () => {
    expect(queryKeys.trackDetail("track-1", "my-project")).toEqual(["tracks", "track-1", "my-project"]);
  });
});
