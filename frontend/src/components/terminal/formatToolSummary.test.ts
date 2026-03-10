import { describe, it, expect } from "vitest";
import { formatToolSummary } from "./formatToolSummary";

describe("formatToolSummary", () => {
  it("returns tool name when no input provided", () => {
    expect(formatToolSummary("Bash", undefined)).toEqual({ label: "Bash" });
  });

  it("returns 'unknown' when no tool name provided", () => {
    expect(formatToolSummary(undefined, undefined)).toEqual({ label: "unknown" });
  });

  describe("Bash", () => {
    it("uses description as label when present", () => {
      const result = formatToolSummary("Bash", {
        command: "ls -la",
        description: "List files",
      });
      expect(result.label).toBe("List files");
      expect(result.detail).toBe("ls -la");
    });

    it("falls back to command when no description", () => {
      const result = formatToolSummary("Bash", { command: "npm test" });
      expect(result.label).toBe("npm test");
      expect(result.detail).toBe("npm test");
    });

    it("returns 'Bash' when empty input", () => {
      expect(formatToolSummary("Bash", {})).toEqual({ label: "Bash" });
    });
  });

  describe("Read", () => {
    it("shows file_path as label", () => {
      const result = formatToolSummary("Read", { file_path: "/src/main.ts" });
      expect(result.label).toBe("/src/main.ts");
    });

    it("includes extra params in detail", () => {
      const result = formatToolSummary("Read", {
        file_path: "/src/main.ts",
        offset: 10,
        limit: 50,
      });
      expect(result.label).toBe("/src/main.ts");
      expect(result.detail).toContain('"offset": 10');
    });
  });

  describe("Write", () => {
    it("shows file_path as label", () => {
      const result = formatToolSummary("Write", { file_path: "/src/new.ts", content: "code" });
      expect(result.label).toBe("/src/new.ts");
    });
  });

  describe("Edit", () => {
    it("shows file_path as label", () => {
      const result = formatToolSummary("Edit", {
        file_path: "/src/fix.ts",
        old_string: "a",
        new_string: "b",
      });
      expect(result.label).toBe("/src/fix.ts");
    });
  });

  describe("Glob", () => {
    it("shows pattern as label", () => {
      const result = formatToolSummary("Glob", { pattern: "**/*.tsx" });
      expect(result.label).toBe("**/*.tsx");
    });

    it("appends path when provided", () => {
      const result = formatToolSummary("Glob", { pattern: "*.ts", path: "src/" });
      expect(result.label).toBe("*.ts in src/");
    });
  });

  describe("Grep", () => {
    it("wraps pattern in slashes", () => {
      const result = formatToolSummary("Grep", { pattern: "ToolResult" });
      expect(result.label).toBe("/ToolResult/");
    });

    it("appends path when provided", () => {
      const result = formatToolSummary("Grep", { pattern: "foo", path: "lib/" });
      expect(result.label).toBe("/foo/ in lib/");
    });
  });

  describe("Agent", () => {
    it("uses description as label", () => {
      const result = formatToolSummary("Agent", {
        description: "Search codebase",
        prompt: "Find all uses of...",
      });
      expect(result.label).toBe("Search codebase");
      expect(result.detail).toBe("Find all uses of...");
    });

    it("falls back to 'Agent' when no description", () => {
      const result = formatToolSummary("Agent", { prompt: "do stuff" });
      expect(result.label).toBe("Agent");
    });
  });

  describe("WebFetch", () => {
    it("shows URL as label", () => {
      const result = formatToolSummary("WebFetch", { url: "https://example.com" });
      expect(result.label).toBe("https://example.com");
    });
  });

  describe("WebSearch", () => {
    it("shows query as label", () => {
      const result = formatToolSummary("WebSearch", { query: "react hooks" });
      expect(result.label).toBe("react hooks");
    });
  });

  describe("NotebookEdit", () => {
    it("shows notebook path as label", () => {
      const result = formatToolSummary("NotebookEdit", { notebook_path: "analysis.ipynb" });
      expect(result.label).toBe("analysis.ipynb");
    });
  });

  describe("unknown tool", () => {
    it("uses tool name as label and JSON as detail", () => {
      const result = formatToolSummary("CustomTool", { key: "value" });
      expect(result.label).toBe("CustomTool");
      expect(result.detail).toContain('"key": "value"');
    });

    it("has no detail when input is empty", () => {
      const result = formatToolSummary("CustomTool", {});
      expect(result.label).toBe("CustomTool");
      expect(result.detail).toBeUndefined();
    });
  });
});
